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

// Package backupclass checks an Instance's backup configuration against the
// BackupClass it names, with Instance field paths, for the reconciler and the
// admission webhook alike, and judges the classes a provider claims.
package backupclass

import (
	"fmt"

	"k8s.io/apimachinery/pkg/util/validation/field"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	apicommon "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

// ProviderVerdict returns the compatibility of bc with provider that core
// publishes in bc's status.providers for bc's current generation (spec 019
// §4.2). A level other than Compatible and Incompatible, such as one a later
// release adds, is reported as Pending: only a terminal verdict rejects.
func ProviderVerdict(bc *backupv1alpha1.BackupClass, provider string) backupv1alpha1.BackupClassProviderCompatibility {
	v := backupv1alpha1.CurrentProviderVerdict(bc, provider)
	if v.Compatibility != backupv1alpha1.CompatibilityCompatible && v.Compatibility != backupv1alpha1.CompatibilityIncompatible {
		v.Compatibility = backupv1alpha1.CompatibilityPending
	}
	return v
}

// VerdictDetail returns the reason and message of a verdict for messages.
func VerdictDetail(v backupv1alpha1.BackupClassProviderCompatibility) string {
	if v.Message == "" {
		return v.Reason
	}
	return v.Reason + ": " + v.Message
}

// LimitErrors returns the limits bc declares that the backup configuration of
// in exceeds. It returns nothing when in has backups off or bc is not
// ProviderManaged.
func LimitErrors(in *corev1alpha1.Instance, bc *backupv1alpha1.BackupClass) field.ErrorList {
	if !applies(in, bc) || bc.Spec.ProviderManaged == nil || bc.Spec.ProviderManaged.Limits == nil {
		return nil
	}
	limits := bc.Spec.ProviderManaged.Limits
	storages := in.Spec.Backup.Storages
	storagesPath := field.NewPath("spec", "backup", "storages")

	var errs field.ErrorList
	if limits.MaxStorages != nil && len(storages) > int(*limits.MaxStorages) {
		errs = append(errs, field.TooMany(storagesPath, len(storages), int(*limits.MaxStorages)))
	}
	if limits.MaxPITREnabledStorages != nil {
		var pitr int
		for _, s := range storages {
			if s.PITR != nil && s.PITR.Enabled {
				pitr++
			}
		}
		if pitr > int(*limits.MaxPITREnabledStorages) {
			errs = append(errs, field.Invalid(storagesPath, field.OmitValueType{}, fmt.Sprintf(
				"%d storages enable PITR; backup class %s allows at most %d",
				pitr, bc.Name, *limits.MaxPITREnabledStorages,
			)))
		}
	}
	if limits.MaxSchedulesPerStorage != nil {
		for i, s := range storages {
			if len(s.Schedules) > int(*limits.MaxSchedulesPerStorage) {
				errs = append(errs, field.TooMany(
					storagesPath.Index(i).Child("schedules"), len(s.Schedules), int(*limits.MaxSchedulesPerStorage),
				))
			}
		}
	}
	return errs
}

// PITRErrors returns the per-storage PITR parameters that the
// pitrParametersSchema of bc rejects. PITR parameters on a class without a
// schema are rejected. It returns nothing when in has backups off or bc is
// not ProviderManaged.
func PITRErrors(in *corev1alpha1.Instance, bc *backupv1alpha1.BackupClass) field.ErrorList {
	if !applies(in, bc) {
		return nil
	}
	var schema *apicommon.ParametersSchema
	if bc.Spec.ProviderManaged != nil {
		schema = bc.Spec.ProviderManaged.PITRParametersSchema
	}

	var errs field.ErrorList
	storagesPath := field.NewPath("spec", "backup", "storages")
	for i, s := range in.Spec.Backup.Storages {
		if s.PITR == nil || s.PITR.Parameters == nil || len(s.PITR.Parameters.Raw) == 0 {
			continue
		}
		if err := schema.Validate(s.PITR.Parameters); err != nil {
			errs = append(errs, field.Invalid(
				storagesPath.Index(i).Child("pitr", "parameters"), field.OmitValueType{}, err.Error(),
			))
		}
	}
	return errs
}

func applies(in *corev1alpha1.Instance, bc *backupv1alpha1.BackupClass) bool {
	return in != nil && in.Spec.Backup != nil && in.Spec.Backup.Enabled &&
		bc != nil && bc.Spec.ExecutionMode == backupv1alpha1.BackupExecutionModeProviderManaged
}
