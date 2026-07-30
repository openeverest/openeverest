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

package validation

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	"github.com/openeverest/openeverest/v2/internal/server/handlers"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
)

func TestCreateRestore_Validation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	namespace := "test-namespace"

	pitrClass := &backupv1alpha1.BackupClass{
		ObjectMeta: metav1.ObjectMeta{Name: "pitr-class"},
		Spec: backupv1alpha1.BackupClassSpec{
			ExecutionMode: backupv1alpha1.BackupExecutionModeProviderManaged,
			ProviderManaged: &backupv1alpha1.ProviderManagedSpec{
				SupportsPITR: true,
			},
		},
	}
	noPitrClass := &backupv1alpha1.BackupClass{
		ObjectMeta: metav1.ObjectMeta{Name: "no-pitr-class"},
		Spec: backupv1alpha1.BackupClassSpec{
			ExecutionMode: backupv1alpha1.BackupExecutionModeProviderManaged,
			ProviderManaged: &backupv1alpha1.ProviderManagedSpec{
				SupportsPITR: false,
			},
		},
	}
	succeededBackup := &backupv1alpha1.Backup{
		ObjectMeta: metav1.ObjectMeta{Name: "succeeded-backup", Namespace: namespace},
		Spec:       backupv1alpha1.BackupSpec{ClassRef: common.ObjectRef{Name: "pitr-class"}},
		Status:     backupv1alpha1.BackupStatus{State: backupv1alpha1.BackupStateSucceeded},
	}
	succeededBackupNoPitr := &backupv1alpha1.Backup{
		ObjectMeta: metav1.ObjectMeta{Name: "succeeded-backup-no-pitr", Namespace: namespace},
		Spec:       backupv1alpha1.BackupSpec{ClassRef: common.ObjectRef{Name: "no-pitr-class"}},
		Status:     backupv1alpha1.BackupStatus{State: backupv1alpha1.BackupStateSucceeded},
	}
	runningBackup := &backupv1alpha1.Backup{
		ObjectMeta: metav1.ObjectMeta{Name: "running-backup", Namespace: namespace},
		Spec:       backupv1alpha1.BackupSpec{ClassRef: common.ObjectRef{Name: "pitr-class"}},
		Status:     backupv1alpha1.BackupStatus{State: backupv1alpha1.BackupStateRunning},
	}

	scheme := runtime.NewScheme()
	require.NoError(t, backupv1alpha1.AddToScheme(scheme))

	newRestore := func(backupName string, pitr *backupv1alpha1.DataSourcePITR) *backupv1alpha1.Restore {
		return &backupv1alpha1.Restore{
			ObjectMeta: metav1.ObjectMeta{Name: "test-restore", Namespace: namespace},
			Spec: backupv1alpha1.RestoreSpec{
				DataSource: backupv1alpha1.DataSource{
					Type: backupv1alpha1.DataSourceTypeBackup,
					Backup: &backupv1alpha1.DataSourceBackup{
						BackupRef: common.ObjectRef{Name: backupName},
						PITR:      pitr,
					},
				},
			},
		}
	}

	tests := []struct {
		name    string
		restore *backupv1alpha1.Restore
		objects []ctrlclient.Object
		err     string
	}{
		{
			name:    "missing backup fails",
			restore: newRestore("missing-backup", nil),
			objects: nil,
			err:     "backup 'missing-backup' does not exist",
		},
		{
			name:    "backup not succeeded fails",
			restore: newRestore("running-backup", nil),
			objects: []ctrlclient.Object{runningBackup},
			err:     "backup 'running-backup' is in state 'Running', must be 'Succeeded' to restore from it",
		},
		{
			name:    "succeeded backup, no PITR, passes",
			restore: newRestore("succeeded-backup-no-pitr", nil),
			objects: []ctrlclient.Object{succeededBackupNoPitr},
			err:     "",
		},
		{
			name:    "succeeded backup, PITR requested, class does not support PITR fails",
			restore: newRestore("succeeded-backup-no-pitr", &backupv1alpha1.DataSourcePITR{Type: backupv1alpha1.PITRTypeLatest}),
			objects: []ctrlclient.Object{succeededBackupNoPitr, noPitrClass},
			err:     "point-in-time recovery is not supported",
		},
		{
			name:    "succeeded backup, PITR requested, class supports PITR passes",
			restore: newRestore("succeeded-backup", &backupv1alpha1.DataSourcePITR{Type: backupv1alpha1.PITRTypeLatest}),
			objects: []ctrlclient.Object{succeededBackup, pitrClass},
			err:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Use DeepCopy to avoid race conditions since the fake client
			// modifies the objects' ResourceVersion during Build().
			objs := make([]ctrlclient.Object, len(tt.objects))
			for i, obj := range tt.objects {
				objs[i] = obj.DeepCopyObject().(ctrlclient.Object) //nolint:forcetypeassert
			}
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()
			kubeConnector := kubernetes.NewEmpty(zap.NewNop().Sugar(), namespace).WithKubernetesClient(fakeClient)

			mockNext := &handlers.MockHandler{}
			mockNext.On("CreateRestore", mock.Anything, mock.Anything).Return(tt.restore, nil)

			handler := &validateHandler{
				log:           zap.NewNop().Sugar(),
				kubeConnector: kubeConnector,
				next:          mockNext,
			}

			_, err := handler.CreateRestore(ctx, tt.restore)
			if tt.err == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tt.err)
		})
	}
}
