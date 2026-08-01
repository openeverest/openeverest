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
	"errors"
	"fmt"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	apicommon "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

// LimitsExceededReason is the reason string used on BackupConfigError and
// the BackupConfigured condition when an Instance violates the limits
// declared by its BackupClass.
const LimitsExceededReason = "LimitsExceeded"

// PITRConfigInvalidReason is the reason string used on BackupConfigError and
// the BackupConfigured condition when a per-storage PITR config on an
// Instance does not conform to the schema declared by its BackupClass.
const PITRConfigInvalidReason = "PITRConfigInvalid"

// ErrBackupClassLimitsExceeded is the sentinel returned by
// ValidateInstanceBackupAgainstClass when an Instance violates the limits
// declared by its BackupClass.
var ErrBackupClassLimitsExceeded = errors.New("backup class limits exceeded")

// ErrPITRConfigInvalid is the sentinel returned by
// ValidateInstanceBackupPITRParameters when per-storage PITR parameters do not
// conform to the BackupClass's providerManaged.pitrParametersSchema.
var ErrPITRConfigInvalid = errors.New("PITR config invalid")

// ErrInvalidReference is the umbrella sentinel every reference-validation
// error below satisfies via errors.Is. Callers (e.g. the API server's
// CreateBackup/CreateRestore validation handlers) check errors.Is(err,
// ErrInvalidReference) once to classify a failure as a client-facing
// validation error, instead of enumerating every individual sentinel — so
// any reference-validation error added in the future is automatically
// classified correctly without touching that call site.
var ErrInvalidReference = errors.New("invalid reference")

// referenceError is a reference-validation sentinel. It implements Is so
// that every referenceError value satisfies errors.Is(err, ErrInvalidReference),
// in addition to satisfying errors.Is against itself.
type referenceError string

// Error implements the error interface.
func (e referenceError) Error() string {
	return string(e)
}

// Is reports whether target is ErrInvalidReference, making every
// referenceError a member of that umbrella sentinel.
func (e referenceError) Is(target error) bool {
	return target == ErrInvalidReference
}

const (
	// ErrBackupNotFound is a sentinel callers may wrap when their own lookup for
	// a referenced Backup returns not-found, so the failure can be classified
	// via errors.Is without duplicating the message across call sites. This
	// package performs no lookups itself; callers own fetching and NotFound
	// classification.
	ErrBackupNotFound referenceError = "backup not found"

	// ErrBackupClassNotFound is a sentinel callers may wrap when their own
	// lookup for a referenced BackupClass returns not-found.
	ErrBackupClassNotFound referenceError = "backup class not found"

	// ErrInstanceNotFound is a sentinel callers may wrap when their own lookup
	// for a referenced Instance returns not-found.
	ErrInstanceNotFound referenceError = "instance not found"

	// ErrBackupStorageNotFound is a sentinel callers may wrap when their own
	// lookup for a referenced BackupStorage returns not-found.
	ErrBackupStorageNotFound referenceError = "backup storage not found"

	// ErrBackupNotSucceeded is the sentinel returned by ValidateBackupSucceeded
	// when a Backup exists but has not reached the Succeeded state.
	ErrBackupNotSucceeded referenceError = "backup not succeeded"

	// ErrProviderUnsupported is the sentinel returned by
	// ValidateClassSupportsProvider when a BackupClass does not list a provider
	// in its SupportedProviders.
	ErrProviderUnsupported referenceError = "provider not supported by backup class"

	// ErrRestorePITRUnsupported is the sentinel returned by ValidateRestorePITR
	// when a Restore requests point-in-time recovery but the resolved
	// ProviderManaged BackupClass does not advertise supportsPITR.
	ErrRestorePITRUnsupported referenceError = "point-in-time recovery is not supported"

	// ErrRestorePITRDateRequired is the sentinel returned by ValidateRestorePITR
	// when a Restore requests date-based point-in-time recovery without
	// specifying spec.dataSource.backup.pitr.date.
	ErrRestorePITRDateRequired referenceError = "point-in-time recovery date is required"
)

// ValidateInstanceBackupAgainstClass enforces the generic limits declared on
// a ProviderManaged BackupClass against an Instance's backup configuration.
// It is safe to call with any combination of nil inputs:
//   - If the Instance has no backup config, no class is selected, or the
//     class is not ProviderManaged, the function is a no-op and returns nil.
//   - If the class has no limits set, the function is a no-op and returns nil.
//
// Engine-specific constraints (e.g. PSMDB's single PITR stream) are NOT
// enforced here; providers add those checks in their own Validate() and
// ConfigureBackup() implementations, typically by calling
// Context.BackupClassLimits() and adding extra rules on top.
func ValidateInstanceBackupAgainstClass(in *corev1alpha1.Instance, bc *backupv1alpha1.BackupClass) error {
	if in == nil || in.Spec.Backup == nil || !in.Spec.Backup.Enabled {
		return nil
	}
	if bc == nil || bc.Spec.ExecutionMode != backupv1alpha1.BackupExecutionModeProviderManaged {
		return nil
	}
	if bc.Spec.ProviderManaged == nil || bc.Spec.ProviderManaged.Limits == nil {
		return nil
	}
	limits := bc.Spec.ProviderManaged.Limits
	storages := in.Spec.Backup.Storages

	if limits.MaxStorages != nil && int32(len(storages)) > *limits.MaxStorages {
		return fmt.Errorf("%w: spec.backup.storages has %d entries, BackupClass %q allows at most %d",
			ErrBackupClassLimitsExceeded, len(storages), bc.Name, *limits.MaxStorages)
	}

	if limits.MaxPITREnabledStorages != nil {
		var pitrCount int32
		for _, s := range storages {
			if s.PITR != nil && s.PITR.Enabled {
				pitrCount++
			}
		}
		if pitrCount > *limits.MaxPITREnabledStorages {
			return fmt.Errorf("%w: %d storages have PITR enabled, BackupClass %q allows at most %d",
				ErrBackupClassLimitsExceeded, pitrCount, bc.Name, *limits.MaxPITREnabledStorages)
		}
	}

	if limits.MaxSchedulesPerStorage != nil {
		for _, s := range storages {
			if int32(len(s.Schedules)) > *limits.MaxSchedulesPerStorage {
				return fmt.Errorf("%w: storage %q has %d schedules, BackupClass %q allows at most %d per storage",
					ErrBackupClassLimitsExceeded, s.StorageRef.Name, len(s.Schedules), bc.Name, *limits.MaxSchedulesPerStorage)
			}
		}
	}

	return nil
}

