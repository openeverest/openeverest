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
	// RestoreConfig contains the OpenAPI v3 schema describing the restore-time
	// configuration accepted by this class. Restore.spec.config is validated
	// against this schema.
	// +optional
	RestoreConfig BackupClassConfig `json:"restoreConfig,omitempty"`
	// InstanceConstraints defines compatibility requirements that must be
	// satisfied by an Instance before this backup class can be used with it.
	// +optional
	InstanceConstraints BackupClassInstanceConstraints `json:"instanceConstraints,omitempty"`

	// UISchema contains free-form rendering hints for the frontend forms that
	// configure backup, restore, and PITR for an Instance using this class.
	// The runtime treats this field as opaque; only the UI consumes it. The
	// recommended shape groups fields by the modal that renders them
	// (e.g. "backup", "pitr", "restore"), mirroring Provider.spec.uiSchema.
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	UISchema *runtime.RawExtension `json:"uiSchema,omitempty"`

	// Job contains execution detail for ExecutionMode="Job". Must be unset
	// when ExecutionMode is "ProviderManaged".
	// +optional
	Job *JobExecution `json:"job,omitempty"`
	// RestoreJob contains execution detail for the restore job in
	// ExecutionMode="Job". Must be unset when ExecutionMode is
	// "ProviderManaged".
	// +optional
	RestoreJob *JobExecution `json:"restoreJob,omitempty"`
}

// JobExecution bundles the Kubernetes resources the controller needs to spawn
// to perform a single backup or restore operation in ExecutionMode="Job".
type JobExecution struct {
	// JobSpec is the specification of the backup or restore job.
	// +kubebuilder:validation:Required
	JobSpec *BackupJobSpec `json:"jobSpec"`
	// CleanupJobSpec is the optional specification of a cleanup job that runs
	// when the parent Backup or Restore CR is deleted.
	// +optional
	CleanupJobSpec *BackupJobSpec `json:"cleanupJobSpec,omitempty"`
	// Permissions are namespace-scoped PolicyRules granted to the job pod via
	// a generated Role and RoleBinding.
	// +optional
	Permissions []rbacv1.PolicyRule `json:"permissions,omitempty"`
	// ClusterPermissions are cluster-scoped PolicyRules granted via a
	// generated ClusterRole and ClusterRoleBinding.
	// +optional
	ClusterPermissions []rbacv1.PolicyRule `json:"clusterPermissions,omitempty"`
}

// ProviderManagedSpec carries opaque hints for ExecutionMode="ProviderManaged"
// classes. It mirrors the Config/RestoreConfig pattern: the field is opaque
// to the runtime; providers interpret it.
type ProviderManagedSpec struct {
	// SupportsPITR indicates whether this class supports point-in-time recovery.
	// Used by Restore validation when Restore.spec.dataSource.pitr is set.
	// +optional
	SupportsPITR bool `json:"supportsPITR,omitempty"`

	// Limits caps how many storages, PITR-enabled storages, and schedules per
	// storage an Instance may declare under .spec.backup when this class is
	// selected. Unset fields mean "unlimited" (still subject to the core
	// MaxItems ceilings on InstanceBackupSpec). The runtime enforces these
	// caps both at admission time (provider validation webhook) and before
	// dispatching ConfigureBackup; providers may add engine-specific
	// constraints on top via Context.BackupClassLimits().
	// +optional
	Limits *BackupClassLimits `json:"limits,omitempty"`

	// PITRConfigSchema describes the shape of per-storage PITR custom config
	// (InstanceBackupStoragePITR.Config). The field is free-form and opaque
	// to the runtime; the provider validates Instance.spec.backup PITR
	// payloads against it inside Validate(). The recommended payload is an
	// OpenAPI v3 schema fragment so the UI can render a matching form, but
	// any provider-specific dialect is permitted.
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	PITRConfigSchema *runtime.RawExtension `json:"pitrConfigSchema,omitempty"`
}

// BackupClassLimits expresses the caps a ProviderManaged BackupClass places
// on the backup configuration of an Instance that uses it. All fields are
// optional pointers; nil means "unlimited" (the core MaxItems ceilings on
// InstanceBackupSpec still apply).
type BackupClassLimits struct {
	// MaxStorages is the maximum number of entries allowed in
	// Instance.spec.backup.storages.
	// +optional
	// +kubebuilder:validation:Minimum=1
	MaxStorages *int32 `json:"maxStorages,omitempty"`

	// MaxPITREnabledStorages is the maximum number of storages on an Instance
	// that may set .pitr.enabled=true at the same time. Engines that support
	// a single PITR stream (e.g. PSMDB, PXC) declare 1 here. Engines that
	// archive WAL to every repo (e.g. PG) leave this unset.
	// +optional
	// +kubebuilder:validation:Minimum=0
	MaxPITREnabledStorages *int32 `json:"maxPITREnabledStorages,omitempty"`

	// MaxSchedulesPerStorage is the maximum number of recurring schedules
	// allowed per Instance storage entry.
	// +optional
	// +kubebuilder:validation:Minimum=0
	MaxSchedulesPerStorage *int32 `json:"maxSchedulesPerStorage,omitempty"`
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
	// Image is the image of the backup class.
	Image string `json:"image,omitempty"`
	// Command is the command to run the backup class.
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
		if s.Job != nil || s.RestoreJob != nil {
			return fmt.Errorf("%w: executionMode=ProviderManaged must not set .spec.job or .spec.restoreJob", ErrInvalidExecutionMode)
		}
	case BackupExecutionModeJob:
		if s.Job == nil {
			return fmt.Errorf("%w: executionMode=Job requires .spec.job", ErrInvalidExecutionMode)
		}
		if s.ProviderManaged != nil {
			return fmt.Errorf("%w: executionMode=Job must not set .spec.providerManaged", ErrInvalidExecutionMode)
		}
	default:
		return fmt.Errorf("%w: unknown executionMode %q", ErrInvalidExecutionMode, s.ExecutionMode)
	}
	return nil
}

// BackupClassInstanceConstraints defines compatibility requirements and prerequisites
// that must be satisfied by a Instance before this backup class can be used with it.
type BackupClassInstanceConstraints struct {
	// RequiredFields contains a list of fields that must be set in the Instance spec.
	// Each key is a JSON path expressions that points to a field in the Instance spec.
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
