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

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

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

// mkPITRSchema returns a RawExtension holding an OpenAPI v3 schema fragment
// with a single integer property "oplogSpanMin" (minimum 1) and a string
// enum property "compressionType".
func mkPITRSchema() *runtime.RawExtension {
	return &runtime.RawExtension{Raw: []byte(`{
		"type": "object",
		"properties": {
			"oplogSpanMin": {"type": "integer", "minimum": 1},
			"compressionType": {"type": "string", "enum": ["none", "snappy", "zstd"]}
		}
	}`)}
}

func mkPITRClass(schema *runtime.RawExtension) *backupv1alpha1.BackupClass {
	return &backupv1alpha1.BackupClass{
		ObjectMeta: metav1.ObjectMeta{Name: "bc"},
		Spec: backupv1alpha1.BackupClassSpec{
			ExecutionMode: backupv1alpha1.BackupExecutionModeProviderManaged,
			ProviderManaged: &backupv1alpha1.ProviderManagedSpec{
				SupportsPITR:     true,
				PITRConfigSchema: schema,
			},
		},
	}
}

func withPITRConfig(s corev1alpha1.InstanceBackupStorage, raw string) corev1alpha1.InstanceBackupStorage {
	if s.PITR == nil {
		s.PITR = &corev1alpha1.InstanceBackupStoragePITR{Enabled: true}
	}
	s.PITR.Config = &runtime.RawExtension{Raw: []byte(raw)}
	return s
}