// ValidateInstanceBackupPITRParameters validates each per-storage PITR
// parameters payload on the Instance (.spec.backup.storages[].pitr.parameters)
// against the OpenAPI v3 schema declared by the BackupClass under
// .spec.providerManaged.pitrParametersSchema.
//
// It is safe to call with any combination of nil inputs:
//   - If the Instance has no backup config, no class is selected, or the
//     class is not ProviderManaged, the function is a no-op and returns nil.
//   - Storages without PITR parameters are skipped.
//
// PITR parameters on a class that declares no pitrParametersSchema are
// rejected, mirroring ParametersSchema.Validate's behavior for backup and
// restore parameters.
func ValidateInstanceBackupPITRParameters(in *corev1alpha1.Instance, bc *backupv1alpha1.BackupClass) error {
	if in == nil || in.Spec.Backup == nil || !in.Spec.Backup.Enabled {
		return nil
	}
	if bc == nil || bc.Spec.ExecutionMode != backupv1alpha1.BackupExecutionModeProviderManaged {
		return nil
	}

	var schema *apicommon.ParametersSchema
	if bc.Spec.ProviderManaged != nil {
		schema = bc.Spec.ProviderManaged.PITRParametersSchema
	}

	for _, s := range in.Spec.Backup.Storages {
		if s.PITR == nil || s.PITR.Parameters == nil || len(s.PITR.Parameters.Raw) == 0 {
			continue
		}
		if err := schema.Validate(s.PITR.Parameters); err != nil {
			return fmt.Errorf("%w: storage %q: %s", ErrPITRConfigInvalid, s.StorageRef.Name, err.Error())
		}
	}
	return nil
}

// ValidateBackupSucceeded returns ErrBackupNotSucceeded unless the Backup has
// reached the Succeeded state. Callers are responsible for fetching the
// Backup themselves and handling a not-found lookup error with their own
// context before calling this.
func ValidateBackupSucceeded(backup *backupv1alpha1.Backup) error {
	if backup.Status.State != backupv1alpha1.BackupStateSucceeded {
		state := backup.Status.State
		if state == "" {
			// The controller has not reconciled the Backup yet; report the
			// state a user would recognize instead of an empty string.
			state = backupv1alpha1.BackupStatePending
		}
		return fmt.Errorf("%w: '%s' is in state '%s'", ErrBackupNotSucceeded, backup.GetName(), state)
	}
	return nil
}

// ValidateClassSupportsProvider returns ErrProviderUnsupported unless bc
// lists provider in its SupportedProviders. Callers are responsible for
// fetching the BackupClass themselves.
func ValidateClassSupportsProvider(bc *backupv1alpha1.BackupClass, provider string) error {
	if !bc.Spec.SupportedProviders.Has(provider) {
		return fmt.Errorf("%w: class '%s', provider '%s'", ErrProviderUnsupported, bc.GetName(), provider)
	}
	return nil
}

// ValidateRestorePITR checks whether the point-in-time recovery options on a
// Restore are acceptable for the given BackupClass:
//   - When the resolved class is ProviderManaged, PITR may only be requested
//     if the class advertises .spec.providerManaged.supportsPITR.
//   - Job-mode classes have no PITR capability declaration; they are not
//     gated here.
//
// A nil class or a Restore without PITR options passes. The CRD's CEL rule
// already requires date when type is "date"; the check is repeated here for
// defense in depth on paths that bypass admission.
func ValidateRestorePITR(restore *backupv1alpha1.Restore, bc *backupv1alpha1.BackupClass) error {
	if restore == nil || restore.Spec.DataSource.Backup == nil || restore.Spec.DataSource.Backup.PITR == nil {
		return nil
	}
	pitr := restore.Spec.DataSource.Backup.PITR
	if pitr.Type == backupv1alpha1.PITRTypeDate && pitr.Date == nil {
		return fmt.Errorf(
			"%w: spec.dataSource.backup.pitr.date must be set when type is %q",
			ErrRestorePITRDateRequired, backupv1alpha1.PITRTypeDate,
		)
	}
	if bc == nil || bc.Spec.ExecutionMode != backupv1alpha1.BackupExecutionModeProviderManaged {
		return nil
	}
	if bc.Spec.ProviderManaged == nil || !bc.Spec.ProviderManaged.SupportsPITR {
		return fmt.Errorf("%w: BackupClass %q", ErrRestorePITRUnsupported, bc.Name)
	}
	return nil
}
