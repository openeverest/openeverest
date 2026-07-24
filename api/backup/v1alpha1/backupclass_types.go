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
	"slices"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
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
	// All execution detail lives under .spec.job.
	BackupExecutionModeJob BackupExecutionMode = "Job"
)

// BackupClassSpec defines the desired state of BackupClass.
//
// +kubebuilder:validation:XValidation:rule="self.executionMode != 'Job' || has(self.job)",message="spec.job is required when executionMode is Job"
// +kubebuilder:validation:XValidation:rule="!has(self.job) || self.executionMode == 'Job'",message="spec.job is only allowed when executionMode is Job"
// +kubebuilder:validation:XValidation:rule="!has(self.providerManaged) || self.executionMode == 'ProviderManaged'",message="spec.providerManaged is only allowed when executionMode is ProviderManaged"
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
	// ParametersSchema declares the OpenAPI v3 schema describing the
	// backup-time parameters accepted by this class. Backup.spec.parameters
	// and InstanceBackupSchedule.parameters are both validated against it.
	// +optional
	ParametersSchema common.ParametersSchema `json:"parametersSchema,omitempty"`
	// RestoreParametersSchema declares the OpenAPI v3 schema describing the
	// restore-time parameters accepted by this class. Restore.spec.parameters
	// is validated against it.
	// +optional
	RestoreParametersSchema common.ParametersSchema `json:"restoreParametersSchema,omitempty"`
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

	// Job contains all execution detail for ExecutionMode="Job". Required
	// when ExecutionMode is "Job"; must be unset when ExecutionMode is
	// "ProviderManaged".
	// +optional
	Job *JobModeSpec `json:"job,omitempty"`
}

// JobModeSpec bundles everything the in-tree controller needs to run backup
// and restore operations as Kubernetes Jobs in ExecutionMode="Job".
type JobModeSpec struct {
	// Backup describes the job spawned per Backup CR.
	// +kubebuilder:validation:Required
	Backup JobExecution `json:"backup"`
	// Restore describes the job spawned per Restore CR. When unset, restores
	// are not supported by this class.
	// +optional
	Restore *JobExecution `json:"restore,omitempty"`
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

	// PITRParametersSchema declares the OpenAPI v3 schema for per-storage
	// PITR parameters (InstanceBackupStoragePITR.Parameters). The provider
	// validates Instance.spec.backup PITR payloads against it inside
	// Validate(); the UI renders a matching form from it.
	// +optional
	PITRParametersSchema *common.ParametersSchema `json:"pitrParametersSchema,omitempty"`
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

// BackupJobSpec defines the specification for the Kubernetes job.
type BackupJobSpec struct {
	// Image is the image of the backup class.
	Image string `json:"image,omitempty"`
	// Command is the command to run the backup class.
	// +optional
	Command []string `json:"command,omitempty"`
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
// +kubebuilder:resource:scope=Cluster,shortName=bc

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
