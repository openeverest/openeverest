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

package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/yaml"

	"github.com/openeverest/openeverest/v2/internal/secretdefinition"
	rbachandler "github.com/openeverest/openeverest/v2/internal/server/handlers/rbac"
	"github.com/openeverest/openeverest/v2/pkg/oidc"
)

func TestEverestErrorHandler_Conflict(t *testing.T) {
	t.Parallel()

	secretsResource := schema.GroupResource{Resource: "secrets"}

	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "already exists",
			err:  k8serrors.NewAlreadyExists(secretsResource, "my-secret"),
			want: `{
				"message": "secrets \"my-secret\" already exists",
				"reason": "AlreadyExists",
				"field": "metadata.name"
			}`,
		},
		{
			name: "resource version conflict store",
			// https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apiserver/pkg/registry/generic/registry/store.go#L277
			err: k8serrors.NewConflict(secretsResource, "my-secret", errors.New("the object has been modified; please apply your changes to the latest version and try again")),
			want: `{
				"message": "Operation cannot be fulfilled on secrets \"my-secret\": the object has been modified; please apply your changes to the latest version and try again",
				"reason": "Conflict",
				"field": "metadata.resourceVersion"
			}`,
		},
		{
			name: "resource version conflict delete",
			// https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apiserver/pkg/registry/rest/delete.go#L89
			err: k8serrors.NewConflict(secretsResource, "my-secret", errors.New("the ResourceVersion in the precondition (\"330540\") does not match the ResourceVersion in record (\"330541\"). The object might have been modified")),
			want: `{
				"message": "Operation cannot be fulfilled on secrets \"my-secret\": the ResourceVersion in the precondition (\"330540\") does not match the ResourceVersion in record (\"330541\"). The object might have been modified",
				"reason": "Conflict",
				"field": "metadata.resourceVersion"
			}`,
		},
		{
			name: "uid precondition conflict delete",
			// https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apiserver/pkg/registry/rest/delete.go#L86
			err: k8serrors.NewConflict(secretsResource, "my-secret", errors.New("the UID in the precondition (11111111-1111-1111-1111-111111111111) does not match the UID in record (22222222-2222-2222-2222-222222222222). The object might have been deleted and then recreated")),
			want: `{
				"message": "Operation cannot be fulfilled on secrets \"my-secret\": the UID in the precondition (11111111-1111-1111-1111-111111111111) does not match the UID in record (22222222-2222-2222-2222-222222222222). The object might have been deleted and then recreated",
				"reason": "Conflict",
				"field": "metadata.uid"
			}`,
		},
		{
			name: "unknown leaves empty",
			err:  k8serrors.NewConflict(secretsResource, "my-secret", errors.New("something went wrong")),
			want: `{
				"message": "Operation cannot be fulfilled on secrets \"my-secret\": something went wrong",
				"reason": "Conflict"
			}`,
		},
		{
			name: "Secret owned by a controller",
			err:  &secretdefinition.ManagedByControllerError{Kind: "ExternalSecret", Name: "pmm-credentials"},
			want: `{
				"message": "managed by ExternalSecret/pmm-credentials",
				"reason": "ManagedByController"
			}`,
		},
		{
			name: "Secret not created through the API",
			err:  secretdefinition.ErrManagedOutside,
			want: `{
				"message": "managed outside OpenEverest; change it at its source",
				"reason": "ManagedOutside"
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			rec := httptest.NewRecorder()
			c := e.NewContext(httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", nil), rec)

			handler := everestErrorHandler(e.DefaultHTTPErrorHandler)
			handler(tt.err, c)

			require.Equal(t, http.StatusConflict, rec.Code)
			assert.JSONEq(t, tt.want, rec.Body.String())
		})
	}
}

func TestEverestErrorHandler_Forbidden(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "missing permission",
			err:  rbachandler.ErrInsufficientPermissions,
			want: `{"message": "insufficient permissions for performing the operation"}`,
		},
		{
			name: "instance differs from its preset",
			err: fmt.Errorf("create: %w", &rbachandler.PresetMismatchError{
				PresetKind: "ClusterInstancePreset",
				PresetName: "small",
				Fields:     []string{"spec.version"},
				Metadata:   []string{"metadata.labels[team]"},
			}),
			want: `{
				"message": "instance does not match ClusterInstancePreset \"small\": it changes spec.version and sets metadata.labels[team]. instances:create-from-preset allows only an exact copy of the preset; changes need instances:create",
				"reason": "PresetMismatch",
				"causes": [
					{"reason": "FieldValueForbidden", "message": "differs from ClusterInstancePreset \"small\"", "field": "spec.version"},
					{"reason": "FieldValueForbidden", "message": "not allowed on an instance created from a preset", "field": "metadata.labels[team]"}
				]
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			rec := httptest.NewRecorder()
			c := e.NewContext(httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", nil), rec)

			handler := everestErrorHandler(e.DefaultHTTPErrorHandler)
			handler(tt.err, c)

			require.Equal(t, http.StatusForbidden, rec.Code)
			assert.JSONEq(t, tt.want, rec.Body.String())
		})
	}
}

