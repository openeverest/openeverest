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

func TestCreateSecret_Validation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	namespace := "test-namespace"
	cluster := "test-cluster"

	// Provider with secret definition.
	provider := &corev1alpha1.Provider{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-provider",
		},
		Spec: corev1alpha1.ProviderSpec{
			Secrets: map[string]corev1alpha1.SecretDefinition{
				"database-credentials": {
					ParametersSchema: &commonv1alpha1.ParametersSchema{
						OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
							Type: "object",
							Properties: map[string]apiextensionsv1.JSONSchemaProps{
								"username": {Type: "string"},
								"password": {Type: "string"},
							},
							Required: []string{"username", "password"},
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
		secret    *corev1.Secret
		providers []*corev1alpha1.Provider
		err       string
	}{
		{
			name: "missing provider label fails",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "no-provider-label",
					Labels: map[string]string{
						common.OpenEverestDefinitionLabel: "database-credentials",
						// No provider label
					},
				},
				Type: corev1.SecretTypeOpaque,
				StringData: map[string]string{
					"username": "admin",
					"password": "secret123",
				},
			},
			providers: []*corev1alpha1.Provider{provider},
			err:       "missing openeverest.io/provider label",
		},
		{
			name: "missing definition label fails",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "no-definition-label",
					Labels: map[string]string{},
				},
				Type: corev1.SecretTypeOpaque,
				StringData: map[string]string{
					"key": "value",
				},
			},
			providers: []*corev1alpha1.Provider{provider},
			err:       "missing openeverest.io/definition label",
		},
		{
			name: "secret not conforming to schema fails",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "empty-secret",
					Labels: map[string]string{
						common.OpenEverestDefinitionLabel: "database-credentials",
						common.OpenEverestProviderLabel:   "test-provider",
					},
				},
				Type: corev1.SecretTypeOpaque,
				// No data or stringData
			},
			providers: []*corev1alpha1.Provider{provider},
			err:       "secret does not conform to schema",
		},
		{
			name: "invalid secret name fails",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "Invalid_Name_With_Underscore",
					Labels: map[string]string{
						common.OpenEverestDefinitionLabel: "database-credentials",
						common.OpenEverestProviderLabel:   "test-provider",
					},
				},
				Type: corev1.SecretTypeOpaque,
				StringData: map[string]string{
					"username": "admin",
					"password": "secret123",
				},
			},
			providers: []*corev1alpha1.Provider{provider},
			err:       "'name' can be at most 22 characters long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Use DeepCopy to avoid race conditions since the fake client
			// modifies the objects' ResourceVersion during Build().
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(provider.DeepCopy()).Build()
			kubeConnector := kubernetes.NewEmpty(zap.NewNop().Sugar(), namespace).WithKubernetesClient(fakeClient)

			// Create mock next handler.
			mockNext := &handlers.MockHandler{}
			mockNext.On("CreateSecret", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
				Return(tt.secret, nil)

			// Create validation handler.
			handler := &validateHandler{
				log:           zap.NewNop().Sugar(),
				kubeConnector: kubeConnector,
				next:          mockNext,
			}

			_, err := handler.CreateSecret(ctx, cluster, namespace, tt.secret)
			require.ErrorContains(t, err, tt.err)
		})
	}
}
