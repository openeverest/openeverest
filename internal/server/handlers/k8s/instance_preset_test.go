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
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
)

func newTestPreset(components map[string]corev1alpha1.ComponentSpec) *corev1alpha1.InstancePreset {
	return &corev1alpha1.InstancePreset{
		ObjectMeta: metav1.ObjectMeta{Name: "test"},
		Spec: corev1alpha1.InstancePresetSpec{
			InstanceSpec: corev1alpha1.InstanceSpec{Components: components},
		},
	}
}

func TestApplyNamespaceDefaults_New(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	namespace := "test-namespace"

	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, monitoringv1alpha1.AddToScheme(scheme))

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "default-secret",
					Namespace:   namespace,
					Annotations: map[string]string{"openeverest.io/is-default-components-pmm": "true"},
				},
			},
			&monitoringv1alpha1.MonitoringConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "default-monitoring",
					Namespace:   namespace,
					Annotations: map[string]string{"openeverest.io/is-default-components-pmm": "true"},
				},
				Spec: monitoringv1alpha1.MonitoringConfigSpec{Type: "pmm"},
			},
		).
		Build()

	handler := &k8sHandler{
		kubeConnector: kubernetes.NewEmpty(zap.NewNop().Sugar(), namespace).WithKubernetesClient(fakeClient),
		log:           zap.NewNop().Sugar(),
	}

	emptySecretRef := &corev1alpha1.Config{SecretRef: corev1.LocalObjectReference{Name: ""}}
	resolvedSecretRef := &corev1alpha1.Config{SecretRef: corev1.LocalObjectReference{Name: "default-secret"}}

	tests := []struct {
		name     string
		input    *corev1alpha1.InstancePreset
		expected *corev1alpha1.InstancePreset
	}{
		{
			name:     "nil components",
			input:    newTestPreset(nil),
			expected: newTestPreset(nil),
		},
		{
			name:     "empty components",
			input:    newTestPreset(map[string]corev1alpha1.ComponentSpec{}),
			expected: newTestPreset(map[string]corev1alpha1.ComponentSpec{}),
		},
		{
			name:     "resolves secretRef",
			input:    newTestPreset(map[string]corev1alpha1.ComponentSpec{"pmm": {Config: emptySecretRef}}),
			expected: newTestPreset(map[string]corev1alpha1.ComponentSpec{"pmm": {Config: resolvedSecretRef}}),
		},
		{
			name: "other component does not resolve secretRef",
			input: newTestPreset(map[string]corev1alpha1.ComponentSpec{
				"other": {Config: emptySecretRef},
			}),
			expected: newTestPreset(map[string]corev1alpha1.ComponentSpec{
				"other": {Config: emptySecretRef},
			}),
		},
		{
			name: "resolve monitoringConfigName",
			input: newTestPreset(map[string]corev1alpha1.ComponentSpec{
				"pmm": {
					CustomSpec: &runtime.RawExtension{
						Raw: mustMarshal(t, map[string]any{"monitoringConfigName": ""}),
					},
				},
			}),
			expected: newTestPreset(map[string]corev1alpha1.ComponentSpec{
				"pmm": {
					CustomSpec: &runtime.RawExtension{
						Raw: mustMarshal(t, map[string]any{"monitoringConfigName": "default-monitoring"}),
					},
				},
			}),
		},
		{
			name: "resolve monitoringConfig",
			input: newTestPreset(map[string]corev1alpha1.ComponentSpec{
				"pmm": {
					CustomSpec: &runtime.RawExtension{
						Raw: mustMarshal(t, map[string]any{"monitoringConfig": ""}),
					},
				},
			}),
			expected: newTestPreset(map[string]corev1alpha1.ComponentSpec{
				"pmm": {
					CustomSpec: &runtime.RawExtension{
						Raw: mustMarshal(t, map[string]any{"monitoringConfig": "default-monitoring"}),
					},
				},
			}),
		},
		{
			name: "resolve monitoringConfigRef",
			input: newTestPreset(map[string]corev1alpha1.ComponentSpec{
				"pmm": {
					CustomSpec: &runtime.RawExtension{
						Raw: mustMarshal(t, map[string]any{"monitoringConfigRef": corev1.LocalObjectReference{Name: ""}}),
					},
				},
			}),
			expected: newTestPreset(map[string]corev1alpha1.ComponentSpec{
				"pmm": {
					CustomSpec: &runtime.RawExtension{
						Raw: mustMarshal(t, map[string]any{"monitoringConfigRef": corev1.LocalObjectReference{Name: "default-monitoring"}}),
					},
				},
			}),
		},
		{
			name: "other component does not resolve monitoringConfig",
			input: newTestPreset(map[string]corev1alpha1.ComponentSpec{
				"other": {
					CustomSpec: &runtime.RawExtension{
						Raw: mustMarshal(t, map[string]any{"monitoringConfigName": ""}),
					},
				},
			}),
			expected: newTestPreset(map[string]corev1alpha1.ComponentSpec{
				"other": {
					CustomSpec: &runtime.RawExtension{
						Raw: mustMarshal(t, map[string]any{"monitoringConfigName": ""}),
					},
				},
			}),
		},
		{
			name: "resolve nested monitoringConfig",
			input: newTestPreset(map[string]corev1alpha1.ComponentSpec{
				"pmm": {
					CustomSpec: &runtime.RawExtension{
						Raw: mustMarshal(t, map[string]any{"nested": map[string]any{"monitoringConfigName": ""}}),
					},
				},
			}),
			expected: newTestPreset(map[string]corev1alpha1.ComponentSpec{
				"pmm": {
					CustomSpec: &runtime.RawExtension{
						Raw: mustMarshal(t, map[string]any{"nested": map[string]any{"monitoringConfigName": "default-monitoring"}}),
					},
				},
			}),
		},
		{
			name: "other not supported fields do not resolve",
			input: newTestPreset(map[string]corev1alpha1.ComponentSpec{
				"pmm": {
					CustomSpec: &runtime.RawExtension{
						Raw: mustMarshal(t, map[string]any{"randomField": ""}),
					},
				},
			}),
			expected: newTestPreset(map[string]corev1alpha1.ComponentSpec{
				"pmm": {
					CustomSpec: &runtime.RawExtension{
						Raw: mustMarshal(t, map[string]any{"randomField": ""}),
					},
				},
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := handler.resolveNamespaceDefaults(ctx, tt.input, namespace)
			require.NoError(t, err)
			require.EqualValues(t, tt.expected.Spec, actual.Spec)
		})
	}
}

func mustMarshal(t *testing.T, v any) []byte {
	t.Helper()
	data, err := json.Marshal(v)
	require.NoError(t, err)
	return data
}
