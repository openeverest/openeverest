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
	"encoding/json"
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/pkg/common"
)

// ErrSecretSchemaValidation is returned when a secret fails schema validation.
var ErrSecretSchemaValidation = errors.New("secret does not conform to schema")

// ValidateSecretSchema validates a Kubernetes Secret's data or stringData against
// an OpenAPI v3 schema defined in the provider's secret definitions.
//
// The secret must have the openeverest.io/definition label set to identify which
// schema to validate against. The function extracts and merges data and stringData
// from the secret (with stringData taking precedence), then validates against the
// schema from providerSpec.Secrets[definition].
//
// Returns ErrSecretSchemaValidation if the secret is nil, missing the definition label,
// or the definition does not exist in the provider spec.
func ValidateSecretSchema(secret *corev1.Secret, providerSpec *corev1alpha1.ProviderSpec) error {
	if secret == nil {
		return fmt.Errorf("secret cannot be nil: %w", ErrSecretSchemaValidation)
	}

	if providerSpec == nil {
		return fmt.Errorf("providerSpec cannot be nil: %w", ErrSecretSchemaValidation)
	}

	// Get the secret definition name from the label.
	secretDefinition := secret.GetLabels()[common.OpenEverestDefinitionLabel]
	if secretDefinition == "" {
		return fmt.Errorf("missing %s label: %w", common.OpenEverestDefinitionLabel, ErrSecretSchemaValidation)
	}

	// Look up the secret definition from the provider spec.
	secretDef, ok := providerSpec.Secrets[secretDefinition]
	if !ok {
		return fmt.Errorf("secret definition %q does not exist: %w", secretDefinition, ErrSecretSchemaValidation)
	}

	// ParametersSchema is optional; if not defined, no validation is performed.
	if secretDef.ParametersSchema == nil {
		return nil
	}

	// Extract and merge data from the secret.
	// stringData takes precedence (same as Kubernetes behavior).
	data := extractSecretData(secret)

	dataJSON, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal secret data: %w", errors.Join(ErrSecretSchemaValidation, err))
	}

	if err := secretDef.ParametersSchema.Validate(&runtime.RawExtension{Raw: dataJSON}); err != nil {
		return fmt.Errorf("%w: %w", ErrSecretSchemaValidation, err)
	}

	return nil
}

// extractSecretData extracts and merges data and stringData from a Kubernetes Secret.
// stringData takes precedence over data (same as Kubernetes behavior).
// Returns nil if the secret is nil.
func extractSecretData(secret *corev1.Secret) map[string]any {
	if secret == nil {
		return nil
	}

	data := make(map[string]any)

	// Add decoded data entries.
	for k, v := range secret.Data {
		data[k] = string(v)
	}

	// Add stringData entries (overrides data if same key).
	for k, v := range secret.StringData {
		data[k] = v
	}

	return data
}
