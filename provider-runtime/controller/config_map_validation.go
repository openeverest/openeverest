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
	"maps"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/pkg/common"
)

// ErrConfigMapSchemaValidation is returned when a configmap fails schema validation.
var ErrConfigMapSchemaValidation = errors.New("configmap does not conform to schema")

// ValidateConfigMapSchema validates a Kubernetes ConfigMap's data against
// an OpenAPI v3 schema defined in the provider's configmap definitions.
//
// The configmap must have the openeverest.io/definition label set to identify which
// schema to validate against. The function extracts data from the configmap,
// then validates against the schema from providerSpec.ConfigMaps[definition].
//
// Returns ErrConfigMapSchemaValidation if the configmap is nil, missing the definition label,
// or the definition does not exist in the provider spec.
func ValidateConfigMapSchema(configMap *corev1.ConfigMap, providerSpec *corev1alpha1.ProviderSpec) error {
	if configMap == nil {
		return fmt.Errorf("configmap cannot be nil: %w", ErrConfigMapSchemaValidation)
	}

	if providerSpec == nil {
		return fmt.Errorf("providerSpec cannot be nil: %w", ErrConfigMapSchemaValidation)
	}

	// Get the configmap definition name from the label.
	configMapDefinition := configMap.GetLabels()[common.OpenEverestDefinitionLabel]
	if configMapDefinition == "" {
		return fmt.Errorf("missing %s label: %w", common.OpenEverestDefinitionLabel, ErrConfigMapSchemaValidation)
	}

	// Look up the configmap definition from the provider spec.
	configMapDef, ok := providerSpec.ConfigMaps[configMapDefinition]
	if !ok {
		return fmt.Errorf("configmap definition %q does not exist: %w", configMapDefinition, ErrConfigMapSchemaValidation)
	}

	// ParametersSchema is optional; if not defined, no validation is performed.
	if configMapDef.ParametersSchema == nil {
		return nil
	}

	// Merge Data and BinaryData for validation.
	// BinaryData is converted to string, which may corrupt non-UTF-8 bytes.
	// This is acceptable for schema validation purposes since binary content
	// typically doesn't conform to JSON schema validation.
	data := make(map[string]string, len(configMap.Data)+len(configMap.BinaryData))
	maps.Copy(data, configMap.Data)
	for k, v := range configMap.BinaryData {
		data[k] = string(v)
	}

	dataJSON, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal configmap data: %w", errors.Join(ErrConfigMapSchemaValidation, err))
	}

	if err := configMapDef.ParametersSchema.Validate(&runtime.RawExtension{Raw: dataJSON}); err != nil {
		return fmt.Errorf("%w: %w", ErrConfigMapSchemaValidation, err)
	}

	return nil
}
