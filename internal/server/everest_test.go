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
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestEverestErrorHandler_Conflict(t *testing.T) {
	t.Parallel()

	e := echo.New()

	t.Run("AlreadyExists with StatusError", func(t *testing.T) {
		t.Parallel()

		var capturedErr error
		handler := everestErrorHandler(func(err error, _ echo.Context) {
			capturedErr = err
		})

		req := httptest.NewRequest(http.MethodPost, "/v1/clusters/default/namespaces/default/instances", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		k8sErr := k8serrors.NewAlreadyExists(schema.GroupResource{Resource: "instances"}, "test-db")
		handler(k8sErr, c)

		require.Error(t, capturedErr)
		var httpErr *echo.HTTPError
		require.True(t, errors.As(capturedErr, &httpErr))
		assert.Equal(t, http.StatusConflict, httpErr.Code)
		assert.Contains(t, httpErr.Message, "already exists")
	})

	t.Run("Conflict with StatusError message", func(t *testing.T) {
		t.Parallel()

		var capturedErr error
		handler := everestErrorHandler(func(err error, _ echo.Context) {
			capturedErr = err
		})

		req := httptest.NewRequest(http.MethodPut, "/v1/clusters/default/namespaces/default/instances/test-db", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		statusErr := &k8serrors.StatusError{
			ErrStatus: metav1.Status{
				Reason:  metav1.StatusReasonConflict,
				Message: "Operation cannot be fulfilled on instances: custom conflict message",
			},
		}

		handler(statusErr, c)

		require.Error(t, capturedErr)
		var httpErr *echo.HTTPError
		require.True(t, errors.As(capturedErr, &httpErr))
		assert.Equal(t, http.StatusConflict, httpErr.Code)
		assert.Equal(t, "Operation cannot be fulfilled on instances: custom conflict message", httpErr.Message)
	})

	t.Run("Conflict without StatusError message uses fallback message", func(t *testing.T) {
		t.Parallel()

		var capturedErr error
		handler := everestErrorHandler(func(err error, _ echo.Context) {
			capturedErr = err
		})

		req := httptest.NewRequest(http.MethodPost, "/v1/clusters/default/namespaces/default/instances", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		statusErr := &k8serrors.StatusError{
			ErrStatus: metav1.Status{
				Reason: metav1.StatusReasonConflict,
			},
		}
		handler(statusErr, c)

		require.Error(t, capturedErr)
		var httpErr *echo.HTTPError
		require.True(t, errors.As(capturedErr, &httpErr))
		assert.Equal(t, http.StatusConflict, httpErr.Code)
		assert.Equal(t, "resource already exists or conflicts with existing state", httpErr.Message)
	})
}
