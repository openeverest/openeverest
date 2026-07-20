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

package v1alpha1

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/xeipuuv/gojsonschema"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
)

// ProviderSpec defines the desired state of Provider
type ProviderSpec struct {
	ComponentTypes map[string]ComponentType `json:"componentTypes,omitempty"`
	Components     map[string]Component     `json:"components,omitempty"`
	Topologies     map[string]Topology      `json:"topologies,omitempty"`

	// Versions defines curated version bundles — named sets of component
	// versions that are known to be mutually compatible. Users reference
	// a bundle via Instance.Spec.Version. If the user does not set a version,
	// the bundle whose Default field is true is used automatically.
	Versions []VersionBundle `json:"versions,omitempty"`

	// ParametersSchema declares the OpenAPI v3 schema for the instance-wide
	// parameters payload (Instance.spec.parameters).
	// +optional
	ParametersSchema *common.ParametersSchema `json:"parametersSchema,omitempty"`

	// UISchema holds the UI rendering hints for each topology.
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	UISchema *runtime.RawExtension `json:"uiSchema,omitempty"`

	// Secrets defines Secret types this provider supports.
	// +optional
	Secrets map[string]SecretDefinition `json:"secrets,omitempty"`
}

// VersionBundle is a curated set of component versions known to be mutually
// compatible. Provider developers define bundles in definition/versions.yaml.
type VersionBundle struct {
	// Name is the unique identifier for this bundle (e.g. "8.0.12").
	// Users set Instance.Spec.Version to this value to select the bundle.
	Name string `json:"name"`

	// Components maps component names to their version strings for this bundle.
	// Keys must match component names defined in ProviderSpec.Components.
	Components map[string]string `json:"components,omitempty"`

	// Default marks this bundle as the implicit choice when an Instance omits
	// Spec.Version entirely. Exactly one bundle should have Default: true.
	Default bool `json:"default,omitempty"`
}

type ComponentType struct {
	Versions []ComponentVersion `json:"versions,omitempty"`
}

type ComponentVersion struct {
	Version string `json:"version,omitempty"`
	Image   string `json:"image,omitempty"`
	Default bool   `json:"default,omitempty"`
}

type Component struct {
	Type string `json:"type,omitempty"`

	// ParametersSchema declares the OpenAPI v3 schema for this component's
	// parameters payload (Instance.spec.components[].parameters).
	// +optional
	ParametersSchema *common.ParametersSchema `json:"parametersSchema,omitempty"`
}

type Topology struct {
	Components map[string]TopologyComponent `json:"components,omitempty"`

	// ParametersSchema declares the OpenAPI v3 schema for topology-specific
	// parameters (Instance.spec.topology.parameters).
	// +optional
	ParametersSchema *common.ParametersSchema `json:"parametersSchema,omitempty"`
}

type TopologyComponent struct {
	Optional bool `json:"optional,omitempty"`
	// TODO: Do we need defaults?
	// Defaults map[string]interface{} `json:"defaults,omitempty"`
}

// SecretDefinition describes a Secret type that a provider supports.
type SecretDefinition struct {
	// UISchema holds UI rendering hints for the secret creation form.
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	UISchema *runtime.RawExtension `json:"uiSchema,omitempty"`

	// OpenAPIV3Schema is the OpenAPI v3 schema for validating secret data/stringData.
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	OpenAPIV3Schema *apiextensionsv1.JSONSchemaProps `json:"openAPIV3Schema,omitempty"` //nolint:tagliatelle
}

// ErrSecretSchemaValidation is returned when data does not conform to
// the OpenAPI v3 schema defined in the provider's secret definition.
var ErrSecretSchemaValidation = errors.New("secret schema validation failed")

// Validate validates data against the OpenAPI v3 schema in the SecretDefinition.
// The data parameter should be a map of key-value pairs representing the secret's
// data or stringData fields.
//
// If the schema is nil, validation passes for any non-nil data.
// Returns ErrSecretSchemaValidation if validation fails.
func (secretDef *SecretDefinition) Validate(data map[string]any) error {
	if secretDef.OpenAPIV3Schema == nil {
		return nil
	}

	if data == nil {
		return fmt.Errorf("data cannot be nil: %w", ErrSecretSchemaValidation)
	}

	// Copy the schema to avoid mutating the original.
	schemaCopy := secretDef.OpenAPIV3Schema.DeepCopy()

	// Additional properties are disallowed.
	schemaCopy.AdditionalProperties = &apiextensionsv1.JSONSchemaPropsOrBool{
		Allows: false,
	}

	// Convert the OpenAPI v3 schema to JSON for the validator.
	schemaJSON, err := json.Marshal(schemaCopy)
	if err != nil {
		return fmt.Errorf("failed to marshal schema: %w", errors.Join(ErrSecretSchemaValidation, err))
	}

	schemaLoader := gojsonschema.NewStringLoader(string(schemaJSON))
	dataLoader := gojsonschema.NewGoLoader(data)

	// Validate the data against the schema.
	result, err := gojsonschema.Validate(schemaLoader, dataLoader)
	if err != nil {
		return fmt.Errorf("validation failed: %w", errors.Join(ErrSecretSchemaValidation, err))
	}

	if !result.Valid() {
		var validationErrors []string
		for _, verr := range result.Errors() {
			validationErrors = append(validationErrors, verr.String())
		}

		return fmt.Errorf("%w: %s", ErrSecretSchemaValidation, strings.Join(validationErrors, "; "))
	}

	return nil
}

// ProviderStatus defines the observed state of Provider.
type ProviderStatus struct {
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,shortName=pr;prv;prov

// Provider is the Schema for the providers API
type Provider struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// +required
	Spec ProviderSpec `json:"spec"`
	// +optional
	Status ProviderStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// ProviderList contains a list of Provider
type ProviderList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []Provider `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Provider{}, &ProviderList{})
}
