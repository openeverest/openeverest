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

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/pkg/common"
)

func TestValidateSecretSchema(t *testing.T) {
	t.Parallel()

	schema := &apiextensionsv1.JSONSchemaProps{
		Type: "object",
		Properties: map[string]apiextensionsv1.JSONSchemaProps{
			"username": {Type: "string"},
			"password": {Type: "string"},
		},
		Required: []string{"username", "password"},
	}

	providerSpec := &corev1alpha1.ProviderSpec{
		Secrets: map[string]corev1alpha1.SecretDefinition{
			"database-credentials": {
				OpenAPIV3Schema: schema,
			},
		},
	}

	tests := []struct {
		name         string
		secret       *corev1.Secret
		providerSpec *corev1alpha1.ProviderSpec
		wantErr      string
	}{
		{
			name:         "nil secret fails",
			secret:       nil,
			providerSpec: providerSpec,
			wantErr:      "secret cannot be nil",
		},
		{
			name: "nil provider spec fails",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-secret",
					Labels: map[string]string{
						common.OpenEverestDefinitionLabel: "database-credentials",
					},
				},
				StringData: map[string]string{
					"username": "admin",
					"password": "secret",
				},
			},
			providerSpec: nil,
			wantErr:      "providerSpec cannot be nil",
		},
		{
			name: "missing labels fails",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-secret",
				},
				Data: map[string][]byte{
					"username": []byte("admin"),
					"password": []byte("secret"),
				},
			},
			providerSpec: providerSpec,
			wantErr:      "missing openeverest.io/definition label",
		},
		{
			name: "missing definition label fails",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: "test-secret"},
				StringData: map[string]string{
					"username": "admin",
					"password": "secret",
				},
			},
			providerSpec: providerSpec,
			wantErr:      "missing openeverest.io/definition label",
		},
		{
			name: "definition not found fails",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-secret",
					Labels: map[string]string{
						common.OpenEverestDefinitionLabel: "non-existent",
					},
				},
				StringData: map[string]string{
					"username": "admin",
					"password": "secret",
				},
			},
			providerSpec: providerSpec,
			wantErr:      `secret definition "non-existent" does not exist`,
		},
		{
			name: "valid secret with data",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-secret",
					Labels: map[string]string{
						common.OpenEverestDefinitionLabel: "database-credentials",
					},
				},
				Data: map[string][]byte{
					"username": []byte("admin"),
					"password": []byte("secret"),
				},
			},
			providerSpec: providerSpec,
		},
		{
			name: "valid secret with stringData",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-secret",
					Labels: map[string]string{
						common.OpenEverestDefinitionLabel: "database-credentials",
					},
				},
				StringData: map[string]string{
					"username": "admin",
					"password": "secret",
				},
			},
			providerSpec: providerSpec,
		},
		{
			name: "stringData overrides data",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-secret",
					Labels: map[string]string{
						common.OpenEverestDefinitionLabel: "database-credentials",
					},
				},
				Data: map[string][]byte{
					"username": []byte("data-user"),
					"password": []byte("data-pass"),
				},
				StringData: map[string]string{
					"username": "stringdata-user",
					"password": "stringdata-pass",
				},
			},
			providerSpec: providerSpec,
		},
		{
			name: "empty secret fails schema validation",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-secret",
					Labels: map[string]string{
						common.OpenEverestDefinitionLabel: "database-credentials",
					},
				},
			},
			providerSpec: providerSpec,
			wantErr:      "username is required",
		},
		{
			name: "missing required field fails",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-secret",
					Labels: map[string]string{
						common.OpenEverestDefinitionLabel: "database-credentials",
					},
				},
				StringData: map[string]string{
					"username": "admin",
				},
			},
			providerSpec: providerSpec,
			wantErr:      "password is required",
		},
		{
			name: "additional property not allowed",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-secret",
					Labels: map[string]string{
						common.OpenEverestDefinitionLabel: "database-credentials",
					},
				},
				StringData: map[string]string{
					"username": "admin",
					"password": "secret",
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

			err := ValidateSecretSchema(tt.secret, tt.providerSpec)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)

				return
			}

			require.NoError(t, err)
		})
	}
}
