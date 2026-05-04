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

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"
	"github.com/percona/everest/pkg/common"
	"github.com/percona/everest/pkg/kubernetes"
	"github.com/percona/everest/pkg/session"
)

type want struct {
	err    string
	status int
	next   bool
}

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
				return echo.New().NewContext(&http.Request{
					Method: http.MethodGet,
				}, nil,
				)
			},
			allow: true,
		},
		{
			description: "allow non-target paths",
			ctxFn: func() echo.Context {
				return echo.New().NewContext(&http.Request{
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
				return echo.New().NewContext(&http.Request{
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
				ctx := echo.New().NewContext(&http.Request{
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
				ctx := echo.New().NewContext(&http.Request{
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
			k := kubernetes.NewEmpty(zap.NewNop().Sugar()).WithKubernetesClient(mockClient.Build())
			e := EverestServer{kubeConnector: k}
			ctx := tc.ctxFn()

			allow, err := e.shouldAllowRequestDuringEngineUpgrade(ctx)
			require.NoError(t, err)
			assert.Equal(t, tc.allow, allow)
		})
	}
}

func TestValidateIfPasswordChangeIsRequired(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		description string
		ctxFn       func() echo.Context
		want        want
	}{
		{
			description: "allow allowlisted endpoint without token",
			ctxFn: func() echo.Context {
				e := echo.New()
				ctx := e.NewContext(&http.Request{
					Method: http.MethodGet,
					URL: &url.URL{
						Path: "/v1/accounts",
					},
				}, nil)
				ctx.SetPath("/v1/accounts")
				return ctx
			},
			want: want{err: "", status: 0, next: true},
		},
		{
			description: "deny non allowlisted endpoint if token is missing",
			ctxFn: func() echo.Context {
				e := echo.New()
				ctx := e.NewContext(&http.Request{
					Method: http.MethodGet,
					URL: &url.URL{
						Path: "/v1/database-clusters",
					},
				}, nil)
				ctx.SetPath("/v1/database-clusters")
				return ctx
			},
			want: want{
				err:  "failed to get token from context",
				next: false,
			},
		},
		{
			description: "return forbidden if token claims are not map claims",
			ctxFn: func() echo.Context {
				e := echo.New()
				req := &http.Request{
					Method: http.MethodGet,
					URL: &url.URL{
						Path: "/v1/database-clusters",
					},
				}
				req = req.WithContext(context.WithValue(req.Context(), common.UserCtxKey, &jwt.Token{
					Claims: &session.EverestClaims{},
				}))
				ctx := e.NewContext(req, httptest.NewRecorder())
				ctx.SetPath("/v1/database-clusters")
				return ctx
			},
			want: want{
				status: http.StatusForbidden,
				next:   false,
			},
		},
		{
			description: "deny non allowlisted endpoint if claim type is invalid",
			ctxFn: func() echo.Context {
				e := echo.New()
				req := &http.Request{
					Method: http.MethodGet,
					URL: &url.URL{
						Path: "/v1/database-clusters",
					},
				}
				req = req.WithContext(context.WithValue(req.Context(), common.UserCtxKey, &jwt.Token{
					Claims: jwt.MapClaims{
						session.MustChangePasswordClaim: "true",
					},
				}))
				ctx := e.NewContext(req, nil)
				ctx.SetPath("/v1/database-clusters")
				return ctx
			},
			want: want{
				err:  "failed to parse claim from token",
				next: false,
			},
		},
		{
			description: "allow non allowlisted endpoint with valid claim",
			ctxFn: func() echo.Context {
				e := echo.New()
				req := &http.Request{
					Method: http.MethodGet,
					URL: &url.URL{
						Path: "/v1/database-clusters",
					},
				}
				req = req.WithContext(context.WithValue(req.Context(), common.UserCtxKey, &jwt.Token{
					Claims: jwt.MapClaims{
						session.MustChangePasswordClaim: true,
					},
				}))
				ctx := e.NewContext(req, nil)
				ctx.SetPath("/v1/database-clusters")
				return ctx
			},
			want: want{
				next: true,
			},
		},
		{
			description: "allow non allowlisted endpoint when claim is false",
			ctxFn: func() echo.Context {
				e := echo.New()
				req := &http.Request{
					Method: http.MethodGet,
					URL: &url.URL{
						Path: "/v1/database-clusters",
					},
				}
				req = req.WithContext(context.WithValue(req.Context(), common.UserCtxKey, &jwt.Token{
					Claims: jwt.MapClaims{
						session.MustChangePasswordClaim: false,
					},
				}))
				ctx := e.NewContext(req, nil)
				ctx.SetPath("/v1/database-clusters")
				return ctx
			},
			want: want{
				next: true,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()
			mockClient := fakeclient.NewClientBuilder().WithScheme(kubernetes.CreateScheme())
			k := kubernetes.NewEmpty(zap.NewNop().Sugar()).WithKubernetesClient(mockClient.Build())
			e := EverestServer{kubeConnector: k}
			ctx := tc.ctxFn()
			called := false
			next := func(c echo.Context) error {
				called = true
				return nil
			}
			err := e.validateIfPasswordChangeIsRequired(next)(ctx)
			if tc.want.err != "" {
				require.ErrorContains(t, err, tc.want.err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tc.want.next, called)
			if tc.want.status != 0 {
				assert.Equal(t, tc.want.status, ctx.Response().Status)
			}
		})
	}
}
