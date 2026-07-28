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

package preset

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

func TestEnsureNamespaceRefsEmpty(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		spec    *corev1alpha1.InstanceSpec
		wantErr string
	}{
		{
			name: "empty spec is valid",
			spec: &corev1alpha1.InstanceSpec{},
		},
		{
			name: "cluster-scoped storageClass is allowed",
			spec: &corev1alpha1.InstanceSpec{
				Components: map[string]corev1alpha1.ComponentSpec{
					"engine": {
						Storage: &corev1alpha1.Storage{
							Size:         resource.MustParse("25Gi"),
							StorageClass: ptr.To("local-path"),
						},
					},
				},
			},
		},
		{
			name: "secret ref must be empty",
			spec: &corev1alpha1.InstanceSpec{
				Components: map[string]corev1alpha1.ComponentSpec{
					"engine": {
						Parameters: mustRawExt(t, map[string]any{
							"secretRef": map[string]any{
								"name": "my-secret",
							},
						}),
					},
				},
			},
			wantErr: `component "engine": parameters.secretRef must be empty in preset`,
		},
		{
			name: "configmap ref must be empty",
			spec: &corev1alpha1.InstanceSpec{
				Components: map[string]corev1alpha1.ComponentSpec{
					"proxy": {
						Parameters: mustRawExt(t, map[string]any{
							"objectRef": map[string]any{
								"name": "my-config",
							},
						}),
					},
				},
			},
			wantErr: `component "proxy": parameters.objectRef must be empty in preset`,
		},
		{
			name: "customSpec monitoringConfigName must be empty",
			spec: &corev1alpha1.InstanceSpec{
				Components: map[string]corev1alpha1.ComponentSpec{
					"monitoring": {
						Parameters: mustRawExt(t, map[string]any{
							"monitoringConfigName": "pmm-config",
						}),
					},
				},
			},
			wantErr: `component "monitoring": parameters.monitoringConfigName must be empty in preset`,
		},
		{
			name: "empty secretRef object is valid",
			spec: &corev1alpha1.InstanceSpec{
				Components: map[string]corev1alpha1.ComponentSpec{
					"monitoring": {
						Parameters: mustRawExt(t, map[string]any{
							"secretRef": map[string]any{
								"name": "",
							},
						}),
					},
				},
			},
		},
		{
			name: "nested customSpec refs are checked",
			spec: &corev1alpha1.InstanceSpec{
				Components: map[string]corev1alpha1.ComponentSpec{
					"pmm": {
						Parameters: mustRawExt(t, map[string]any{
							"nested": map[string]any{
								"monitoringConfigName": "config",
							},
						}),
					},
				},
			},
			wantErr: `component "pmm": parameters.nested.monitoringConfigName must be empty in preset`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := EnsureNamespaceRefsEmpty(tt.spec)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestClearNamespaceRefs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    *corev1alpha1.InstanceSpec
		expected *corev1alpha1.InstanceSpec
	}{
		{
			name:     "nil components",
			input:    &corev1alpha1.InstanceSpec{},
			expected: &corev1alpha1.InstanceSpec{},
		},
		{
			name: "clears secretRef and configMapRef",
			input: &corev1alpha1.InstanceSpec{
				Components: map[string]corev1alpha1.ComponentSpec{
					"engine": {
						Parameters: mustRawExt(t, map[string]any{
							"secretRef": map[string]any{
								"name": "my-secret",
							},
							"objectRef": map[string]any{
								"name": "my-config",
							},
							"key": "data.conf",
						}),
						Storage: &corev1alpha1.Storage{
							Size:         resource.MustParse("10Gi"),
							StorageClass: ptr.To("fast-ssd"),
						},
					},
				},
			},
			expected: &corev1alpha1.InstanceSpec{
				Components: map[string]corev1alpha1.ComponentSpec{
					"engine": {
						Parameters: mustRawExt(t, map[string]any{
							"secretRef": map[string]any{"name": ""},
							"objectRef": map[string]any{"name": ""},
							"key":       "data.conf",
						}),
						Storage: &corev1alpha1.Storage{
							Size:         resource.MustParse("10Gi"),
							StorageClass: ptr.To("fast-ssd"),
						},
					},
				},
			},
		},
		{
			name: "clears customSpec namespace refs",
			input: &corev1alpha1.InstanceSpec{
				Components: map[string]corev1alpha1.ComponentSpec{
					"monitoring": {
						Parameters: mustRawExt(t, map[string]any{
							"monitoringConfigName": "pmm-config",
							"secretRef": map[string]any{
								"name": "pmm-secret",
							},
							"retention": "7d",
						}),
					},
				},
			},
			expected: &corev1alpha1.InstanceSpec{
				Components: map[string]corev1alpha1.ComponentSpec{
					"monitoring": {
						Parameters: mustRawExt(t, map[string]any{
							"monitoringConfigName": "",
							"secretRef": map[string]any{
								"name": "",
							},
							"retention": "7d",
						}),
					},
				},
			},
		},
		{
			name: "leaves cluster-scoped storageClass untouched",
			input: &corev1alpha1.InstanceSpec{
				Components: map[string]corev1alpha1.ComponentSpec{
					"engine": {
						Storage: &corev1alpha1.Storage{
							Size:         resource.MustParse("25Gi"),
							StorageClass: ptr.To("premium-rwo"),
						},
						Parameters: mustRawExt(t, map[string]any{
							"secretRef": map[string]any{
								"name": "creds",
							},
						}),
					},
				},
			},
			expected: &corev1alpha1.InstanceSpec{
				Components: map[string]corev1alpha1.ComponentSpec{
					"engine": {
						Storage: &corev1alpha1.Storage{
							Size:         resource.MustParse("25Gi"),
							StorageClass: ptr.To("premium-rwo"),
						},
						Parameters: mustRawExt(t, map[string]any{
							"secretRef": map[string]any{
								"name": "",
							},
						}),
					},
				},
			},
		},
		{
			name: "clears nested customSpec refs",
			input: &corev1alpha1.InstanceSpec{
				Components: map[string]corev1alpha1.ComponentSpec{
					"pmm": {
						Parameters: mustRawExt(t, map[string]any{
							"nested": map[string]any{
								"monitoringConfigName": "config",
								"other":                "value",
							},
						}),
					},
				},
			},
			expected: &corev1alpha1.InstanceSpec{
				Components: map[string]corev1alpha1.ComponentSpec{
					"pmm": {
						Parameters: mustRawExt(t, map[string]any{
							"nested": map[string]any{
								"monitoringConfigName": "",
								"other":                "value",
							},
						}),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ClearNamespaceRefs(tt.input)
			require.NoError(t, err)
			require.EqualValues(t, tt.expected, tt.input)
		})
	}
}

func TestResolveNamespaceRefs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     *corev1alpha1.InstanceSpec
		namespace string
		resolver  *mockResolver
		expected  *corev1alpha1.InstanceSpec
		wantErr   string
	}{
		{
			name:      "nil components",
			input:     &corev1alpha1.InstanceSpec{},
			namespace: "test",
			resolver:  &mockResolver{},
			expected:  &corev1alpha1.InstanceSpec{},
		},
		{
			name: "resolves empty secretRef",
			input: &corev1alpha1.InstanceSpec{
				Components: map[string]corev1alpha1.ComponentSpec{
					"engine": {
						Parameters: mustRawExt(t, map[string]any{
							"secretRef": map[string]any{
								"name": "",
							},
						}),
					},
				},
			},
			namespace: "prod",
			resolver: &mockResolver{
				defaults: map[resolverKey]string{
					{ns: "prod", kind: KindSecret, component: "engine"}: "default-secret",
				},
			},
			expected: &corev1alpha1.InstanceSpec{
				Components: map[string]corev1alpha1.ComponentSpec{
					"engine": {
						Parameters: mustRawExt(t, map[string]any{
							"secretRef": map[string]any{
								"name": "default-secret",
							},
						}),
					},
				},
			},
		},
		{
			name: "resolves empty configMapRef",
			input: &corev1alpha1.InstanceSpec{
				Components: map[string]corev1alpha1.ComponentSpec{
					"proxy": {
						Parameters: mustRawExt(t, map[string]any{
							"objectRef": map[string]any{
								"name": "",
							},
						}),
					},
				},
			},
			namespace: "dev",
			resolver: &mockResolver{
				defaults: map[resolverKey]string{
					{ns: "dev", kind: KindConfigMap, component: "proxy"}: "default-config",
				},
			},
			expected: &corev1alpha1.InstanceSpec{
				Components: map[string]corev1alpha1.ComponentSpec{
					"proxy": {
						Parameters: mustRawExt(t, map[string]any{
							"objectRef": map[string]any{
								"name": "default-config",
							},
						}),
					},
				},
			},
		},
		{
			name: "resolves empty storageClass",
			input: &corev1alpha1.InstanceSpec{
				Components: map[string]corev1alpha1.ComponentSpec{
					"engine": {
						Storage: &corev1alpha1.Storage{
							Size: resource.MustParse("50Gi"),
						},
					},
				},
			},
			namespace: "prod",
			resolver: &mockResolver{
				defaults: map[resolverKey]string{
					{ns: "prod", kind: KindStorageClass, component: "engine"}: "fast-ssd",
				},
			},
			expected: &corev1alpha1.InstanceSpec{
				Components: map[string]corev1alpha1.ComponentSpec{
					"engine": {
						Storage: &corev1alpha1.Storage{
							Size:         resource.MustParse("50Gi"),
							StorageClass: ptr.To("fast-ssd"),
						},
					},
				},
			},
		},
		{
			name: "resolves customSpec monitoringConfigName",
			input: &corev1alpha1.InstanceSpec{
				Components: map[string]corev1alpha1.ComponentSpec{
					"monitoring": {
						Parameters: mustRawExt(t, map[string]any{
							"monitoringConfigName": "",
							"interval":             "30s",
						}),
					},
				},
			},
			namespace: "staging",
			resolver: &mockResolver{
				defaults: map[resolverKey]string{
					{ns: "staging", kind: KindMonitoringConfig, component: "monitoring"}: "pmm-config",
				},
			},
			expected: &corev1alpha1.InstanceSpec{
				Components: map[string]corev1alpha1.ComponentSpec{
					"monitoring": {
						Parameters: mustRawExt(t, map[string]any{
							"monitoringConfigName": "pmm-config",
							"interval":             "30s",
						}),
					},
				},
			},
		},
		{
			name: "does not override filled refs",
			input: &corev1alpha1.InstanceSpec{
				Components: map[string]corev1alpha1.ComponentSpec{
					"engine": {
						Storage: &corev1alpha1.Storage{
							Size:         resource.MustParse("50Gi"),
							StorageClass: ptr.To("standard"),
						},
					},
				},
			},
			namespace: "prod",
			resolver: &mockResolver{
				defaults: map[resolverKey]string{
					{ns: "prod", kind: KindStorageClass, component: "engine"}: "local-path",
				},
			},
			expected: &corev1alpha1.InstanceSpec{
				Components: map[string]corev1alpha1.ComponentSpec{
					"engine": {
						Storage: &corev1alpha1.Storage{
							Size:         resource.MustParse("50Gi"),
							StorageClass: ptr.To("standard"),
						},
					},
				},
			},
		},
		{
			name: "resolver error propagates",
			input: &corev1alpha1.InstanceSpec{
				Components: map[string]corev1alpha1.ComponentSpec{
					"engine": {
						Parameters: mustRawExt(t, map[string]any{
							"secretRef": map[string]any{
								"name": "",
							},
						}),
					},
				},
			},
			namespace: "prod",
			resolver: &mockResolver{
				err: errors.New("no default secret found"),
			},
			wantErr: "no default secret found",
		},
		{
			name: "resolves nested customSpec refs",
			input: &corev1alpha1.InstanceSpec{
				Components: map[string]corev1alpha1.ComponentSpec{
					"pmm": {
						Parameters: mustRawExt(t, map[string]any{
							"nested": map[string]any{
								"monitoringConfigName": "",
							},
						}),
					},
				},
			},
			namespace: "staging",
			resolver: &mockResolver{
				defaults: map[resolverKey]string{
					{ns: "staging", kind: KindMonitoringConfig, component: "pmm"}: "pmm-config",
				},
			},
			expected: &corev1alpha1.InstanceSpec{
				Components: map[string]corev1alpha1.ComponentSpec{
					"pmm": {
						Parameters: mustRawExt(t, map[string]any{
							"nested": map[string]any{
								"monitoringConfigName": "pmm-config",
							},
						}),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ResolveNamespaceRefs(context.Background(), tt.input, tt.namespace, tt.resolver)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
				require.EqualValues(t, tt.expected, tt.input)
			}
		})
	}
}

// mockResolver implements DefaultResolver for testing.
type mockResolver struct {
	defaults map[resolverKey]string
	err      error
}

type resolverKey struct {
	ns        string
	kind      ResourceKind
	component string
}

func (m *mockResolver) ResolveDefault(ctx context.Context, namespace string, kind ResourceKind, component string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	key := resolverKey{ns: namespace, kind: kind, component: component}
	return m.defaults[key], nil
}

// mustRawExt marshals a value to runtime.RawExtension or panics.
func mustRawExt(t *testing.T, v any) *runtime.RawExtension {
	t.Helper()
	data, err := json.Marshal(v)
	require.NoError(t, err)
	return &runtime.RawExtension{Raw: data}
}
