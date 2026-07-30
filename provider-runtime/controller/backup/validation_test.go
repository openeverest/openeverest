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

package backup

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
)

func mkPITRClass(schema *common.ParametersSchema) *backupv1alpha1.BackupClass {
	return &backupv1alpha1.BackupClass{
		ObjectMeta: metav1.ObjectMeta{Name: "bc"},
		Spec: backupv1alpha1.BackupClassSpec{
			ExecutionMode: backupv1alpha1.BackupExecutionModeProviderManaged,
			ProviderManaged: &backupv1alpha1.ProviderManagedSpec{
				SupportsPITR:         true,
				PITRParametersSchema: schema,
			},
		},
	}
}

func mkPITRRestore(pitr *backupv1alpha1.DataSourcePITR) *backupv1alpha1.Restore {
	return &backupv1alpha1.Restore{
		Spec: backupv1alpha1.RestoreSpec{
			InstanceRef: common.ObjectRef{Name: "db"},
			DataSource: backupv1alpha1.DataSource{
				Type: backupv1alpha1.DataSourceTypeBackup,
				Backup: &backupv1alpha1.DataSourceBackup{
					BackupRef: common.ObjectRef{Name: "bkp"},
					PITR:      pitr,
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

func TestResolveSucceededBackup(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	notFoundErr := notFoundError{}

	succeeded := &backupv1alpha1.Backup{
		ObjectMeta: metav1.ObjectMeta{Name: "b"},
		Status:     backupv1alpha1.BackupStatus{State: backupv1alpha1.BackupStateSucceeded},
	}
	running := &backupv1alpha1.Backup{
		ObjectMeta: metav1.ObjectMeta{Name: "b"},
		Status:     backupv1alpha1.BackupStatus{State: backupv1alpha1.BackupStateRunning},
	}

	tests := []struct {
		name       string
		get        GetBackupFunc
		wantErr    error
		wantBackup bool
	}{
		{
			name:    "not found",
			get:     func(context.Context, string) (*backupv1alpha1.Backup, error) { return nil, notFoundErr },
			wantErr: ErrBackupNotFound,
		},
		{
			name:       "not succeeded",
			get:        func(context.Context, string) (*backupv1alpha1.Backup, error) { return running, nil },
			wantErr:    ErrBackupNotSucceeded,
			wantBackup: true,
		},
		{
			name:       "succeeded",
			get:        func(context.Context, string) (*backupv1alpha1.Backup, error) { return succeeded, nil },
			wantBackup: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			backup, err := ResolveSucceededBackup(ctx, "b", tc.get)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
			} else {
				require.NoError(t, err)
			}
			if tc.wantBackup {
				require.NotNil(t, backup)
			} else {
				require.Nil(t, backup)
			}
		})
	}
}

func TestResolveBackupClass(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	bc := &backupv1alpha1.BackupClass{ObjectMeta: metav1.ObjectMeta{Name: "bc"}}

	_, err := ResolveBackupClass(ctx, "bc", func(context.Context, string) (*backupv1alpha1.BackupClass, error) {
		return nil, notFoundError{}
	})
	require.ErrorIs(t, err, ErrBackupClassNotFound)

	got, err := ResolveBackupClass(ctx, "bc", func(context.Context, string) (*backupv1alpha1.BackupClass, error) {
		return bc, nil
	})
	require.NoError(t, err)
	require.Same(t, bc, got)
}

func TestValidateClassSupportsProvider(t *testing.T) {
	t.Parallel()

	bc := &backupv1alpha1.BackupClass{
		ObjectMeta: metav1.ObjectMeta{Name: "bc"},
		Spec:       backupv1alpha1.BackupClassSpec{SupportedProviders: backupv1alpha1.ProviderNameList{"postgresql"}},
	}

	require.NoError(t, ValidateClassSupportsProvider(bc, "postgresql"))

	err := ValidateClassSupportsProvider(bc, "mongodb")
	require.ErrorIs(t, err, ErrProviderUnsupported)
}

// notFoundError is a minimal apierrors-recognizable "not found" error for
// exercising the k8serrors.IsNotFound branch without depending on a fake client.
type notFoundError struct{}

func (notFoundError) Error() string { return "not found" }
func (notFoundError) Status() metav1.Status {
	return metav1.Status{Reason: metav1.StatusReasonNotFound}
}
