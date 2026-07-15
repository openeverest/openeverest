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

	// Create a schema that requires "username" and "password" fields.
	requiredSchema := &apiextensionsv1.JSONSchemaProps{
		Type: "object",
		Properties: map[string]apiextensionsv1.JSONSchemaProps{
			"username": {Type: "string"},
			"password": {Type: "string"},
		},
		Required: []string{"username", "password"},
	}

	// Provider with secret definition.
	providerWithSchema := &corev1alpha1.Provider{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-provider",
		},
		Spec: corev1alpha1.ProviderSpec{
			Secrets: map[string]corev1alpha1.SecretDefinition{
				"database-credentials": {
					OpenAPIV3Schema: requiredSchema,
				},
			},
		},
	}

	// Provider with shared secret definition.
	providerWithSharedSchema := &corev1alpha1.Provider{
		ObjectMeta: metav1.ObjectMeta{
			Name: "shared-provider",
		},
		Spec: corev1alpha1.ProviderSpec{
			Secrets: map[string]corev1alpha1.SecretDefinition{
				"shared-credentials": {
					OpenAPIV3Schema: requiredSchema,
					Shared:          true,
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
			name: "valid secret with stringData",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "valid-secret",
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
			providers: []*corev1alpha1.Provider{providerWithSchema},
		},
		{
			name: "valid secret with data (not stringData)",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "valid-secret-data",
					Labels: map[string]string{
						common.OpenEverestDefinitionLabel: "database-credentials",
						common.OpenEverestProviderLabel:   "test-provider",
					},
				},
				Type: corev1.SecretTypeOpaque,
				Data: map[string][]byte{
					"username": []byte("admin"),
					"password": []byte("secret123"),
				},
			},
			providers: []*corev1alpha1.Provider{providerWithSchema},
		},
		{
			name: "valid secret without provider",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "shared-secret",
					Labels: map[string]string{
						common.OpenEverestDefinitionLabel: "shared-credentials",
						// No provider label
					},
				},
				Type: corev1.SecretTypeOpaque,
				StringData: map[string]string{
					"username": "admin",
					"password": "secret123",
				},
			},
			providers: []*corev1alpha1.Provider{providerWithSharedSchema},
		},
		{
			name: "missing definition label",
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
			providers: []*corev1alpha1.Provider{providerWithSchema},
			err:       "definition label",
		},
		{
			name: "missing required field",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "missing-field",
					Labels: map[string]string{
						common.OpenEverestDefinitionLabel: "database-credentials",
						common.OpenEverestProviderLabel:   "test-provider",
					},
				},
				Type: corev1.SecretTypeOpaque,
				StringData: map[string]string{
					"username": "admin",
					// missing "password"
				},
			},
			providers: []*corev1alpha1.Provider{providerWithSchema},
			err:       "password",
		},
		{
			name: "additional property not allowed",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "extra-field",
					Labels: map[string]string{
						common.OpenEverestDefinitionLabel: "database-credentials",
						common.OpenEverestProviderLabel:   "test-provider",
					},
				},
				Type: corev1.SecretTypeOpaque,
				StringData: map[string]string{
					"username":    "admin",
					"password":    "secret123",
					"extra-field": "not-allowed",
				},
			},
			providers: []*corev1alpha1.Provider{providerWithSchema},
			err:       "Additional property",
		},
		{
			name: "definition not found in provider",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "unknown-definition",
					Labels: map[string]string{
						common.OpenEverestDefinitionLabel: "unknown-definition",
						common.OpenEverestProviderLabel:   "test-provider",
					},
				},
				Type: corev1.SecretTypeOpaque,
				StringData: map[string]string{
					"key": "value",
				},
			},
			providers: []*corev1alpha1.Provider{providerWithSchema},
			err:       "not found in provider",
		},
		{
			name: "provider not found",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "provider-not-found",
					Labels: map[string]string{
						common.OpenEverestDefinitionLabel: "database-credentials",
						common.OpenEverestProviderLabel:   "non-existent-provider",
					},
				},
				Type: corev1.SecretTypeOpaque,
				StringData: map[string]string{
					"key": "value",
				},
			},
			providers: []*corev1alpha1.Provider{providerWithSchema},
			err:       "failed to get provider",
		},

		{
			name: "no provider label and no shared definition found",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "no-shared-def",
					Labels: map[string]string{
						common.OpenEverestDefinitionLabel: "database-credentials",
						// No provider label
					},
				},
				Type: corev1.SecretTypeOpaque,
				StringData: map[string]string{
					"key": "value",
				},
			},
			providers: []*corev1alpha1.Provider{providerWithSchema}, // has definition but not shared
			err:       "not found",
		},
		{
			name: "empty secret data",
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
			providers: []*corev1alpha1.Provider{providerWithSchema},
			err:       "must have data or stringData",
		},
		{
			name: "invalid secret name",
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
			providers: []*corev1alpha1.Provider{providerWithSchema},
			err:       "name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Build fake client with providers.
			clientBuilder := fake.NewClientBuilder().WithScheme(scheme)
			for _, p := range tt.providers {
				clientBuilder = clientBuilder.WithObjects(p)
			}
			fakeClient := clientBuilder.Build()

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

			result, err := handler.CreateSecret(ctx, cluster, namespace, tt.secret)

			if tt.err != "" {
				require.ErrorContains(t, err, tt.err)

				return
			}

			require.NoError(t, err)
			assert.NotNil(t, result)
			mockNext.AssertCalled(t, "CreateSecret", mock.Anything, cluster, namespace, tt.secret)
		})
	}
}
