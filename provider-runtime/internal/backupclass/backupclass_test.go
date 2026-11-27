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

package backupclass

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	apicommon "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

func class(limits *backupv1alpha1.BackupClassLimits) *backupv1alpha1.BackupClass {
	return &backupv1alpha1.BackupClass{
		ObjectMeta: metav1.ObjectMeta{Name: "pbm"},
		Spec: backupv1alpha1.BackupClassSpec{
			ExecutionMode:   backupv1alpha1.BackupExecutionModeProviderManaged,
			ProviderManaged: &backupv1alpha1.ProviderManagedSpec{ProviderRef: apicommon.ObjectRef{Name: "mongo"}, Limits: limits},
		},
	}
}

func instance(storages ...corev1alpha1.InstanceBackupStorage) *corev1alpha1.Instance {
	return &corev1alpha1.Instance{Spec: corev1alpha1.InstanceSpec{
		ProviderRef: apicommon.ObjectRef{Name: "mongo"},
		Backup:      &corev1alpha1.InstanceBackupSpec{Enabled: true, Storages: storages},
	}}
}

func withPITR(raw string) corev1alpha1.InstanceBackupStorage {
	s := corev1alpha1.InstanceBackupStorage{PITR: &corev1alpha1.InstanceBackupStoragePITR{Enabled: true}}
	if raw != "" {
		s.PITR.Parameters = &runtime.RawExtension{Raw: []byte(raw)}
	}
	return s
}

func fields(errs field.ErrorList) []string {
	if len(errs) == 0 {
		return nil
	}
	out := make([]string, 0, len(errs))
	for _, err := range errs {
		out = append(out, err.Field)
	}
	return out
}

func TestProviderVerdict(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		entry  backupv1alpha1.BackupClassProviderCompatibility
		want   backupv1alpha1.Compatibility
		detail string
	}{
		{
			name:  "compatible",
			entry: backupv1alpha1.BackupClassProviderCompatibility{Compatibility: backupv1alpha1.CompatibilityCompatible},
			want:  backupv1alpha1.CompatibilityCompatible,
		},
		{
			name: "incompatible",
			entry: backupv1alpha1.BackupClassProviderCompatibility{
				Compatibility: backupv1alpha1.CompatibilityIncompatible, Reason: backupv1alpha1.ReasonSpecDrift, Message: "spec.supportsImport",
			},
			want:   backupv1alpha1.CompatibilityIncompatible,
			detail: "SpecDrift: spec.supportsImport",
		},
		{
			name:   "pending",
			entry:  backupv1alpha1.BackupClassProviderCompatibility{Compatibility: backupv1alpha1.CompatibilityPending, Reason: backupv1alpha1.ReasonClassNotAccepted},
			want:   backupv1alpha1.CompatibilityPending,
			detail: backupv1alpha1.ReasonClassNotAccepted,
		},
		{
			name:   "a level this release does not know",
			entry:  backupv1alpha1.BackupClassProviderCompatibility{Compatibility: "BackupOnly", Reason: "RestoreUnsupported"},
			want:   backupv1alpha1.CompatibilityPending,
			detail: "RestoreUnsupported",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			bc := class(nil)
			bc.Generation = 3
			tc.entry.Name, tc.entry.ClassGeneration = "mongo", 3
			bc.Status.Providers = []backupv1alpha1.BackupClassProviderCompatibility{tc.entry}
			v := ProviderVerdict(bc, "mongo")
			assert.Equal(t, tc.want, v.Compatibility)
			assert.Equal(t, tc.detail, VerdictDetail(v))
		})
	}
}

func TestLimitErrors(t *testing.T) {
	t.Parallel()
	one := new(int32(1))
	a := corev1alpha1.InstanceBackupStorage{Schedules: make([]corev1alpha1.InstanceBackupSchedule, 2)}
	b := corev1alpha1.InstanceBackupStorage{}

	tests := []struct {
		name   string
		in     *corev1alpha1.Instance
		limits *backupv1alpha1.BackupClassLimits
		want   []string
	}{
		{name: "no limits", in: instance(a, b)},
		{name: "storages", in: instance(a, b), limits: &backupv1alpha1.BackupClassLimits{MaxStorages: one}, want: []string{"spec.backup.storages"}},
		{
			name:   "PITR storages",
			in:     instance(withPITR(""), withPITR("")),
			limits: &backupv1alpha1.BackupClassLimits{MaxPITREnabledStorages: one},
			want:   []string{"spec.backup.storages"},
		},
		{
			name:   "schedules per storage",
			in:     instance(b, a),
			limits: &backupv1alpha1.BackupClassLimits{MaxSchedulesPerStorage: one},
			want:   []string{"spec.backup.storages[1].schedules"},
		},
		{
			name:   "schedules without storage",
			in:     withSchedules(instance(), 2),
			limits: &backupv1alpha1.BackupClassLimits{MaxSchedulesPerStorage: one},
			want:   []string{"spec.backup.schedules"},
		},
		{
			name:   "backups off",
			in:     &corev1alpha1.Instance{Spec: corev1alpha1.InstanceSpec{Backup: &corev1alpha1.InstanceBackupSpec{}}},
			limits: &backupv1alpha1.BackupClassLimits{MaxStorages: new(int32(0))},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, fields(LimitErrors(tc.in, class(tc.limits))))
		})
	}
}

func withSchedules(in *corev1alpha1.Instance, n int) *corev1alpha1.Instance {
	in.Spec.Backup.Schedules = make([]corev1alpha1.InstanceBackupSchedule, n)
	return in
}

func TestStorageModeErrors(t *testing.T) {
	t.Parallel()
	snapshots := class(nil)
	snapshots.Spec.StorageMode = backupv1alpha1.BackupStorageModeNone
	required := class(nil)
	required.Spec.StorageMode = backupv1alpha1.BackupStorageModeRequired
	storage := corev1alpha1.InstanceBackupStorage{}

	tests := []struct {
		name  string
		in    *corev1alpha1.Instance
		class *backupv1alpha1.BackupClass
		want  []string
	}{
		{name: "storages with a class that needs storage", in: instance(storage), class: required},
		{name: "unset storageMode needs storage", in: withSchedules(instance(), 1), class: class(nil), want: []string{"spec.backup.schedules"}},
		{name: "schedules with a class that needs storage", in: withSchedules(instance(storage), 1), class: required, want: []string{"spec.backup.schedules"}},
		{name: "schedules with a class without storage", in: withSchedules(instance(), 1), class: snapshots},
		{name: "storages with a class without storage", in: instance(storage), class: snapshots, want: []string{"spec.backup.storages"}},
		{
			name:  "backups off",
			in:    &corev1alpha1.Instance{Spec: corev1alpha1.InstanceSpec{Backup: &corev1alpha1.InstanceBackupSpec{Storages: []corev1alpha1.InstanceBackupStorage{storage}}}},
			class: snapshots,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, fields(StorageModeErrors(tc.in, tc.class)))
		})
	}
}

func TestPITRErrors(t *testing.T) {
	t.Parallel()
	assert.Equal(t,
		[]string{"spec.backup.storages[1].pitr.parameters"},
		fields(PITRErrors(instance(withPITR(""), withPITR(`{"interval":"10m"}`)), class(nil))),
		"PITR parameters on a class without a schema",
	)
}
