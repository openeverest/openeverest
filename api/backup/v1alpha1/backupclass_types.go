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
	"slices"
	"strings"

	"github.com/xeipuuv/gojsonschema"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// BackupExecutionMode selects how a BackupClass implements backup and restore
// operations.
//
// +kubebuilder:validation:Enum=ProviderManaged;Job
type BackupExecutionMode string

const (
	// BackupExecutionModeProviderManaged delegates backup and restore to the
	// provider's reconciler, which typically configures an in-cluster agent
	// (PBM, pgBackRest, Barman, ...) on the engine itself. The Backup and
	// Restore CRs become trigger + status holders; the actual orchestration
	// happens inside the provider's Sync loop.
	BackupExecutionModeProviderManaged BackupExecutionMode = "ProviderManaged"

	// BackupExecutionModeJob runs backup and restore operations as Kubernetes
	// Jobs that talk to the database from outside (e.g., pg_dump, mysqldump).
	// All execution detail lives under .spec.job and .spec.restoreJob.
	BackupExecutionModeJob BackupExecutionMode = "Job"
)

// BackupClassSpec defines the desired state of BackupClass.
type BackupClassSpec struct {
	// DisplayName is a human-readable name for the backup class.
	DisplayName string `json:"displayName,omitempty"`
	// Description is the description of the backup class.
	Description string `json:"description,omitempty"`
	// SupportedProviders is the list of provider names that this backup class
	// supports. The Instance.spec.provider must appear in this list for the
	// class to be usable on that Instance.
	SupportedProviders ProviderNameList `json:"supportedProviders,omitempty"`
	// ExecutionMode selects between job-based and provider-managed execution.
	// +kubebuilder:validation:Required
	ExecutionMode BackupExecutionMode `json:"executionMode"`
	// ProviderManaged contains hints for ExecutionMode="ProviderManaged". The
	// schema is intentionally open: providers may surface capability
	// information (e.g., whether PITR is supported, schedule expression
	// dialect) without forcing a CRD change. Must be unset when
	// ExecutionMode is "Job".
	// +optional
	ProviderManaged *ProviderManagedSpec `json:"providerManaged,omitempty"`
	// Config contains the OpenAPI v3 schema describing the backup-time
	// configuration accepted by this class. Backup.spec.config is validated
	// against this schema.
	Config BackupClassConfig `json:"config,omitempty"`
	// JobSpec is the specification of the backup job.
	// +optional
	JobSpec *BackupJobSpec `json:"jobSpec,omitempty"`
	// CleanupJobSpec is the specification of the cleanup job.
	// +optional
	CleanupJobSpec *BackupJobSpec `json:"cleanupJobSpec,omitempty"`
	// DataStoreConstraints defines compatibility requirements and prerequisites that must be satisfied
	// by a DataStore before this backup tool can be used with it. This allows the backup tool to
	// express specific requirements about the database configuration needed for successful backup operations,
	// such as required database fields, specific engine configurations, or other database properties.
	// When a DataStore references this backup tool, the operator will validate the DataStore
	// against these constraints before proceeding with the backup operation.
	// +optional
	DataStoreConstraints BackupClassDataStoreConstraints `json:"dataStoreConstraints,omitempty"`
	// Permissions defines the permissions required by the backup tool.
	// These permissions are used to generate a Role for the backup job.
	// +optional
	Permissions []rbacv1.PolicyRule `json:"permissions,omitempty"`
	// ClusterPermissions defines the cluster-wide permissions required by the backup tool.
	// These permissions are used to generate a ClusterRole for the backup job.
	// +optional
	ClusterPermissions []rbacv1.PolicyRule `json:"clusterPermissions,omitempty"`
}

// ProviderManagedSpec carries opaque hints for ExecutionMode="ProviderManaged"
// classes. It mirrors the Config pattern: the field is opaque
// to the runtime; providers interpret it.
type ProviderManagedSpec struct {
	// SupportsPITR indicates whether this class supports point-in-time recovery.
	// Used by Restore validation when Restore.spec.dataSource.pitr is set.
	// +optional
	SupportsPITR bool `json:"supportsPITR,omitempty"`
}

// ProviderNameList is a type alias for a list of provider names.
type ProviderNameList []string

// Has checks if the list contains the specified provider.
func (e ProviderNameList) Has(provider string) bool {
	return slices.Contains(e, provider)
}

