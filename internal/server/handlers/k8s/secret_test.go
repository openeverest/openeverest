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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/openeverest/openeverest/v2/pkg/common"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
)

func TestStripSecretData(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	namespace := "test-namespace"

	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))

	tests := []struct {
		name   string
		secret *corev1.Secret
	}{
		{
			name: "secret with stringData",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "secret-stringdata",
					Labels: map[string]string{
						common.OpenEverestCategoryLabel: "backup-storage",
					},
				},
				Type: corev1.SecretTypeOpaque,
				StringData: map[string]string{
					"accessKey": "myaccesskey",
					"secretKey": "mysecretkey",
				},
			},
		},
		{
			name: "secret with data",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "secret-data",
					Labels: map[string]string{
						common.OpenEverestCategoryLabel: "database-credentials",
					},
				},
				Type: corev1.SecretTypeOpaque,
				Data: map[string][]byte{
					"username": []byte("admin"),
					"password": []byte("secret123"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run("CreateSecret_"+tt.name, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

			handler := &k8sHandler{
				kubeConnector: kubernetes.NewEmpty(zap.NewNop().Sugar(), namespace).WithKubernetesClient(fakeClient),
				log:           zap.NewNop().Sugar(),
			}

			result, err := handler.CreateSecret(ctx, "", namespace, tt.secret)
			require.NoError(t, err)
			assert.Nil(t, result.Data)
			assert.Nil(t, result.StringData)
		})

		t.Run("GetSecret_"+tt.name, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tt.secret).Build()

			handler := &k8sHandler{
				kubeConnector: kubernetes.NewEmpty(zap.NewNop().Sugar(), namespace).WithKubernetesClient(fakeClient),
				log:           zap.NewNop().Sugar(),
			}

			result, err := handler.GetSecret(ctx, "", namespace, tt.secret.Name)
			require.NoError(t, err)
			assert.Nil(t, result.Data)
			assert.Nil(t, result.StringData)
		})

		t.Run("ListSecrets_"+tt.name, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithLists(&corev1.SecretList{Items: []corev1.Secret{*tt.secret}}).
				Build()

			handler := &k8sHandler{
				kubeConnector: kubernetes.NewEmpty(zap.NewNop().Sugar(), namespace).WithKubernetesClient(fakeClient),
				log:           zap.NewNop().Sugar(),
			}

			result, err := handler.ListSecrets(ctx, "", namespace, "", "")
			require.NoError(t, err)
			assert.Len(t, result.Items, 1, "Should have expected number of secrets")
			assert.Nil(t, result.Items[0].Data)
			assert.Nil(t, result.Items[0].StringData)
		})
	}
}
