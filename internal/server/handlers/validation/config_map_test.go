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

package validation

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	commonv1alpha1 "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/internal/server/handlers"
	"github.com/openeverest/openeverest/v2/pkg/common"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
)

func TestCreateConfigMap_Validation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	namespace := "test-namespace"
	cluster := "test-cluster"

	provider := &corev1alpha1.Provider{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-provider",
		},
		Spec: corev1alpha1.ProviderSpec{
			ConfigMaps: map[string]corev1alpha1.ConfigMapDefinition{
				"custom-config": {
					ParametersSchema: &commonv1alpha1.ParametersSchema{
						OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
							Type: "object",
							Properties: map[string]apiextensionsv1.JSONSchemaProps{
								"key1": {Type: "string"},
								"key2": {Type: "string"},
							},
							Required: []string{"key1"},
						},
					},
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, corev1alpha1.AddToScheme(scheme))

	tests := []struct {
		name      string
		configMap *corev1.ConfigMap
		err       string
	}{
		{
			name:      "nil configmap fails",
			configMap: nil,
			err:       "configmap cannot be nil",
		},
		{
			name: "missing provider label fails",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "no-provider",
					Labels: map[string]string{
						common.OpenEverestDefinitionLabel: "custom-config",
					},
				},
				Data: map[string]string{
					"key1": "val1",
				},
			},
			err: "missing openeverest.io/provider label",
		},
		{
			name: "missing definition label fails",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "no-definition",
					Labels: map[string]string{
						common.OpenEverestProviderLabel: "test-provider",
					},
				},
				Data: map[string]string{
					"key1": "val1",
				},
			},
			err: "missing openeverest.io/definition label",
		},
		{
			name: "provider not found fails",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "missing-provider",
					Labels: map[string]string{
						common.OpenEverestDefinitionLabel: "custom-config",
						common.OpenEverestProviderLabel:   "non-existent-provider",
					},
				},
				Data: map[string]string{
					"key1": "val1",
				},
			},
			err: "failed to get provider \"non-existent-provider\"",
		},
		{
			name: "invalid configmap name fails",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "Invalid_Name_With_Underscore",
					Labels: map[string]string{
						common.OpenEverestDefinitionLabel: "custom-config",
						common.OpenEverestProviderLabel:   "test-provider",
					},
				},
				Data: map[string]string{
					"key1": "val1",
				},
			},
			err: "'name' can be at most 22 characters long",
		},
		{
			name: "configmap not conforming to schema fails",
			configMap: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "empty-data",
					Labels: map[string]string{
						common.OpenEverestDefinitionLabel: "custom-config",
						common.OpenEverestProviderLabel:   "test-provider",
					},
				},
				Data: map[string]string{},
			},
			err: "configmap does not conform to schema",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(provider.DeepCopy()).Build()
			kubeConnector := kubernetes.NewEmpty(zap.NewNop().Sugar(), namespace).WithKubernetesClient(fakeClient)

			mockNext := &handlers.MockHandler{}
			mockNext.On("CreateConfigMap", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
				Return(tt.configMap, nil)

			handler := &validateHandler{
				log:           zap.NewNop().Sugar(),
				kubeConnector: kubeConnector,
				next:          mockNext,
			}

			_, err := handler.CreateConfigMap(ctx, cluster, namespace, tt.configMap)
			require.ErrorContains(t, err, tt.err)
		})
	}
}

func TestCreateConfigMap_Success(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	namespace := "test-namespace"
	cluster := "test-cluster"

	provider := &corev1alpha1.Provider{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-provider",
		},
		Spec: corev1alpha1.ProviderSpec{
			ConfigMaps: map[string]corev1alpha1.ConfigMapDefinition{
				"custom-config": {
					ParametersSchema: &commonv1alpha1.ParametersSchema{
						OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
							Type: "object",
							Properties: map[string]apiextensionsv1.JSONSchemaProps{
								"key1": {Type: "string"},
								"key2": {Type: "string"},
							},
							Required: []string{"key1"},
						},
					},
				},
			},
		},
	}

	validConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: "valid-config",
			Labels: map[string]string{
				common.OpenEverestDefinitionLabel: "custom-config",
				common.OpenEverestProviderLabel:   "test-provider",
			},
		},
		Data: map[string]string{
			"key1": "val1",
			"key2": "val2",
		},
	}

	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, corev1alpha1.AddToScheme(scheme))

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(provider.DeepCopy()).Build()
	kubeConnector := kubernetes.NewEmpty(zap.NewNop().Sugar(), namespace).WithKubernetesClient(fakeClient)

	mockNext := &handlers.MockHandler{}
	mockNext.On("CreateConfigMap", ctx, cluster, namespace, validConfigMap).
		Return(validConfigMap, nil).Once()

	handler := &validateHandler{
		log:           zap.NewNop().Sugar(),
		kubeConnector: kubeConnector,
		next:          mockNext,
	}

	result, err := handler.CreateConfigMap(ctx, cluster, namespace, validConfigMap)
	require.NoError(t, err)
	assert.Equal(t, validConfigMap, result)
	mockNext.AssertExpectations(t)
}

func TestConfigMap_ProxiesToNextHandler(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	namespace := "test-namespace"
	cluster := "test-cluster"
	name := "my-config"
	providerName := "test-provider"
	definitionName := "custom-config"

	expectedList := &corev1.ConfigMapList{
		Items: []corev1.ConfigMap{
			{
				ObjectMeta: metav1.ObjectMeta{Name: name},
			},
		},
	}
	expectedCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}

	mockNext := &handlers.MockHandler{}
	mockNext.On("ListConfigMaps", ctx, cluster, namespace, providerName, definitionName).
		Return(expectedList, nil).Once()
	mockNext.On("GetConfigMap", ctx, cluster, namespace, name).
		Return(expectedCM, nil).Once()
	mockNext.On("DeleteConfigMap", ctx, cluster, namespace, name).
		Return(nil).Once()

	handler := &validateHandler{
		log:  zap.NewNop().Sugar(),
		next: mockNext,
	}

	list, err := handler.ListConfigMaps(ctx, cluster, namespace, providerName, definitionName)
	require.NoError(t, err)
	assert.Equal(t, expectedList, list)

	cm, err := handler.GetConfigMap(ctx, cluster, namespace, name)
	require.NoError(t, err)
	assert.Equal(t, expectedCM, cm)

	err = handler.DeleteConfigMap(ctx, cluster, namespace, name)
	require.NoError(t, err)

	mockNext.AssertExpectations(t)
}
