// Copyright (C) 2026 The OpenEverest Contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/rest"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	api "github.com/openeverest/openeverest/v2/internal/server/api"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
)

func TestShouldAllowRequestDuringEngineUpgrade(t *testing.T) {
	lockedAt := time.Now().Format(time.RFC3339)
	t.Parallel()
	testCases := []struct {
		description string
		objs        []ctrlclient.Object
		ctxFn       func() echo.Context
		allow       bool
	}{
		{
			description: "allow all GET requests",
			ctxFn: func() echo.Context {
				return echo.New().NewContext(
					&http.Request{
						Method: http.MethodGet,
					}, nil,
				)
			},
			allow: true,
		},
		{
			description: "allow non-target paths",
			ctxFn: func() echo.Context {
				return echo.New().NewContext(
					&http.Request{
						Method: http.MethodPost,
						URL: &url.URL{
							Path: "/api/v1/namespaces/default/monitoring-instances",
						},
					}, nil,
				)
			},
			allow: true,
		},
		{
			description: "allow target paths with no namespace",
			ctxFn: func() echo.Context {
				return echo.New().NewContext(
					&http.Request{
						Method: http.MethodPost,
						URL: &url.URL{
							Path: "/api/v1/database-clusters",
						},
					}, nil,
				)
			},
			allow: true,
		},
		{
			description: "allow target path with no lock annotation",
			ctxFn: func() echo.Context {
				ctx := echo.New().NewContext(
					&http.Request{
						Method: http.MethodDelete,
						URL: &url.URL{
							Path: "/api/v1/namespaces/default/database-clusters/1234",
						},
					}, nil,
				)
				ctx.SetParamNames("namespace")
				ctx.SetParamValues("default")
				return ctx
			},
			objs: []ctrlclient.Object{
				&everestv1alpha1.DatabaseEngine{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-engine",
						Namespace: "default",
					},
				},
			},
			allow: true,
		},
		{
			description: "deny request on target path with lock annotation",
			ctxFn: func() echo.Context {
				ctx := echo.New().NewContext(
					&http.Request{
						Method: http.MethodDelete,
						URL: &url.URL{
							Path: "/api/v1/namespaces/default/database-clusters/1234",
						},
					}, nil,
				)
				ctx.SetParamNames("namespace")
				ctx.SetParamValues("default")
				return ctx
			},
			objs: []ctrlclient.Object{
				&everestv1alpha1.DatabaseEngine{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-engine",
						Namespace: "default",
						Annotations: map[string]string{
							everestv1alpha1.DatabaseOperatorUpgradeLockAnnotation: lockedAt,
						},
					},
				},
			},
			allow: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()
			mockClient := fakeclient.NewClientBuilder().WithScheme(kubernetes.CreateScheme()).WithObjects(tc.objs...)
			k := kubernetes.NewEmpty(zap.NewNop().Sugar(), "test-ns").WithKubernetesClient(mockClient.Build())
			e := EverestServer{kubeConnector: k}
			ctx := tc.ctxFn()

			allow, err := e.shouldAllowRequestDuringEngineUpgrade(ctx)
			require.NoError(t, err)
			assert.Equal(t, tc.allow, allow)
		})
	}
}

func TestKubernetesWarnings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		method      string
		warnings    []string
		err         error
		wantCode    int
		wantHeaders []string
	}{
		{
			name:   "admitted write",
			method: http.MethodPost,
			warnings: []string{
				`secret "db-creds" named by spec.userSecretRef not found`,
				"backup class nightly not found",
				`secret "db-creds" named by spec.userSecretRef not found`,
			},
			wantCode: http.StatusCreated,
			wantHeaders: []string{
				`299 - "secret \"db-creds\" named by spec.userSecretRef not found"`,
				`299 - "backup class nightly not found"`,
			},
		},
		{
			name:     "denied write",
			method:   http.MethodPatch,
			warnings: []string{"backup class nightly not found"},
			err: k8serrors.NewInvalid(
				schema.GroupKind{Group: "core.openeverest.io", Kind: "Instance"}, "my-db",
				field.ErrorList{field.Required(field.NewPath("spec", "topology", "type"), "")},
			),
			wantCode:    http.StatusUnprocessableEntity,
			wantHeaders: []string{`299 - "backup class nightly not found"`},
		},
		{
			name:     "write without warnings",
			method:   http.MethodDelete,
			wantCode: http.StatusCreated,
		},
		{
			name:     "text a header cannot carry",
			method:   http.MethodPut,
			warnings: []string{"line one\nline two", "backup class nightly not found"},
			wantCode: http.StatusCreated,
			wantHeaders: []string{
				`299 - "backup class nightly not found"`,
			},
		},
		{
			name:     "read",
			method:   http.MethodGet,
			warnings: []string{"backup class nightly not found"},
			wantCode: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			warningHandler := kubernetes.CollectingWarningHandler{Next: rest.NoWarnings{}}
			handler := func(c echo.Context) error {
				for _, text := range tt.warnings {
					warningHandler.HandleWarningHeaderWithContext(c.Request().Context(), 299, "-", text)
				}
				if tt.err != nil {
					return tt.err
				}
				return c.JSON(http.StatusCreated, struct{}{})
			}
			e := echo.New()
			e.HTTPErrorHandler = everestErrorHandler(e.DefaultHTTPErrorHandler)
			e.Any("/", handler, kubernetesWarnings)

			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, httptest.NewRequestWithContext(context.Background(), tt.method, "/", nil))

			assert.Equal(t, tt.wantCode, rec.Code)
			assert.Equal(t, tt.wantHeaders, rec.Header().Values("Warning"))
		})
	}
}

func TestDryRun(t *testing.T) {
	t.Parallel()

	swagger, err := api.GetSpec()
	require.NoError(t, err)

	tests := []struct {
		name       string
		method     string
		target     string
		wantCode   int
		wantDryRun bool
	}{
		{
			name:       "operation declaring dryRun",
			method:     http.MethodPost,
			target:     "/v1/clusters/main/namespaces/ns/instances?dryRun=All",
			wantCode:   http.StatusOK,
			wantDryRun: true,
		},
		{
			name:     "operation declaring dryRun, called without it",
			method:   http.MethodPatch,
			target:   "/v1/clusters/main/namespaces/ns/instances/db",
			wantCode: http.StatusOK,
		},
		{
			name:     "operation not declaring dryRun",
			method:   http.MethodDelete,
			target:   "/v1/clusters/main/namespaces/ns/instances/db?dryRun=All",
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var gotDryRun *bool
			handler := func(c echo.Context) error {
				gotDryRun = new(kubernetes.IsDryRun(c.Request().Context()))
				return c.NoContent(http.StatusOK)
			}
			e := echo.New()
			g := e.Group("/v1", dryRun(swagger, "/v1"))
			g.POST("/clusters/:cluster/namespaces/:namespace/instances", handler)
			g.PATCH("/clusters/:cluster/namespaces/:namespace/instances/:instance", handler)
			g.DELETE("/clusters/:cluster/namespaces/:namespace/instances/:instance", handler)

			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, httptest.NewRequestWithContext(context.Background(), tt.method, tt.target, nil))

			assert.Equal(t, tt.wantCode, rec.Code)
			if tt.wantCode != http.StatusOK {
				assert.Nil(t, gotDryRun, "the handler must not run")
				return
			}
			require.NotNil(t, gotDryRun)
			assert.Equal(t, tt.wantDryRun, *gotDryRun)
		})
	}
}
