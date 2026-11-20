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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
)

// BackupExecutionMode selects how a BackupClass implements backup and restore
// operations. It discriminates the class's execution union: every mode has a
// member block named after it, and a new mode adds a value and a block.
//
// +kubebuilder:validation:Enum=ProviderManaged
type BackupExecutionMode string

const (
	// BackupExecutionModeProviderManaged delegates backup and restore to the
	// provider named in spec.providerManaged.providerRef, whose reconciler
	// typically configures an in-cluster agent (PBM, pgBackRest, Barman, ...)
	// on the engine itself. The Backup and Restore CRs become trigger and
	// status holders; the orchestration happens inside the provider's Sync
	// loop.
	BackupExecutionModeProviderManaged BackupExecutionMode = "ProviderManaged"
)

// BackupClassSpec defines the desired state of BackupClass.
//
// +kubebuilder:validation:XValidation:rule="self.executionMode != 'ProviderManaged' || has(self.providerManaged)",message="spec.providerManaged is required when executionMode is ProviderManaged"
// +kubebuilder:validation:XValidation:rule="!has(self.providerManaged) || self.executionMode == 'ProviderManaged'",message="spec.providerManaged is only allowed when executionMode is ProviderManaged"
type BackupClassSpec struct {
	// DisplayName is a human-readable name for the backup class.
	DisplayName string `json:"displayName,omitempty"`
	// Description is the description of the backup class.
	Description string `json:"description,omitempty"`
	// BackupStorageClasses lists the BackupStorageClasses the class's executor
	// can write to, each with the authentication modes it supports. A
	// BackupStorage is usable with the class only when both its class and
	// its authentication mode are listed.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=16
	// +listType=map
	// +listMapKey=name
	BackupStorageClasses []BackupStorageClassSupport `json:"backupStorageClasses"`
	// ExecutionMode selects who executes the class's backup and restore
	// operations, and discriminates the execution union: the member named
	// after the mode is required, any other is forbidden. ProviderManaged is
	// the only mode: the provider in providerManaged.providerRef executes
	// them.
	// +kubebuilder:validation:Required
	ExecutionMode BackupExecutionMode `json:"executionMode"`
	// SupportsImport indicates whether the owner of this class can discover and
	// import restorable backups already sitting in a BackupStorage.
	// +optional
	SupportsImport bool `json:"supportsImport,omitempty"`
	// ProviderManaged is the member of the execution union for
	// ExecutionMode=ProviderManaged: the provider that executes the class and
	// what it supports. Required for that mode, forbidden for any other.
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
}

