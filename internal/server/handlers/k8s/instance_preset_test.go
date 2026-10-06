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
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	commonv1alpha1 "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
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
	require.NoError(t, storagev1.AddToScheme(scheme))
	require.NoError(t, corev1alpha1.AddToScheme(scheme))

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(
			&storagev1.StorageClass{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "default-storage",
					Annotations: map[string]string{"storageclass.kubernetes.io/is-default-class": "true"},
				},
			},
			&storagev1.StorageClass{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "non-default-storage",
					Annotations: map[string]string{"storageclass.kubernetes.io/is-default-class": "false"},
				},
			},
			&corev1alpha1.NamespaceDefaults{
				ObjectMeta: metav1.ObjectMeta{Name: namespaceDefaultsName, Namespace: namespace},
				Spec: corev1alpha1.NamespaceDefaultsSpec{
					Defaults: []corev1alpha1.NamespaceDefault{
						{Path: "components.pmm.parameters.monitoringConfigName", DefaultRef: commonv1alpha1.ObjectRef{Name: "monitoring1"}},
						{Path: "components.pmm.parameters.monitoringConfig", DefaultRef: commonv1alpha1.ObjectRef{Name: "monitoring2"}},
						{Path: "components.pmm.parameters.monitoringConfigRef.name", DefaultRef: commonv1alpha1.ObjectRef{Name: "monitoring3"}},
						{Path: "components.pmm.parameters.nested.monitoringConfigName", DefaultRef: commonv1alpha1.ObjectRef{Name: "monitoring4"}},
					},
				},
			},
		).
		Build()

	handler := &k8sHandler{
		kubeConnector: kubernetes.NewEmpty(zap.NewNop().Sugar(), namespace).WithKubernetesClient(fakeClient),
		log:           zap.NewNop().Sugar(),
	}

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
			name: "inline configuration parameter passes through unchanged",
			input: newTestPreset(map[string]corev1alpha1.ComponentSpec{
				"pmm": {
					Parameters: &runtime.RawExtension{
						Raw: mustMarshal(t, map[string]any{"configuration": "key = value"}),
					},
				},
			}),
			expected: newTestPreset(map[string]corev1alpha1.ComponentSpec{
				"pmm": {
					Parameters: &runtime.RawExtension{
						Raw: mustMarshal(t, map[string]any{"configuration": "key = value"}),
					},
				},
			}),
		},
		{
			name: "resolve monitoringConfigName",
			input: newTestPreset(map[string]corev1alpha1.ComponentSpec{
				"pmm": {
					Parameters: &runtime.RawExtension{
						Raw: mustMarshal(t, map[string]any{"monitoringConfigName": ""}),
					},
				},
			}),
			expected: newTestPreset(map[string]corev1alpha1.ComponentSpec{
				"pmm": {
					Parameters: &runtime.RawExtension{
						Raw: mustMarshal(t, map[string]any{"monitoringConfigName": "monitoring1"}),
					},
				},
			}),
		},
		{
			name: "resolve monitoringConfig",
			input: newTestPreset(map[string]corev1alpha1.ComponentSpec{
				"pmm": {
					Parameters: &runtime.RawExtension{
						Raw: mustMarshal(t, map[string]any{"monitoringConfig": ""}),
					},
				},
			}),
			expected: newTestPreset(map[string]corev1alpha1.ComponentSpec{
				"pmm": {
					Parameters: &runtime.RawExtension{
						Raw: mustMarshal(t, map[string]any{"monitoringConfig": "monitoring2"}),
					},
				},
			}),
		},
		{
			name: "resolve monitoringConfigRef empty name",
			input: newTestPreset(map[string]corev1alpha1.ComponentSpec{
				"pmm": {
					Parameters: &runtime.RawExtension{
						Raw: mustMarshal(t, map[string]any{"monitoringConfigRef": corev1.LocalObjectReference{Name: ""}}),
					},
				},
			}),
			expected: newTestPreset(map[string]corev1alpha1.ComponentSpec{
				"pmm": {
					Parameters: &runtime.RawExtension{
						Raw: mustMarshal(t, map[string]any{"monitoringConfigRef": corev1.LocalObjectReference{Name: "monitoring3"}}),
					},
				},
			}),
		},
		{
			name: "resolve monitoringConfigRef empty struct",
			input: newTestPreset(map[string]corev1alpha1.ComponentSpec{
				"pmm": {
					Parameters: &runtime.RawExtension{
						Raw: mustMarshal(t, map[string]any{"monitoringConfigRef": corev1.LocalObjectReference{}}),
					},
				},
			}),
			expected: newTestPreset(map[string]corev1alpha1.ComponentSpec{
				"pmm": {
					Parameters: &runtime.RawExtension{
						Raw: mustMarshal(t, map[string]any{"monitoringConfigRef": corev1.LocalObjectReference{Name: "monitoring3"}}),
					},
				},
			}),
		},
		{
			name: "does not override monitoringConfigRef with existing name",
			input: newTestPreset(map[string]corev1alpha1.ComponentSpec{
				"pmm": {
					Parameters: &runtime.RawExtension{
						Raw: mustMarshal(t, map[string]any{"monitoringConfigRef": corev1.LocalObjectReference{Name: "user-set"}}),
					},
				},
			}),
			expected: newTestPreset(map[string]corev1alpha1.ComponentSpec{
				"pmm": {
					Parameters: &runtime.RawExtension{
						Raw: mustMarshal(t, map[string]any{"monitoringConfigRef": corev1.LocalObjectReference{Name: "user-set"}}),
					},
				},
			}),
		},
		{
			name: "does not override monitoringConfigName with existing value",
			input: newTestPreset(map[string]corev1alpha1.ComponentSpec{
				"pmm": {
					Parameters: &runtime.RawExtension{
						Raw: mustMarshal(t, map[string]any{"monitoringConfigName": "user-set"}),
					},
				},
			}),
			expected: newTestPreset(map[string]corev1alpha1.ComponentSpec{
				"pmm": {
					Parameters: &runtime.RawExtension{
						Raw: mustMarshal(t, map[string]any{"monitoringConfigName": "user-set"}),
					},
				},
			}),
		},
		{
			name: "other component does not resolve monitoringConfig",
			input: newTestPreset(map[string]corev1alpha1.ComponentSpec{
				"other": {
					Parameters: &runtime.RawExtension{
						Raw: mustMarshal(t, map[string]any{"monitoringConfigName": ""}),
					},
				},
			}),
			expected: newTestPreset(map[string]corev1alpha1.ComponentSpec{
				"other": {
					Parameters: &runtime.RawExtension{
						Raw: mustMarshal(t, map[string]any{"monitoringConfigName": ""}),
					},
				},
			}),
		},
		{
			name: "resolve nested monitoringConfig",
			input: newTestPreset(map[string]corev1alpha1.ComponentSpec{
				"pmm": {
					Parameters: &runtime.RawExtension{
						Raw: mustMarshal(t, map[string]any{"nested": map[string]any{"monitoringConfigName": ""}}),
					},
				},
			}),
			expected: newTestPreset(map[string]corev1alpha1.ComponentSpec{
				"pmm": {
					Parameters: &runtime.RawExtension{
						Raw: mustMarshal(t, map[string]any{"nested": map[string]any{"monitoringConfigName": "monitoring4"}}),
					},
				},
			}),
		},
		{
			name: "other not supported fields do not resolve",
			input: newTestPreset(map[string]corev1alpha1.ComponentSpec{
				"pmm": {
					Parameters: &runtime.RawExtension{
						Raw: mustMarshal(t, map[string]any{"randomField": ""}),
					},
				},
			}),
			expected: newTestPreset(map[string]corev1alpha1.ComponentSpec{
				"pmm": {
					Parameters: &runtime.RawExtension{
						Raw: mustMarshal(t, map[string]any{"randomField": ""}),
					},
				},
			}),
		},
		{
			name: "resolve storageClass",
			input: newTestPreset(map[string]corev1alpha1.ComponentSpec{
				"engine": {
					Storage: &corev1alpha1.Storage{StorageClass: nil},
				},
			}),
			expected: newTestPreset(map[string]corev1alpha1.ComponentSpec{
				"engine": {
					Storage: &corev1alpha1.Storage{StorageClass: ptr.To("default-storage")},
				},
			}),
		},
		{
			name: "resolve empty storageClass pointer",
			input: newTestPreset(map[string]corev1alpha1.ComponentSpec{
				"engine": {
					Storage: &corev1alpha1.Storage{StorageClass: ptr.To("")},
				},
			}),
			expected: newTestPreset(map[string]corev1alpha1.ComponentSpec{
				"engine": {
					Storage: &corev1alpha1.Storage{StorageClass: ptr.To("default-storage")},
				},
			}),
		},
		{
			name: "does not override existing storageClass",
			input: newTestPreset(map[string]corev1alpha1.ComponentSpec{
				"engine": {
					Storage: &corev1alpha1.Storage{StorageClass: ptr.To("custom-storage")},
				},
			}),
			expected: newTestPreset(map[string]corev1alpha1.ComponentSpec{
				"engine": {
					Storage: &corev1alpha1.Storage{StorageClass: ptr.To("custom-storage")},
				},
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := handler.resolveDefaults(ctx, tt.input, namespace)
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

func TestResolveDefaults_ProviderPrecedence(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	namespace := "test-namespace"

	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, storagev1.AddToScheme(scheme))
	require.NoError(t, corev1alpha1.AddToScheme(scheme))

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(&corev1alpha1.NamespaceDefaults{
			ObjectMeta: metav1.ObjectMeta{Name: namespaceDefaultsName, Namespace: namespace},
			Spec: corev1alpha1.NamespaceDefaultsSpec{
				Defaults: []corev1alpha1.NamespaceDefault{
					{
						Path:       "components.engine.parameters.certSecretRef.name",
						DefaultRef: commonv1alpha1.ObjectRef{Name: "tls-certificate"},
					},
					{
						ProviderRef: &commonv1alpha1.ObjectRef{Name: "psmdb"},
						Path:        "components.engine.parameters.certSecretRef.name",
						DefaultRef:  commonv1alpha1.ObjectRef{Name: "psmdb-tls-certificate"},
					},
				},
			},
		}).
		Build()

	handler := &k8sHandler{
		kubeConnector: kubernetes.NewEmpty(zap.NewNop().Sugar(), namespace).WithKubernetesClient(fakeClient),
		log:           zap.NewNop().Sugar(),
	}

	tests := []struct {
		name     string
		input    *corev1alpha1.InstancePreset
		expected *corev1alpha1.InstancePreset
	}{
		{
			name: "provider match wins",
			input: &corev1alpha1.InstancePreset{
				ObjectMeta: metav1.ObjectMeta{Name: "test"},
				Spec: corev1alpha1.InstancePresetSpec{
					InstanceSpec: corev1alpha1.InstanceSpec{
						ProviderRef: commonv1alpha1.ObjectRef{Name: "psmdb"},
						Components: map[string]corev1alpha1.ComponentSpec{
							"engine": {
								Parameters: &runtime.RawExtension{
									Raw: mustMarshal(t, map[string]any{
										"certSecretRef": make(map[string]any),
									}),
								},
							},
						},
					},
				},
			},
			expected: &corev1alpha1.InstancePreset{
				ObjectMeta: metav1.ObjectMeta{Name: "test"},
				Spec: corev1alpha1.InstancePresetSpec{
					InstanceSpec: corev1alpha1.InstanceSpec{
						ProviderRef: commonv1alpha1.ObjectRef{Name: "psmdb"},
						Components: map[string]corev1alpha1.ComponentSpec{
							"engine": {
								Parameters: &runtime.RawExtension{
									Raw: mustMarshal(t, map[string]any{
										"certSecretRef": map[string]any{"name": "psmdb-tls-certificate"},
									}),
								},
							},
						},
					},
				},
			},
		},
		{
			name: "fall back to provider agnostic default",
			input: &corev1alpha1.InstancePreset{
				ObjectMeta: metav1.ObjectMeta{Name: "test"},
				Spec: corev1alpha1.InstancePresetSpec{
					InstanceSpec: corev1alpha1.InstanceSpec{
						ProviderRef: commonv1alpha1.ObjectRef{Name: "other-provider"},
						Components: map[string]corev1alpha1.ComponentSpec{
							"engine": {
								Parameters: &runtime.RawExtension{
									Raw: mustMarshal(t, map[string]any{
										"certSecretRef": make(map[string]any),
									}),
								},
							},
						},
					},
				},
			},
			expected: &corev1alpha1.InstancePreset{
				ObjectMeta: metav1.ObjectMeta{Name: "test"},
				Spec: corev1alpha1.InstancePresetSpec{
					InstanceSpec: corev1alpha1.InstanceSpec{
						ProviderRef: commonv1alpha1.ObjectRef{Name: "other-provider"},
						Components: map[string]corev1alpha1.ComponentSpec{
							"engine": {
								Parameters: &runtime.RawExtension{
									Raw: mustMarshal(t, map[string]any{
										"certSecretRef": map[string]any{"name": "tls-certificate"},
									}),
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual, err := handler.resolveDefaults(ctx, tt.input, namespace)
			require.NoError(t, err)
			require.Equal(t, tt.expected, actual)
		})
	}
}

func TestNoNamespaceDefault(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	namespace := "test-namespace"

	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, storagev1.AddToScheme(scheme))
	require.NoError(t, corev1alpha1.AddToScheme(scheme))

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	handler := &k8sHandler{
		kubeConnector: kubernetes.NewEmpty(zap.NewNop().Sugar(), namespace).WithKubernetesClient(fakeClient),
		log:           zap.NewNop().Sugar(),
	}

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
			input:    newTestPreset(make(map[string]corev1alpha1.ComponentSpec)),
			expected: newTestPreset(make(map[string]corev1alpha1.ComponentSpec)),
		},
		{
			name: "resolve monitoringConfigName",
			input: newTestPreset(map[string]corev1alpha1.ComponentSpec{
				"pmm": {
					Parameters: &runtime.RawExtension{
						Raw: mustMarshal(t, map[string]any{"monitoringConfigName": ""}),
					},
				},
			}),
			expected: newTestPreset(map[string]corev1alpha1.ComponentSpec{
				"pmm": {
					Parameters: &runtime.RawExtension{
						Raw: mustMarshal(t, map[string]any{"monitoringConfigName": ""}),
					},
				},
			}),
		},
		{
			name: "resolve storageClass",
			input: newTestPreset(map[string]corev1alpha1.ComponentSpec{
				"engine": {
					Storage: &corev1alpha1.Storage{StorageClass: nil},
				},
			}),
			expected: newTestPreset(map[string]corev1alpha1.ComponentSpec{
				"engine": {
					Storage: &corev1alpha1.Storage{StorageClass: nil},
				},
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual, err := handler.resolveDefaults(ctx, tt.input, namespace)
			require.NoError(t, err)
			require.Equal(t, tt.expected.Spec, actual.Spec)
		})
	}
}
