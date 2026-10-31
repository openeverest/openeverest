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

package k8s

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	api "github.com/openeverest/openeverest/v2/internal/server/api"
	"github.com/openeverest/openeverest/v2/pkg/common"
	"github.com/openeverest/openeverest/v2/pkg/events"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
)

// TestK8s_PatchInstance pins the merge semantics. The fake client does no
// strict decoding, so fieldValidation is not observable here and is covered by
// api-tests/tests/instance.spec.ts instead.
func TestK8s_PatchInstance(t *testing.T) {
	t.Parallel()

	replicas := int32(1)
	seed := func() *corev1alpha1.Instance {
		return &corev1alpha1.Instance{
			ObjectMeta: metav1.ObjectMeta{
				Name: "db1", Namespace: "ns1",
				Annotations: map[string]string{"team.example.com/owner": "platform"},
			},
			Spec: corev1alpha1.InstanceSpec{
				Version: "8.0",
				Components: map[string]corev1alpha1.ComponentSpec{
					"engine": {Type: "mongod", Replicas: &replicas},
					"proxy":  {Type: "mongos", Version: "1.2"},
				},
			},
		}
	}

	newHandler := func() *k8sHandler {
		fakeClient := fake.NewClientBuilder().
			WithScheme(kubernetes.CreateScheme()).
			WithObjects(seed()).
			Build()
		return &k8sHandler{
			kubeConnector: kubernetes.NewEmpty(zap.NewNop().Sugar(), "ns1").WithKubernetesClient(fakeClient),
			log:           zap.NewNop().Sugar(),
		}
	}

	ctx := context.Background()

	t.Run("named member changes and the rest survives", func(t *testing.T) {
		t.Parallel()

		result, err := newHandler().PatchInstance(ctx, "prod", "ns1", "db1",
			[]byte(`{"spec":{"components":{"engine":{"replicas":5}}}}`))
		require.NoError(t, err)

		require.NotNil(t, result.Spec.Components["engine"].Replicas)
		assert.Equal(t, int32(5), *result.Spec.Components["engine"].Replicas)

		// Anything the patch did not name keeps its stored value.
		assert.Equal(t, "8.0", result.Spec.Version)
		assert.Equal(t, "1.2", result.Spec.Components["proxy"].Version)
		assert.Equal(t, "mongod", result.Spec.Components["engine"].Type)
	})

	t.Run("stamps the calling actor", func(t *testing.T) {
		t.Parallel()

		userCtx := context.WithValue(context.Background(), common.UserCtxKey, &jwt.Token{ //nolint:staticcheck
			Claims: jwt.MapClaims{"sub": "bob", "iss": "everest"},
		})
		result, err := newHandler().PatchInstance(userCtx, "prod", "ns1", "db1",
			[]byte(`{"spec":{"version":"8.1"}}`))
		require.NoError(t, err)

		assert.Equal(t, "user", result.Annotations[events.AnnotationActorType])
		assert.Equal(t, "bob", result.Annotations[events.AnnotationActorID])
	})

	t.Run("no JWT leaves no empty stamp", func(t *testing.T) {
		t.Parallel()

		result, err := newHandler().PatchInstance(ctx, "prod", "ns1", "db1",
			[]byte(`{"spec":{"version":"8.1"}}`))
		require.NoError(t, err)

		assert.NotContains(t, result.Annotations, events.AnnotationActorType)
	})

	t.Run("a single annotation can be removed through the stamp merge", func(t *testing.T) {
		t.Parallel()

		userCtx := context.WithValue(context.Background(), common.UserCtxKey, &jwt.Token{ //nolint:staticcheck
			Claims: jwt.MapClaims{"sub": "bob", "iss": "everest"},
		})
		result, err := newHandler().PatchInstance(userCtx, "prod", "ns1", "db1",
			[]byte(`{"metadata":{"annotations":{"team.example.com/owner":null}}}`))
		require.NoError(t, err)

		assert.NotContains(t, result.Annotations, "team.example.com/owner")
		assert.Equal(t, "bob", result.Annotations[events.AnnotationActorID])
	})

	t.Run("null removes a member", func(t *testing.T) {
		t.Parallel()

		result, err := newHandler().PatchInstance(ctx, "prod", "ns1", "db1",
			[]byte(`{"spec":{"components":{"engine":{"replicas":null}}}}`))
		require.NoError(t, err)

		assert.Nil(t, result.Spec.Components["engine"].Replicas)
		assert.Equal(t, "mongod", result.Spec.Components["engine"].Type)
	})
}

