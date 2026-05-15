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
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

func ptrInt32(v int32) *int32 { return &v }

func mkStorage(name string, pitr bool, schedules int) corev1alpha1.InstanceBackupStorage {
	s := corev1alpha1.InstanceBackupStorage{
		Name:       name,
		StorageRef: corev1.LocalObjectReference{Name: name},
	}
	if pitr {
		s.PITR = &corev1alpha1.InstanceBackupStoragePITR{Enabled: true}
	}
	for i := 0; i < schedules; i++ {
		s.Schedules = append(s.Schedules, corev1alpha1.InstanceBackupSchedule{
			Name:    name + "-sched",
			Enabled: true,
			Cron:    "0 * * * *",
		})
	}
	return s
}

func mkInstance(storages ...corev1alpha1.InstanceBackupStorage) *corev1alpha1.Instance {
	return &corev1alpha1.Instance{
		Spec: corev1alpha1.InstanceSpec{
			Backup: &corev1alpha1.InstanceBackupSpec{
				Enabled:  true,
				ClassRef: corev1alpha1.BackupClassReference{Name: "bc"},
				Storages: storages,
			},
		},
	}
}

func mkClass(limits *backupv1alpha1.BackupClassLimits) *backupv1alpha1.BackupClass {
	return &backupv1alpha1.BackupClass{
		ObjectMeta: metav1.ObjectMeta{Name: "bc"},
		Spec: backupv1alpha1.BackupClassSpec{
			ExecutionMode: backupv1alpha1.BackupExecutionModeProviderManaged,
			ProviderManaged: &backupv1alpha1.ProviderManagedSpec{
				Limits: limits,
			},
		},
	}
}

func TestValidateInstanceBackupAgainstClass(t *testing.T) {
	tests := []struct {
		name    string
		in      *corev1alpha1.Instance
		bc      *backupv1alpha1.BackupClass
		wantErr bool
	}{
		{
			name:    "nil inputs are no-ops",
			in:      nil,
			bc:      nil,
			wantErr: false,
		},
		{
			name:    "backup disabled is a no-op",
			in:      &corev1alpha1.Instance{Spec: corev1alpha1.InstanceSpec{Backup: &corev1alpha1.InstanceBackupSpec{Enabled: false}}},
			bc:      mkClass(&backupv1alpha1.BackupClassLimits{MaxStorages: ptrInt32(1)}),
			wantErr: false,
		},
		{
			name:    "job-mode class is ignored",
			in:      mkInstance(mkStorage("a", false, 0), mkStorage("b", false, 0)),
			bc:      &backupv1alpha1.BackupClass{Spec: backupv1alpha1.BackupClassSpec{ExecutionMode: backupv1alpha1.BackupExecutionModeJob}},
			wantErr: false,
		},
		{
			name:    "unset limits = unlimited",
			in:      mkInstance(mkStorage("a", true, 5), mkStorage("b", true, 5)),
			bc:      mkClass(nil),
			wantErr: false,
		},
		{
			name:    "max storages violated",
			in:      mkInstance(mkStorage("a", false, 0), mkStorage("b", false, 0)),
			bc:      mkClass(&backupv1alpha1.BackupClassLimits{MaxStorages: ptrInt32(1)}),
			wantErr: true,
		},
		{
			name:    "max storages satisfied",
			in:      mkInstance(mkStorage("a", false, 0)),
			bc:      mkClass(&backupv1alpha1.BackupClassLimits{MaxStorages: ptrInt32(1)}),
			wantErr: false,
		},
		{
			name:    "max pitr storages violated",
			in:      mkInstance(mkStorage("a", true, 0), mkStorage("b", true, 0)),
			bc:      mkClass(&backupv1alpha1.BackupClassLimits{MaxPITREnabledStorages: ptrInt32(1)}),
			wantErr: true,
		},
		{
			name:    "max pitr storages satisfied",
			in:      mkInstance(mkStorage("a", true, 0), mkStorage("b", false, 0)),
			bc:      mkClass(&backupv1alpha1.BackupClassLimits{MaxPITREnabledStorages: ptrInt32(1)}),
			wantErr: false,
		},
		{
			name:    "max schedules per storage violated",
			in:      mkInstance(mkStorage("a", false, 3)),
			bc:      mkClass(&backupv1alpha1.BackupClassLimits{MaxSchedulesPerStorage: ptrInt32(2)}),
			wantErr: true,
		},
		{
			name:    "max schedules per storage satisfied",
			in:      mkInstance(mkStorage("a", false, 2), mkStorage("b", false, 1)),
			bc:      mkClass(&backupv1alpha1.BackupClassLimits{MaxSchedulesPerStorage: ptrInt32(2)}),
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateInstanceBackupAgainstClass(tc.in, tc.bc)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if !errors.Is(err, ErrBackupClassLimitsExceeded) {
					t.Fatalf("expected ErrBackupClassLimitsExceeded, got %v", err)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
