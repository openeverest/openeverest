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
	corev1 "k8s.io/api/core/v1"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/contract"
	"github.com/openeverest/openeverest/v2/contract/definitions"
	"github.com/openeverest/openeverest/v2/contract/referents"
)

// The identifiers below moved to the contract packages, which core and the
// provider runtime both import, so that core no longer imports the provider
// runtime. They stay here for one release so that providers can move at
// their own pace.

// ProviderLabel is the label key used to identify which provider manages an
// Instance.
//
// Deprecated: use contract.ProviderLabel; removed in the next release.
const ProviderLabel = contract.ProviderLabel

// ProviderFinalizer is the finalizer the provider reconciler adds to an
// Instance when it picks the Instance up.
//
// Deprecated: use contract.ProviderFinalizer; removed in the next release.
const ProviderFinalizer = contract.ProviderFinalizer

// StatusFieldOwner returns the server-side-apply field manager under which
// the runtime of the named provider writes the status it shares with other
// writers.
//
// Deprecated: use contract.ProviderStatusFieldOwner; removed in the next
// release.
func StatusFieldOwner(provider string) string {
	return contract.ProviderStatusFieldOwner(provider)
}

// ErrInvalidReference is the umbrella sentinel every reference-validation
// error satisfies via errors.Is.
//
// Deprecated: use contract/referents.ErrInvalidReference; removed in the next
// release.
var ErrInvalidReference = referents.ErrInvalidReference

// Reference-validation sentinels. Each is the very value its replacement in
// contract/referents is, so errors.Is matches either.
const (
	// ErrBackupNotFound is referents.ErrBackupNotFound.
	//
	// Deprecated: use contract/referents.ErrBackupNotFound; removed in the next
	// release.
	ErrBackupNotFound = referents.ErrBackupNotFound
	// ErrBackupClassNotFound is referents.ErrBackupClassNotFound.
	//
	// Deprecated: use contract/referents.ErrBackupClassNotFound; removed in the
	// next release.
	ErrBackupClassNotFound = referents.ErrBackupClassNotFound
	// ErrInstanceNotFound is referents.ErrInstanceNotFound.
	//
	// Deprecated: use contract/referents.ErrInstanceNotFound; removed in the
	// next release.
	ErrInstanceNotFound = referents.ErrInstanceNotFound
	// ErrBackupStorageNotFound is referents.ErrBackupStorageNotFound.
	//
	// Deprecated: use contract/referents.ErrBackupStorageNotFound; removed in
	// the next release.
	ErrBackupStorageNotFound = referents.ErrBackupStorageNotFound
	// ErrBackupNotSucceeded is referents.ErrBackupNotSucceeded.
	//
	// Deprecated: use contract/referents.ErrBackupNotSucceeded; removed in the
	// next release.
	ErrBackupNotSucceeded = referents.ErrBackupNotSucceeded
	// ErrProviderUnsupported is referents.ErrProviderUnsupported.
	//
	// Deprecated: use contract/referents.ErrProviderUnsupported; removed in the
	// next release.
	ErrProviderUnsupported = referents.ErrProviderUnsupported
	// ErrRestorePITRUnsupported is referents.ErrRestorePITRUnsupported.
	//
	// Deprecated: use contract/referents.ErrRestorePITRUnsupported; removed in
	// the next release.
	ErrRestorePITRUnsupported = referents.ErrRestorePITRUnsupported
	// ErrRestorePITRDateRequired is referents.ErrRestorePITRDateRequired.
	//
	// Deprecated: use contract/referents.ErrRestorePITRDateRequired; removed in
	// the next release.
	ErrRestorePITRDateRequired = referents.ErrRestorePITRDateRequired
	// ErrBackupImportUnsupported is referents.ErrBackupImportUnsupported.
	//
	// Deprecated: use contract/referents.ErrBackupImportUnsupported; removed in
	// the next release.
	ErrBackupImportUnsupported = referents.ErrBackupImportUnsupported
	// ErrBackupClassIncompatible is referents.ErrBackupClassIncompatible.
	//
	// Deprecated: use contract/referents.ErrBackupClassIncompatible; removed in
	// the next release.
	ErrBackupClassIncompatible = referents.ErrBackupClassIncompatible
	// ErrBackupStorageIncompatible is referents.ErrBackupStorageIncompatible.
	//
	// Deprecated: use contract/referents.ErrBackupStorageIncompatible; removed
	// in the next release.
	ErrBackupStorageIncompatible = referents.ErrBackupStorageIncompatible
)