// ProviderManagedSpec is the ProviderManaged member of a BackupClass's
// execution union.
type ProviderManagedSpec struct {
	// ProviderRef names the Provider that claims and executes the class. It
	// is immutable: moving a class to another provider is delete and
	// recreate.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="providerRef is immutable"
	ProviderRef common.ObjectRef `json:"providerRef"`

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

// ExecutorProvider returns the name of the Provider that executes the class,
// or "" when its execution mode has none.
func (s *BackupClassSpec) ExecutorProvider() string {
	if s.ExecutionMode == BackupExecutionModeProviderManaged && s.ProviderManaged != nil {
		return s.ProviderManaged.ProviderRef.Name
	}
	return ""
}

// BackupStorageClassSupport is a BackupStorageClass a BackupClass can write
// to.
type BackupStorageClassSupport struct {
	// Name of the BackupStorageClass.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`
	Name string `json:"name"`
	// AuthModes are the authentication modes of that class the executor
	// supports.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=16
	// +kubebuilder:validation:items:Pattern=`^[A-Z][A-Za-z0-9]{0,62}$`
	// +listType=set
	AuthModes []string `json:"authModes"`
	// ParametersSchema narrows the BackupStorageClass's parametersSchema to
	// what the executor supports, typically the parameters of the contract
	// version it was built with: a storage whose parameters it rejects
	// cannot be used with the class. Absent adds no restriction.
	// +optional
	ParametersSchema *common.ParametersSchema `json:"parametersSchema,omitempty"`
	// TLS declares which settings of BackupStorage.spec.tls the executor
	// honours. Absent means None for each: a storage that uses a setting the
	// executor does not support cannot be used with the class, and a
	// storage without spec.tls always can.
	// +optional
	TLS *BackupStorageTLSSupport `json:"tls,omitempty"`
}

// TLSOptionSupport is whether an executor honours a setting of
// BackupStorage.spec.tls.
//
// +kubebuilder:validation:Enum=Supported;None
type TLSOptionSupport string

// TLS option support levels.
const (
	// TLSOptionSupported means the executor honours the setting.
	TLSOptionSupported = TLSOptionSupport("Supported")
	// TLSOptionNone means the executor cannot honour the setting.
	TLSOptionNone = TLSOptionSupport("None")
)

// BackupStorageTLSSupport is what an executor honours of
// BackupStorage.spec.tls.
type BackupStorageTLSSupport struct {
	// CustomCA is whether the executor verifies the store against the CA
	// bundle in spec.tls.ca.
	// +kubebuilder:default=None
	// +optional
	CustomCA TLSOptionSupport `json:"customCA,omitempty"` //nolint:tagliatelle // CA, as in spec.tls.ca
	// InsecureSkipVerify is whether the executor can turn certificate
	// verification off, as spec.tls.insecureSkipVerify asks.
	// +kubebuilder:default=None
	// +optional
	InsecureSkipVerify TLSOptionSupport `json:"insecureSkipVerify,omitempty"`
}

// FindBackupStorageClass returns the class's support entry for the
// BackupStorageClass named name, or nil.
func (s *BackupClassSpec) FindBackupStorageClass(name string) *BackupStorageClassSupport {
	i := slices.IndexFunc(s.BackupStorageClasses, func(c BackupStorageClassSupport) bool { return c.Name == name })
	if i < 0 {
		return nil
	}
	return &s.BackupStorageClasses[i]
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

// BackupClassStatus defines the observed state of BackupClass. It has two
// writers, each applying only its own fields with server-side apply: the
// executor that claims the class owns the Accepted condition and core owns
// providers.
type BackupClassStatus struct {
	// Conditions: Accepted, set by the class's executor when it claims the
	// class, with the observedGeneration it judged.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// Providers is the compatibility of the class with each Provider that
	// can execute it: for a ProviderManaged class, the Provider in
	// providerManaged.providerRef. A Provider that is not listed is
	// incompatible. Written by core.
	// +listType=map
	// +listMapKey=name
	// +kubebuilder:validation:MaxItems=64
	// +optional
	Providers []BackupClassProviderCompatibility `json:"providers,omitempty"`
}

// BackupClass conditions and their reasons.
const (
	// BackupClassConditionAccepted reports whether the class's executor
	// claims it. Its observedGeneration is the generation it judged.
	BackupClassConditionAccepted = "Accepted"
	// ReasonPending means no executor has claimed the class yet.
	ReasonPending = "Pending"
	// ReasonUnknownClass means the executor named by the class does not
	// ship a class of that name.
	ReasonUnknownClass = "UnknownClass"
	// ReasonSpecDrift means the class's capabilities differ from those of
	// the class its executor ships under that name.
	ReasonSpecDrift = "SpecDrift"
	// ReasonNewerDefinition means the class is stamped with a newer release
	// of its executor than the one running, as while the executor is being
	// upgraded. The executor leaves Accepted Unknown until it runs that
	// release.
	ReasonNewerDefinition = "NewerDefinition"
	// ReasonOlderDefinition means the class is stamped with an older release
	// of its executor than the one running, as while the class is being
	// upgraded. The executor leaves Accepted Unknown until the class is.
	ReasonOlderDefinition = "OlderDefinition"
	// ReasonClassNotAccepted means the class is not Accepted at its current
	// generation yet.
	ReasonClassNotAccepted = "ClassNotAccepted"
	// ReasonProviderNotFound means the Provider the class names does not
	// exist.
	ReasonProviderNotFound = "ProviderNotFound"
	// ReasonProviderUnsupported means the class does not run on a Provider:
	// status.providers, computed for the class's current generation, has no
	// entry for it.
	ReasonProviderUnsupported = "ProviderUnsupported"
)

// ProviderVersionAnnotation stamps a BackupClass a provider ships with the
// provider release it belongs to, the appVersion of the provider chart. The
// provider compares it with the stamp of the class it ships under that name
// before comparing capabilities, so a class from another release is pending
// during an upgrade rather than drifted.
const ProviderVersionAnnotation = "openeverest.io/provider-version"

// BackupClassProviderCompatibility is the compatibility of a BackupClass
// with a Provider.
type BackupClassProviderCompatibility struct {
	// Name is the Provider name.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MaxLength=253
	Name string `json:"name"`
	// Compatibility is whether Instances of the Provider can use the class.
	// +kubebuilder:validation:Required
	Compatibility Compatibility `json:"compatibility"`
	// Reason is why the class is not Compatible.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Reason string `json:"reason,omitempty"`
	// Message is the human-readable detail of Reason.
	// +kubebuilder:validation:MaxLength=32768
	// +optional
	Message string `json:"message,omitempty"`
	// ClassGeneration is the generation of the class the entry was computed
	// for. An entry for another generation counts as Pending.
	// +kubebuilder:validation:Minimum=0
	// +optional
	ClassGeneration int64 `json:"classGeneration,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,shortName=bc
// +kubebuilder:printcolumn:name="Mode",type="string",JSONPath=".spec.executionMode"
// +kubebuilder:printcolumn:name="Provider",type="string",JSONPath=".spec.providerManaged.providerRef.name"
// +kubebuilder:printcolumn:name="Accepted",type="string",JSONPath=".status.conditions[?(@.type==\"Accepted\")].reason"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// BackupClass is the Schema for the backupclasses API
type BackupClass struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// +required
	Spec BackupClassSpec `json:"spec"`
	// +optional
	// +kubebuilder:default={conditions: {{type: "Accepted", status: "Unknown", reason: "Pending", message: "Waiting for the executor to claim the class", lastTransitionTime: "1970-01-01T00:00:00Z"}}}
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