func TestValidateInstanceBackupPITRConfigs(t *testing.T) {
	t.Parallel()

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
			bc:      mkPITRClass(mkPITRSchema()),
			wantErr: false,
		},
		{
			name:    "job-mode class is ignored",
			in:      mkInstance(withPITRConfig(mkStorage("a", true, 0), `{"bogus": true}`)),
			bc:      &backupv1alpha1.BackupClass{Spec: backupv1alpha1.BackupClassSpec{ExecutionMode: backupv1alpha1.BackupExecutionModeJob}},
			wantErr: false,
		},
		{
			name:    "storages without pitr config are skipped",
			in:      mkInstance(mkStorage("a", true, 0), mkStorage("b", false, 0)),
			bc:      mkPITRClass(mkPITRSchema()),
			wantErr: false,
		},
		{
			name:    "valid config passes",
			in:      mkInstance(withPITRConfig(mkStorage("a", true, 0), `{"oplogSpanMin": 10, "compressionType": "zstd"}`)),
			bc:      mkPITRClass(mkPITRSchema()),
			wantErr: false,
		},
		{
			name:    "unknown field is rejected",
			in:      mkInstance(withPITRConfig(mkStorage("a", true, 0), `{"unknownField": 1}`)),
			bc:      mkPITRClass(mkPITRSchema()),
			wantErr: true,
		},
		{
			name:    "constraint violation is rejected",
			in:      mkInstance(withPITRConfig(mkStorage("a", true, 0), `{"oplogSpanMin": 0}`)),
			bc:      mkPITRClass(mkPITRSchema()),
			wantErr: true,
		},
		{
			name:    "enum violation is rejected",
			in:      mkInstance(withPITRConfig(mkStorage("a", true, 0), `{"compressionType": "gzip"}`)),
			bc:      mkPITRClass(mkPITRSchema()),
			wantErr: true,
		},
		{
			name:    "config without declared schema is rejected",
			in:      mkInstance(withPITRConfig(mkStorage("a", true, 0), `{"oplogSpanMin": 10}`)),
			bc:      mkPITRClass(nil),
			wantErr: true,
		},
		{
			name:    "malformed schema on class is rejected",
			in:      mkInstance(withPITRConfig(mkStorage("a", true, 0), `{"oplogSpanMin": 10}`)),
			bc:      mkPITRClass(&runtime.RawExtension{Raw: []byte(`not-json`)}),
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateInstanceBackupPITRConfigs(tc.in, tc.bc)
			if tc.wantErr {
				require.Error(t, err)
				require.ErrorIs(t, err, ErrPITRConfigInvalid)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func mkPITRRestore(pitr *backupv1alpha1.DataSourcePITR) *backupv1alpha1.Restore {
	return &backupv1alpha1.Restore{
		Spec: backupv1alpha1.RestoreSpec{
			InstanceName: "db",
			DataSource: backupv1alpha1.DataSource{
				Type: backupv1alpha1.DataSourceTypeBackup,
				Backup: &backupv1alpha1.DataSourceBackup{
					BackupName: "bkp",
					PITR:       pitr,
				},
			},
		},
	}
}

func TestValidateRestorePITR(t *testing.T) {
	t.Parallel()

	now := metav1.Now()
	supported := mkPITRClass(nil)
	unsupported := &backupv1alpha1.BackupClass{
		ObjectMeta: metav1.ObjectMeta{Name: "bc"},
		Spec: backupv1alpha1.BackupClassSpec{
			ExecutionMode:   backupv1alpha1.BackupExecutionModeProviderManaged,
			ProviderManaged: &backupv1alpha1.ProviderManagedSpec{SupportsPITR: false},
		},
	}
	jobMode := &backupv1alpha1.BackupClass{
		Spec: backupv1alpha1.BackupClassSpec{ExecutionMode: backupv1alpha1.BackupExecutionModeJob},
	}

	tests := []struct {
		name         string
		restore      *backupv1alpha1.Restore
		bc           *backupv1alpha1.BackupClass
		wantErr      bool
		wantUnsupErr bool
	}{
		{
			name:    "no pitr passes",
			restore: mkPITRRestore(nil),
			bc:      unsupported,
			wantErr: false,
		},
		{
			name:    "pitr latest with supporting class passes",
			restore: mkPITRRestore(&backupv1alpha1.DataSourcePITR{Type: backupv1alpha1.PITRTypeLatest}),
			bc:      supported,
			wantErr: false,
		},
		{
			name:    "pitr date with date and supporting class passes",
			restore: mkPITRRestore(&backupv1alpha1.DataSourcePITR{Type: backupv1alpha1.PITRTypeDate, Date: &now}),
			bc:      supported,
			wantErr: false,
		},
		{
			name:    "pitr date without date is rejected",
			restore: mkPITRRestore(&backupv1alpha1.DataSourcePITR{Type: backupv1alpha1.PITRTypeDate}),
			bc:      supported,
			wantErr: true,
		},
		{
			name:         "unsupporting class is rejected",
			restore:      mkPITRRestore(&backupv1alpha1.DataSourcePITR{Type: backupv1alpha1.PITRTypeLatest}),
			bc:           unsupported,
			wantErr:      true,
			wantUnsupErr: true,
		},
		{
			name: "class without providerManaged is rejected",
			restore: mkPITRRestore(&backupv1alpha1.DataSourcePITR{
				Type: backupv1alpha1.PITRTypeLatest,
			}),
			bc: &backupv1alpha1.BackupClass{
				Spec: backupv1alpha1.BackupClassSpec{ExecutionMode: backupv1alpha1.BackupExecutionModeProviderManaged},
			},
			wantErr:      true,
			wantUnsupErr: true,
		},
		{
			name:    "job-mode class is not gated",
			restore: mkPITRRestore(&backupv1alpha1.DataSourcePITR{Type: backupv1alpha1.PITRTypeLatest}),
			bc:      jobMode,
			wantErr: false,
		},
		{
			name:    "nil class passes",
			restore: mkPITRRestore(&backupv1alpha1.DataSourcePITR{Type: backupv1alpha1.PITRTypeLatest}),
			bc:      nil,
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateRestorePITR(tc.restore, tc.bc)
			if tc.wantErr {
				require.Error(t, err)
				if tc.wantUnsupErr {
					require.ErrorIs(t, err, ErrRestorePITRUnsupported)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}
