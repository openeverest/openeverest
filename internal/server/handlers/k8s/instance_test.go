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
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
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
					"engine": {Name: "engine", Replicas: &replicas},
					"proxy":  {Name: "proxy", Version: "1.2"},
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
		assert.Equal(t, "engine", result.Spec.Components["engine"].Name)
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
		assert.Equal(t, "engine", result.Spec.Components["engine"].Name)
	})
}
