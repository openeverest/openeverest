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

package referents_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/contract/referents"
)

// otherMode is an execution mode ValidateRestorePITR does not gate.
const otherMode = backupv1alpha1.BackupExecutionMode("Other")

// The API server answers with a client error for every error that satisfies
// ErrInvalidReference, so each sentinel must, even wrapped.
func TestSentinelsAreInvalidReferences(t *testing.T) {
	t.Parallel()

	for _, sentinel := range []error{
		referents.ErrBackupNotFound,
		referents.ErrBackupClassNotFound,
		referents.ErrInstanceNotFound,
		referents.ErrBackupStorageNotFound,
		referents.ErrBackupNotSucceeded,
		referents.ErrProviderUnsupported,
		referents.ErrRestorePITRUnsupported,
		referents.ErrRestorePITRDateRequired,
		referents.ErrBackupImportUnsupported,
		referents.ErrBackupClassIncompatible,
		referents.ErrBackupStorageIncompatible,
	} {
		t.Run(sentinel.Error(), func(t *testing.T) {
			t.Parallel()

			err := fmt.Errorf("%w: 'b'", sentinel)
			require.ErrorIs(t, err, referents.ErrInvalidReference)
			require.ErrorIs(t, err, sentinel)
		})
	}
}

func TestRestoreStreamInstanceName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		pitr *backupv1alpha1.DataSourcePointInTime
		want string
	}{
		{name: "restore from a backup reads the target", want: "db"},
		{name: "stream of the target", pitr: &backupv1alpha1.DataSourcePointInTime{}, want: "db"},
		{
			name: "stream of another instance",
			pitr: &backupv1alpha1.DataSourcePointInTime{Source: backupv1alpha1.StreamSource{InstanceRef: &common.ObjectRef{Name: "source"}}},
			want: "source",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, referents.RestoreStreamInstanceName(mkPITRRestore(tc.pitr)))
		})
	}
}

func TestValidatePITRStorage(t *testing.T) {
	t.Parallel()

	storage := func(name string, pitr bool) corev1alpha1.InstanceBackupStorage {
		return corev1alpha1.InstanceBackupStorage{
			StorageRef: common.ObjectRef{Name: name},
			PITR:       &corev1alpha1.InstanceBackupStoragePITR{Enabled: pitr},
		}
	}
	instance := func(storages ...corev1alpha1.InstanceBackupStorage) *corev1alpha1.Instance {
		return &corev1alpha1.Instance{
			ObjectMeta: metav1.ObjectMeta{Name: "db"},
			Spec:       corev1alpha1.InstanceSpec{Backup: &corev1alpha1.InstanceBackupSpec{Storages: storages}},
		}
	}
	pitrFrom := &backupv1alpha1.DataSourcePointInTime{Source: backupv1alpha1.StreamSource{StorageRef: common.ObjectRef{Name: "s3"}}}

	tests := []struct {
		name     string
		pitr     *backupv1alpha1.DataSourcePointInTime
		instance *corev1alpha1.Instance
		wantText string
	}{
		{name: "no point-in-time recovery passes", instance: instance()},
		{name: "a PITR-enabled storage passes", pitr: pitrFrom, instance: instance(storage("gcs", true), storage("s3", true))},
		{name: "no backup configuration", pitr: pitrFrom, instance: &corev1alpha1.Instance{}, wantText: "instance has no backup configuration"},
		{name: "no PITR-enabled storage", pitr: pitrFrom, instance: instance(storage("s3", false)), wantText: `instance "db" has no storage with`},
		{name: "another storage is PITR-enabled", pitr: pitrFrom, instance: instance(storage("s3", false), storage("gcs", true)), wantText: `storage "s3" is not a PITR-enabled storage of instance "db" (PITR-enabled: [gcs])`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := referents.ValidatePITRStorage(tc.pitr, tc.instance)
			if tc.wantText == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, referents.ErrRestorePITRUnsupported)
			require.ErrorContains(t, err, tc.wantText)
		})
	}
}