// BackupClassConfig contains additional configuration defined for the backup class.
type BackupClassConfig struct {
	// OpenAPIV3Schema is the OpenAPI v3 schema of the backup class.
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	// +optional
	OpenAPIV3Schema *apiextensionsv1.JSONSchemaProps `json:"openAPIV3Schema,omitempty"`
}

// ErrSchemaValidationFailure is returned when the parameters do not conform to the BackupClass schema defined in .spec.config.
var ErrSchemaValidationFailure = errors.New("schema validation failed")

// Validate the config for the backup class.
func (cfg *BackupClassConfig) Validate(params *runtime.RawExtension) error {
	schema := cfg.OpenAPIV3Schema
	if schema == nil && params != nil {
		return ErrSchemaValidationFailure
	}
	if schema == nil && params == nil {
		return nil
	}

	// Additional properties are implicitly disallowed
	schema.AdditionalProperties = &apiextensionsv1.JSONSchemaPropsOrBool{
		Allows: false,
	}

	// Unmarshal the parameters into a generic map
	var paramsMap map[string]interface{}
	if err := json.Unmarshal(params.Raw, &paramsMap); err != nil {
		return fmt.Errorf("failed to unmarshal parameters: %w", err)
	}

	// Convert the OpenAPI v3 schema to a JSON schema validator
	schemaJSON, err := json.Marshal(schema)
	if err != nil {
		return fmt.Errorf("failed to marshal OpenAPI v3 schema: %w", err)
	}

	schemaLoader := gojsonschema.NewStringLoader(string(schemaJSON))
	paramsLoader := gojsonschema.NewGoLoader(paramsMap)

	// Validate the parameters against the schema
	result, err := gojsonschema.Validate(schemaLoader, paramsLoader)
	if err != nil {
		return fmt.Errorf("failed to validate parameters: %w", err)
	}

	if !result.Valid() {
		var validationErrors []string
		for _, err := range result.Errors() {
			validationErrors = append(validationErrors, err.String())
		}
		return errors.Join(ErrSchemaValidationFailure, fmt.Errorf("validation errors: %s", strings.Join(validationErrors, "; ")))
	}
	return nil
}

// BackupJobSpec defines the specification for the Kubernetes job.
type BackupJobSpec struct {
	// Image is the image of the backup tool.
	Image string `json:"image,omitempty"`
	// Command is the command to run the backup tool.
	// +optional
	Command []string `json:"command,omitempty"`
}

// ErrInvalidExecutionMode is returned when the BackupClassSpec mixes fields
// from multiple execution modes or omits the required block for the chosen
// mode.
var ErrInvalidExecutionMode = errors.New("invalid execution mode configuration")

// ValidateExecutionMode enforces the invariants between ExecutionMode and the
// mode-specific blocks (Job/RestoreJob vs ProviderManaged).
func (s *BackupClassSpec) ValidateExecutionMode() error {
	switch s.ExecutionMode {
	case BackupExecutionModeProviderManaged:
		if s.JobSpec != nil {
			return fmt.Errorf("%w: executionMode=ProviderManaged must not set .spec.jobSpec", ErrInvalidExecutionMode)
		}
	case BackupExecutionModeJob:
		if s.JobSpec == nil {
			return fmt.Errorf("%w: executionMode=Job requires .spec.jobSpec", ErrInvalidExecutionMode)
		}
		if s.ProviderManaged != nil {
			return fmt.Errorf("%w: executionMode=Job must not set .spec.providerManaged", ErrInvalidExecutionMode)
		}
	default:
		return fmt.Errorf("%w: unknown executionMode %q", ErrInvalidExecutionMode, s.ExecutionMode)
	}
	return nil
}

// BackupClassDataStoreConstraints defines compatibility requirements and prerequisites
// that must be satisfied by a DataStore before this backup tool can be used with it.
type BackupClassDataStoreConstraints struct {
	// RequiredFields contains a list of fields that must be set in the DataStore spec.
	// Each key is a JSON path expressions that points to a field in the DataStore spec.
	// For example, ".spec.engine.type" or ".spec.dataSource.dataImport.config.someField".
	// +optional
	RequiredFields []string `json:"requiredFields,omitempty"`
}

// BackupClassStatus defines the observed state of BackupClass.
type BackupClassStatus struct {
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=bc
// +kubebuilder:resource:scope=Cluster

// BackupClass is the Schema for the backupclasses API
type BackupClass struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// +required
	Spec BackupClassSpec `json:"spec"`
	// +optional
	Status BackupClassStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// BackupClassList contains a list of BackupClass
type BackupClassList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []BackupClass `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BackupClass{}, &BackupClassList{})
}
