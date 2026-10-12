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
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
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
