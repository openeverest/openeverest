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
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

// LimitsExceededReason is the reason string used on BackupConfigError and
// the BackupConfigured condition when an Instance violates the limits
// declared by its BackupClass.
const LimitsExceededReason = "LimitsExceeded"

// ErrBackupClassLimitsExceeded is the sentinel returned by
// ValidateInstanceBackupAgainstClass when an Instance violates the limits
// declared by its BackupClass.
var ErrBackupClassLimitsExceeded = errors.New("backup class limits exceeded")

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
					ErrBackupClassLimitsExceeded, s.Name, len(s.Schedules), bc.Name, *limits.MaxSchedulesPerStorage)
			}
		}
	}

	return nil
}