func TestK8s_DeleteInstance(t *testing.T) {
	t.Parallel()

	scheme := kubernetes.CreateScheme()
	ctx := context.Background()

	t.Run("live instance with policy change updates policy and deletes", func(t *testing.T) {
		t.Parallel()

		inst := &corev1alpha1.Instance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "db-live",
				Namespace: "ns1",
			},
			Spec: corev1alpha1.InstanceSpec{
				DeletionPolicy: corev1alpha1.InstanceDeletionPolicyCascade,
			},
		}

		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(inst).
			Build()

		h := &k8sHandler{
			kubeConnector: kubernetes.NewEmpty(zap.NewNop().Sugar(), "ns1").WithKubernetesClient(fakeClient),
			log:           zap.NewNop().Sugar(),
		}

		policyOrphan := string(corev1alpha1.InstanceDeletionPolicyOrphan)
		params := &api.DeleteInstanceParams{
			DeletionPolicy: (*api.DeleteInstanceParamsDeletionPolicy)(&policyOrphan),
		}

		err := h.DeleteInstance(ctx, "main", "ns1", "db-live", params)
		require.NoError(t, err)

		// Verify the instance is deleted
		_, err = h.kubeConnector.GetInstance(ctx, types.NamespacedName{Namespace: "ns1", Name: "db-live"})
		require.Error(t, err)
		assert.True(t, k8serrors.IsNotFound(err))
	})

	t.Run("terminating instance with same policy proceeds without error", func(t *testing.T) {
		t.Parallel()

		now := metav1.Now()
		inst := &corev1alpha1.Instance{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "db-terminating-same",
				Namespace:         "ns1",
				DeletionTimestamp: &now,
				Finalizers:        []string{"test-finalizer"},
			},
			Spec: corev1alpha1.InstanceSpec{
				DeletionPolicy: corev1alpha1.InstanceDeletionPolicyCascade,
			},
		}

		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(inst).
			Build()

		h := &k8sHandler{
			kubeConnector: kubernetes.NewEmpty(zap.NewNop().Sugar(), "ns1").WithKubernetesClient(fakeClient),
			log:           zap.NewNop().Sugar(),
		}

		policyCascade := string(corev1alpha1.InstanceDeletionPolicyCascade)
		params := &api.DeleteInstanceParams{
			DeletionPolicy: (*api.DeleteInstanceParamsDeletionPolicy)(&policyCascade),
		}

		err := h.DeleteInstance(ctx, "main", "ns1", "db-terminating-same", params)
		require.NoError(t, err)
	})

	t.Run("terminating instance with conflicting policy returns HTTP 409 Conflict", func(t *testing.T) {
		t.Parallel()

		now := metav1.Now()
		inst := &corev1alpha1.Instance{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "db-terminating-conflict",
				Namespace:         "ns1",
				DeletionTimestamp: &now,
				Finalizers:        []string{"test-finalizer"},
			},
			Spec: corev1alpha1.InstanceSpec{
				DeletionPolicy: corev1alpha1.InstanceDeletionPolicyCascade,
			},
		}

		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(inst).
			Build()

		h := &k8sHandler{
			kubeConnector: kubernetes.NewEmpty(zap.NewNop().Sugar(), "ns1").WithKubernetesClient(fakeClient),
			log:           zap.NewNop().Sugar(),
		}

		policyOrphan := string(corev1alpha1.InstanceDeletionPolicyOrphan)
		params := &api.DeleteInstanceParams{
			DeletionPolicy: (*api.DeleteInstanceParamsDeletionPolicy)(&policyOrphan),
		}

		err := h.DeleteInstance(ctx, "main", "ns1", "db-terminating-conflict", params)
		require.Error(t, err)

		var httpErr *echo.HTTPError
		require.True(t, errors.As(err, &httpErr), "expected *echo.HTTPError, got %T: %v", err, err)
		assert.Equal(t, http.StatusConflict, httpErr.Code)
		assert.Contains(t, httpErr.Message, "cannot change deletion policy to \"Orphan\"")
		assert.Contains(t, httpErr.Message, "already terminating with policy \"Cascade\"")

		// Verify the stored instance's deletionPolicy was NOT modified
		fetched, getErr := h.kubeConnector.GetInstance(ctx, types.NamespacedName{Namespace: "ns1", Name: "db-terminating-conflict"})
		require.NoError(t, getErr)
		assert.Equal(t, corev1alpha1.InstanceDeletionPolicyCascade, fetched.Spec.DeletionPolicy)
	})
}
