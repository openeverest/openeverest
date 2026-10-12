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

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/openeverest/openeverest/v2/pkg/common"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
)

func TestDeleteConfigMap(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	namespace := "test-namespace"

	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))

	tests := []struct {
		name      string
		configMap *corev1.ConfigMap
		err       error
	}{
		{
			name: "managed configmap is deleted",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "managed-configmap",
					Namespace: namespace,
					Labels: map[string]string{
						common.OpenEverestManagedLabel: "true",
					},
				},
			},
		},

		{
			name: "unmanaged configmap is rejected",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "unmanaged-configmap",
					Namespace: namespace,
				},
			},
			err: ErrNotFound,
		},
		{
			name: "configmap blocked by finalizer is pending",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pending-configmap",
					Namespace: namespace,
					Labels: map[string]string{
						common.OpenEverestManagedLabel: "true",
					},
					Finalizers: []string{"openeverest.io/test-finalizer"},
				},
			},
			err: ErrDeletionPending,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.configMap).
				Build()

			handler := &k8sHandler{
				kubeConnector: kubernetes.
					NewEmpty(zap.NewNop().Sugar(), namespace).
					WithKubernetesClient(fakeClient),
				log: zap.NewNop().Sugar(),
			}

			err := handler.DeleteConfigMap(ctx, "", namespace, tt.configMap.Name)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)

				return
			}

			require.NoError(t, err)
		})
	}
}