func TestNewSSOTokenVerifier(t *testing.T) {
	t.Parallel()

	provider := func(clientID string) *oidc.ProviderConfig {
		return &oidc.ProviderConfig{
			Issuer:   "https://idp.example.com",
			JWKSURL:  "https://idp.example.com/jwks",
			ClientID: clientID,
		}
	}

	t.Run("OIDC not configured disables SSO", func(t *testing.T) {
		t.Parallel()
		assert.Nil(t, newSSOTokenVerifier(t.Context(), zap.NewNop().Sugar(), nil))
	})

	t.Run("invalid OIDC settings disable SSO instead of failing the server", func(t *testing.T) {
		t.Parallel()
		core, logs := observer.New(zap.ErrorLevel)
		assert.Nil(t, newSSOTokenVerifier(t.Context(), zap.New(core).Sugar(), provider("")))
		assert.Equal(t, 1, logs.FilterMessageSnippet("SSO is disabled").Len())
	})

	t.Run("valid OIDC settings enable SSO", func(t *testing.T) {
		t.Parallel()
		assert.NotNil(t, newSSOTokenVerifier(t.Context(), zap.NewNop().Sugar(), provider("everest")))
	})
}

func TestEverestErrorHandler_AdmissionDenial(t *testing.T) {
	t.Parallel()

	destinations := schema.GroupResource{Group: "monitoring.openeverest.io", Resource: "monitoringdestinations"}
	inUse := "in use by 1 database; remove the destination from them first " +
		"(kubectl get monitoringbindings -n team-a -l monitoring.openeverest.io/destination=pmm-prod)"
	secrets := schema.GroupResource{Resource: "secrets"}
	secretInUse := "in use by 2 monitoring destination(s) and 0 backup storage(s); replace its values instead, " +
		"or point them at another Secret first (kubectl get monitoringdestinations,backupstorages -n team-a " +
		"--field-selector spec.tls.ca.secretKeyRef.name=pmm-ca)"

	tests := []struct {
		name       string
		err        error
		code       int
		want       string
		wantReason string
	}{
		{
			name: "in-use policy denial is a conflict",
			// admission.NewForbidden in the ValidatingAdmissionPolicy dispatcher of k8s.io/apiserver.
			err: k8serrors.NewForbidden(destinations, "pmm-prod", errors.New(
				"ValidatingAdmissionPolicy 'monitoringdestination-in-use.monitoring.openeverest.io' with binding "+
					"'monitoringdestination-in-use.monitoring.openeverest.io' denied request: "+inUse,
			)),
			code:       http.StatusConflict,
			want:       inUse,
			wantReason: "InUse",
		},
		{
			name: "in-use policy named with kustomize's prefix",
			err: k8serrors.NewForbidden(destinations, "pmm-prod", errors.New(
				"ValidatingAdmissionPolicy 'openeverest-monitoringdestination-in-use.monitoring.openeverest.io' with binding "+
					"'openeverest-monitoringdestination-in-use.monitoring.openeverest.io' denied request: "+inUse,
			)),
			code:       http.StatusConflict,
			want:       inUse,
			wantReason: "InUse",
		},
		{
			name: "Secret in-use policy denial is a conflict",
			err: k8serrors.NewForbidden(secrets, "pmm-ca", errors.New(
				"ValidatingAdmissionPolicy 'secret-in-use.openeverest.io' with binding "+
					"'secret-in-use.openeverest.io' denied request: "+secretInUse,
			)),
			code:       http.StatusConflict,
			want:       secretInUse,
			wantReason: "InUse",
		},
		{
			name: "Secret in-use policy named with kustomize's prefix",
			err: k8serrors.NewForbidden(secrets, "pmm-ca", errors.New(
				"ValidatingAdmissionPolicy 'openeverest-secret-in-use.openeverest.io' with binding "+
					"'openeverest-secret-in-use.openeverest.io' denied request: "+secretInUse,
			)),
			code:       http.StatusConflict,
			want:       secretInUse,
			wantReason: "InUse",
		},
		{
			name: "other policy denial stays forbidden",
			err: k8serrors.NewForbidden(destinations, "pmm-prod", errors.New(
				"ValidatingAdmissionPolicy 'destination-naming.example.com' denied request: names must start with team-",
			)),
			code: http.StatusForbidden,
			want: "names must start with team-",
		},
		{
			// A missing permission, not a state the caller can wait out.
			name: "secret-reference policy denial stays forbidden",
			err: k8serrors.NewForbidden(destinations, "pmm-prod", errors.New(
				"ValidatingAdmissionPolicy 'openeverest-secret-reference.openeverest.io' with binding "+
					"'openeverest-secret-reference.openeverest.io' denied request: "+
					"you need get on Secret pmm-prod-credentials to reference it or to change an object that uses it",
			)),
			code: http.StatusForbidden,
			want: "you need get on Secret pmm-prod-credentials to reference it or to change an object that uses it",
		},
		{
			name: "webhook denial",
			// webhookerrors.ToStatusErr in k8s.io/apiserver.
			err: &k8serrors.StatusError{ErrStatus: metav1.Status{
				Status:  metav1.StatusFailure,
				Code:    http.StatusForbidden,
				Reason:  metav1.StatusReasonForbidden,
				Message: `admission webhook "vinstance.percona-server-mongodb.openeverest.io" denied the request: replicas must be odd`,
			}},
			code: http.StatusForbidden,
			want: "replicas must be odd",
		},
		{
			name: "other forbidden error unchanged",
			err:  k8serrors.NewForbidden(destinations, "pmm-prod", errors.New("namespace team-a is being terminated")),
			code: http.StatusForbidden,
			want: `monitoringdestinations.monitoring.openeverest.io "pmm-prod" is forbidden: namespace team-a is being terminated`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			rec := httptest.NewRecorder()
			c := e.NewContext(httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/", nil), rec)

			handler := everestErrorHandler(e.DefaultHTTPErrorHandler)
			handler(tt.err, c)

			require.Equal(t, tt.code, rec.Code)
			var body struct {
				Message string `json:"message"`
				Reason  string `json:"reason"`
			}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			assert.Equal(t, tt.want, body.Message)
			assert.Equal(t, tt.wantReason, body.Reason)
		})
	}
}