func TestValidateBackupSucceeded(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		backup   *backupv1alpha1.Backup
		wantErr  bool
		wantText string
	}{
		{
			name:   "succeeded passes",
			backup: &backupv1alpha1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "b"}, Status: backupv1alpha1.BackupStatus{State: backupv1alpha1.BackupStateSucceeded}},
		},
		{
			name:    "running is rejected",
			backup:  &backupv1alpha1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "b"}, Status: backupv1alpha1.BackupStatus{State: backupv1alpha1.BackupStateRunning}},
			wantErr: true,
		},
		{
			name:     "empty state is reported as pending",
			backup:   &backupv1alpha1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "b"}, Status: backupv1alpha1.BackupStatus{State: ""}},
			wantErr:  true,
			wantText: "'b' is in state 'Pending'",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := referents.ValidateBackupSucceeded(tc.backup)
			if tc.wantErr {
				require.ErrorIs(t, err, referents.ErrBackupNotSucceeded)
				if tc.wantText != "" {
					require.ErrorContains(t, err, tc.wantText)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func mkPITRRestore(pitr *backupv1alpha1.DataSourcePointInTime) *backupv1alpha1.Restore {
	ds := backupv1alpha1.DataSource{
		Type: backupv1alpha1.DataSourceTypeBackup,
		Backup: &backupv1alpha1.DataSourceBackup{
			BackupRef: common.ObjectRef{Name: "bkp"},
		},
	}
	if pitr != nil {
		ds = backupv1alpha1.DataSource{
			Type:        backupv1alpha1.DataSourceTypePointInTime,
			PointInTime: pitr,
		}
	}
	return &backupv1alpha1.Restore{
		Spec: backupv1alpha1.RestoreSpec{
			InstanceRef: common.ObjectRef{Name: "db"},
			DataSource:  ds,
		},
	}
}

func TestValidateRestorePITR(t *testing.T) {
	t.Parallel()

	now := metav1.Now()
	supported := &backupv1alpha1.BackupClass{
		ObjectMeta: metav1.ObjectMeta{Name: "bc"},
		Spec: backupv1alpha1.BackupClassSpec{
			ExecutionMode:   backupv1alpha1.BackupExecutionModeProviderManaged,
			ProviderManaged: &backupv1alpha1.ProviderManagedSpec{SupportsPITR: true},
		},
	}
	unsupported := &backupv1alpha1.BackupClass{
		ObjectMeta: metav1.ObjectMeta{Name: "bc"},
		Spec: backupv1alpha1.BackupClassSpec{
			ExecutionMode:   backupv1alpha1.BackupExecutionModeProviderManaged,
			ProviderManaged: &backupv1alpha1.ProviderManagedSpec{SupportsPITR: false},
		},
	}
	otherModeClass := &backupv1alpha1.BackupClass{
		Spec: backupv1alpha1.BackupClassSpec{ExecutionMode: otherMode},
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
			restore: mkPITRRestore(&backupv1alpha1.DataSourcePointInTime{RecoveryTarget: backupv1alpha1.RecoveryTargetLatest}),
			bc:      supported,
			wantErr: false,
		},
		{
			name:    "pitr date with date and supporting class passes",
			restore: mkPITRRestore(&backupv1alpha1.DataSourcePointInTime{RecoveryTarget: backupv1alpha1.RecoveryTargetDate, Date: &now}),
			bc:      supported,
			wantErr: false,
		},
		{
			name:    "pitr date without date is rejected",
			restore: mkPITRRestore(&backupv1alpha1.DataSourcePointInTime{RecoveryTarget: backupv1alpha1.RecoveryTargetDate}),
			bc:      supported,
			wantErr: true,
		},
		{
			name:         "unsupporting class is rejected",
			restore:      mkPITRRestore(&backupv1alpha1.DataSourcePointInTime{RecoveryTarget: backupv1alpha1.RecoveryTargetLatest}),
			bc:           unsupported,
			wantErr:      true,
			wantUnsupErr: true,
		},
		{
			name: "class without providerManaged is rejected",
			restore: mkPITRRestore(&backupv1alpha1.DataSourcePointInTime{
				RecoveryTarget: backupv1alpha1.RecoveryTargetLatest,
			}),
			bc: &backupv1alpha1.BackupClass{
				Spec: backupv1alpha1.BackupClassSpec{ExecutionMode: backupv1alpha1.BackupExecutionModeProviderManaged},
			},
			wantErr:      true,
			wantUnsupErr: true,
		},
		{
			name:    "class of another execution mode is not gated",
			restore: mkPITRRestore(&backupv1alpha1.DataSourcePointInTime{RecoveryTarget: backupv1alpha1.RecoveryTargetLatest}),
			bc:      otherModeClass,
			wantErr: false,
		},
		{
			name:    "nil class passes",
			restore: mkPITRRestore(&backupv1alpha1.DataSourcePointInTime{RecoveryTarget: backupv1alpha1.RecoveryTargetLatest}),
			bc:      nil,
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := referents.ValidateRestorePITR(tc.restore, tc.bc)
			if tc.wantErr {
				require.Error(t, err)
				if tc.wantUnsupErr {
					require.ErrorIs(t, err, referents.ErrRestorePITRUnsupported)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}
