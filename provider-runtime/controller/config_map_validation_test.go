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

package controller

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	commonv1alpha1 "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/pkg/common"
)

func TestValidateConfigMapSchema(t *testing.T) {
	t.Parallel()

	schema := &apiextensionsv1.JSONSchemaProps{
		Type: "object",
		Properties: map[string]apiextensionsv1.JSONSchemaProps{
			"endpoint": {Type: "string"},
			"timeout":  {Type: "string"},
		},
		Required: []string{"endpoint"},
	}

	providerSpec := &corev1alpha1.ProviderSpec{
		ConfigMaps: map[string]corev1alpha1.ConfigMapDefinition{
			"test-config": {
				ParametersSchema: &commonv1alpha1.ParametersSchema{
					OpenAPIV3Schema: schema,
				},
			},
		},
	}

	tests := []struct {
		name         string
		configMap    *corev1.ConfigMap
		providerSpec *corev1alpha1.ProviderSpec
		wantErr      string
	}{
		{
			name:         "nil configmap fails",
			configMap:    nil,
			providerSpec: providerSpec,
			wantErr:      "configmap cannot be nil",
		},
		{
			name: "nil provider spec fails",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-configmap",
					Labels: map[string]string{
						common.OpenEverestDefinitionLabel: "test-config",
					},
				},
				Data: map[string]string{
					"endpoint": "https://example.com",
				},
			},
			providerSpec: nil,
			wantErr:      "providerSpec cannot be nil",
		},
		{
			name: "missing labels fails",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-configmap",
				},
				Data: map[string]string{
					"endpoint": "https://example.com",
				},
			},
			providerSpec: providerSpec,
			wantErr:      "missing openeverest.io/definition label",
		},
		{
			name: "missing definition label fails",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: "test-configmap"},
				Data: map[string]string{
					"endpoint": "https://example.com",
				},
			},
			providerSpec: providerSpec,
			wantErr:      "missing openeverest.io/definition label",
		},
		{
			name: "definition not found fails",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-configmap",
					Labels: map[string]string{
						common.OpenEverestDefinitionLabel: "non-existent",
					},
				},
				Data: map[string]string{
					"endpoint": "https://example.com",
				},
			},
			providerSpec: providerSpec,
			wantErr:      `configmap definition "non-existent" does not exist`,
		},
		{
			name: "valid configmap with data",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-configmap",
					Labels: map[string]string{
						common.OpenEverestDefinitionLabel: "test-config",
					},
				},
				Data: map[string]string{
					"endpoint": "https://example.com",
					"timeout":  "30s",
				},
			},
			providerSpec: providerSpec,
		},
		{
			name: "valid configmap with only required field",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-configmap",
					Labels: map[string]string{
						common.OpenEverestDefinitionLabel: "test-config",
					},
				},
				Data: map[string]string{
					"endpoint": "https://example.com",
				},
			},
			providerSpec: providerSpec,
		},
		{
			name: "valid configmap with binaryData",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-configmap",
					Labels: map[string]string{
						common.OpenEverestDefinitionLabel: "test-config",
					},
				},
				BinaryData: map[string][]byte{
					"endpoint": []byte("https://example.com"),
				},
			},
			providerSpec: providerSpec,
		},
		{
			name: "binaryData merged with data",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-configmap",
					Labels: map[string]string{
						common.OpenEverestDefinitionLabel: "test-config",
					},
				},
				Data: map[string]string{
					"timeout": "30s",
				},
				BinaryData: map[string][]byte{
					"endpoint": []byte("https://example.com"),
				},
			},
			providerSpec: providerSpec,
		},
		{
			name: "empty configmap fails schema validation",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-configmap",
					Labels: map[string]string{
						common.OpenEverestDefinitionLabel: "test-config",
					},
				},
			},
			providerSpec: providerSpec,
			wantErr:      "endpoint is required",
		},
		{
			name: "missing required field fails",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-configmap",
					Labels: map[string]string{
						common.OpenEverestDefinitionLabel: "test-config",
					},
				},
				Data: map[string]string{
					"timeout": "30s",
				},
			},
			providerSpec: providerSpec,
			wantErr:      "endpoint is required",
		},
		{
			name: "additional property not allowed",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-configmap",
					Labels: map[string]string{
						common.OpenEverestDefinitionLabel: "test-config",
					},
				},
				Data: map[string]string{
					"endpoint": "https://example.com",
					"extra":    "not-allowed",
				},
			},
			providerSpec: providerSpec,
			wantErr:      "Additional property extra is not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateConfigMapSchema(tt.configMap, tt.providerSpec)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)

				return
			}

			require.NoError(t, err)
		})
	}
}