// TestInUsePolicyCoversPolicies proves every in-use policy core ships
// answers 409 InUse over HTTP, named as in the chart or with the prefix
// kustomize's config/default adds.
func TestInUsePolicyCoversPolicies(t *testing.T) {
	t.Parallel()

	files, err := filepath.Glob(filepath.Join("..", "..", "config", "policy", "*_in_use.yaml"))
	require.NoError(t, err)
	var names []string
	for _, file := range files {
		content, err := os.ReadFile(file) //nolint:gosec // the repository's own policy files
		require.NoError(t, err)
		for _, doc := range regexp.MustCompile(`(?m)^---\s*$`).Split(string(content), -1) {
			obj := &metav1.PartialObjectMetadata{}
			require.NoError(t, yaml.Unmarshal([]byte(doc), obj), file)
			if obj.Kind == "ValidatingAdmissionPolicy" {
				names = append(names, obj.Name)
			}
		}
	}
	require.Contains(t, names, "secret-in-use.openeverest.io")
	require.Contains(t, names, "monitoringdestination-in-use.monitoring.openeverest.io")
	for _, name := range names {
		for _, deployed := range []string{name, "openeverest-" + name} {
			assert.True(t, inUsePolicy.MatchString("ValidatingAdmissionPolicy '"+deployed+"' with binding '"+deployed+"' denied request: in use"),
				deployed,
			)
		}
	}
}

func TestEverestErrorHandler_Invalid(t *testing.T) {
	t.Parallel()

	instanceKind := schema.GroupKind{Group: "core.openeverest.io", Kind: "Instance"}

	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "every field is a cause",
			err: k8serrors.NewInvalid(instanceKind, "my-db", field.ErrorList{
				field.Invalid(field.NewPath("spec", "components").Key("hidden").Child("replicas"), 1, "replicas in body should be a multiple of 2"),
				field.Required(field.NewPath("spec", "topology", "type"), ""),
			}),
			want: `{
				"message": "Instance.core.openeverest.io \"my-db\" is invalid: [spec.components[hidden].replicas: Invalid value: 1: replicas in body should be a multiple of 2, spec.topology.type: Required value]",
				"reason": "Invalid",
				"causes": [
					{
						"reason": "FieldValueInvalid",
						"message": "Invalid value: 1: replicas in body should be a multiple of 2",
						"field": "spec.components[hidden].replicas"
					},
					{
						"reason": "FieldValueRequired",
						"message": "Required value",
						"field": "spec.topology.type"
					}
				]
			}`,
		},
		{
			name: "strict decoding drops the object",
			err: k8serrors.NewInvalid(schema.GroupKind{}, "", field.ErrorList{
				field.Invalid(field.NewPath("patch"), `{"spec":{"replicaz":3}}`, `strict decoding error: unknown field "spec.replicaz"`),
			}),
			want: `{
				"message": "strict decoding error: unknown field \"spec.replicaz\"",
				"reason": "Invalid",
				"causes": [
					{
						"reason": "FieldValueInvalid",
						"message": "strict decoding error: unknown field \"spec.replicaz\"",
						"field": "patch"
					}
				]
			}`,
		},
		{
			name: "no details",
			err: &k8serrors.StatusError{ErrStatus: metav1.Status{
				Status:  metav1.StatusFailure,
				Code:    http.StatusUnprocessableEntity,
				Reason:  metav1.StatusReasonInvalid,
				Message: "the object is invalid",
			}},
			want: `{
				"message": "the object is invalid",
				"reason": "Invalid"
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			rec := httptest.NewRecorder()
			c := e.NewContext(httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", nil), rec)

			handler := everestErrorHandler(e.DefaultHTTPErrorHandler)
			handler(tt.err, c)

			require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
			assert.JSONEq(t, tt.want, rec.Body.String())
		})
	}
}