// RestoreStreamInstanceName returns the name of the Instance that owns the
// backup stream a point-in-time Restore reads from.
//
// Deprecated: use contract/referents.RestoreStreamInstanceName; removed in the
// next release.
func RestoreStreamInstanceName(restore *backupv1alpha1.Restore) string {
	return referents.RestoreStreamInstanceName(restore)
}

// ValidateBackupSucceeded returns ErrBackupNotSucceeded if the Backup has not
// reached the Succeeded state.
//
// Deprecated: use contract/referents.ValidateBackupSucceeded; removed in the
// next release.
func ValidateBackupSucceeded(backup *backupv1alpha1.Backup) error {
	return referents.ValidateBackupSucceeded(backup)
}

// ValidateBackupImportSupported returns ErrBackupImportUnsupported if the
// BackupClass does not advertise spec.supportsImport.
//
// Deprecated: use contract/referents.ValidateBackupImportSupported; removed in
// the next release.
func ValidateBackupImportSupported(bc *backupv1alpha1.BackupClass) error {
	return referents.ValidateBackupImportSupported(bc)
}

// ValidateRestorePITR checks whether the point-in-time recovery options on a
// Restore are acceptable for the given BackupClass.
//
// Deprecated: use contract/referents.ValidateRestorePITR; removed in the next
// release.
func ValidateRestorePITR(restore *backupv1alpha1.Restore, bc *backupv1alpha1.BackupClass) error {
	return referents.ValidateRestorePITR(restore, bc)
}

// ValidatePITRStorage checks that the storage a point-in-time Restore names
// is carrying a stream on the Instance that owns it.
//
// Deprecated: use contract/referents.ValidatePITRStorage; removed in the next
// release.
func ValidatePITRStorage(pitr *backupv1alpha1.DataSourcePointInTime, instance *corev1alpha1.Instance) error {
	return referents.ValidatePITRStorage(pitr, instance)
}

// ErrSecretSchemaValidation is returned when a secret fails schema
// validation.
//
// Deprecated: use contract/definitions.ErrSecretSchemaValidation; removed in
// the next release.
var ErrSecretSchemaValidation = definitions.ErrSecretSchemaValidation

// ErrConfigMapSchemaValidation is returned when a configmap fails schema
// validation.
//
// Deprecated: use contract/definitions.ErrConfigMapSchemaValidation; removed in
// the next release.
var ErrConfigMapSchemaValidation = definitions.ErrConfigMapSchemaValidation

// ValidateSecretSchema validates a Kubernetes Secret's data or stringData
// against the provider's secret definition its definition label names.
//
// Deprecated: use contract/definitions.ValidateSecretSchema; removed in the
// next release.
func ValidateSecretSchema(secret *corev1.Secret, providerSpec *corev1alpha1.ProviderSpec) error {
	return definitions.ValidateSecretSchema(secret, providerSpec)
}

// ValidateConfigMapSchema validates a Kubernetes ConfigMap's data against
// the provider's configmap definition its definition label names.
//
// Deprecated: use contract/definitions.ValidateConfigMapSchema; removed in the
// next release.
func ValidateConfigMapSchema(configMap *corev1.ConfigMap, providerSpec *corev1alpha1.ProviderSpec) error {
	return definitions.ValidateConfigMapSchema(configMap, providerSpec)
}
