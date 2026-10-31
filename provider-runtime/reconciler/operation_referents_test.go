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

package reconciler

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
)

type countingBackupProvider struct {
	backupTestProvider

	backups  int
	restores int
}

func (p *countingBackupProvider) SyncBackup(*controller.Context, *backupv1alpha1.Backup) (controller.BackupExecutionStatus, error) {
	p.backups++
	return controller.BackupExecutionStatus{}, nil
}

func (p *countingBackupProvider) SyncRestore(*controller.Context, *backupv1alpha1.Restore) (controller.RestoreExecutionStatus, error) {
	p.restores++
	return controller.RestoreExecutionStatus{}, nil
}

func newOperationTestClient(objs ...client.Object) client.Client { //nolint:ireturn
	scheme := newTestScheme()
	_ = corev1.AddToScheme(scheme)
	return fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(objs...).
		WithStatusSubresource(&backupv1alpha1.Backup{}, &backupv1alpha1.Restore{}, &backupv1alpha1.BackupImport{}).
		WithIndex(&backupv1alpha1.Restore{}, controller.IndexRestoreInstanceName, func(obj client.Object) []string {
			rs, ok := obj.(*backupv1alpha1.Restore)
			if !ok {
				return nil
			}
			return []string{rs.Spec.InstanceRef.Name}
		}).
		Build()
}

func testOperationInstance() *v1alpha1.Instance {
	return &v1alpha1.Instance{
		ObjectMeta: metav1.ObjectMeta{Name: "orders", Namespace: "team-a"},
		Spec:       v1alpha1.InstanceSpec{ProviderRef: common.ObjectRef{Name: "test-provider"}},
	}
}

func testBackup(state backupv1alpha1.BackupState) *backupv1alpha1.Backup {
	return &backupv1alpha1.Backup{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "nightly",
			Namespace:  "team-a",
			Finalizers: []string{backupRuntimeFinalizer},
		},
		Spec: backupv1alpha1.BackupSpec{
			Origin: backupv1alpha1.BackupOrigin{
				Type:        backupv1alpha1.BackupOriginTypeInstance,
				InstanceRef: &common.ObjectRef{Name: "orders"},
			},
			ClassRef:   common.ObjectRef{Name: "pbm"},
			StorageRef: common.ObjectRef{Name: "s3-prod"},
		},
		Status: backupv1alpha1.BackupStatus{State: state},
	}
}

func TestBackupReconcile_WaitsForReferents(t *testing.T) {
	t.Parallel()

	class := func() client.Object {
		return testBackupClass(backupv1alpha1.BackupExecutionModeProviderManaged, "test-provider")
	}
	tests := []struct {
		name        string
		state       backupv1alpha1.BackupState
		objs        func() []client.Object
		wantState   backupv1alpha1.BackupState
		wantMessage string
		wantSynced  bool
	}{
		{
			name:        "missing instance before the backup starts",
			objs:        func() []client.Object { return []client.Object{class()} },
			wantState:   backupv1alpha1.BackupStatePending,
			wantMessage: "instance orders not found in team-a",
		},
		{
			name:        "instance deleted while the backup runs",
			state:       backupv1alpha1.BackupStateRunning,
			objs:        func() []client.Object { return []client.Object{class()} },
			wantState:   backupv1alpha1.BackupStateFailed,
			wantMessage: "instance orders was deleted while the backup was running",
		},
		{
			name:      "a finished backup never fails for losing its instance",
			state:     backupv1alpha1.BackupStateSucceeded,
			objs:      func() []client.Object { return []client.Object{class()} },
			wantState: backupv1alpha1.BackupStateSucceeded,
		},
		{
			name:        "missing storage",
			objs:        func() []client.Object { return []client.Object{class(), testOperationInstance()} },
			wantState:   backupv1alpha1.BackupStatePending,
			wantMessage: "backup storage s3-prod not found in team-a",
		},
		{
			name: "storage not accepted",
			objs: func() []client.Object {
				return []client.Object{class(), testOperationInstance(), testBackupStorage("s3-prod", "s3-keys", metav1.ConditionFalse)}
			},
			wantState:   backupv1alpha1.BackupStatePending,
			wantMessage: "backup storage s3-prod is not accepted: CredentialsNotFound: ",
		},
		{
			name: "every referent present",
			objs: func() []client.Object {
				return []client.Object{
					class(), testOperationInstance(), testBackupStorage("s3-prod", "s3-keys", metav1.ConditionTrue), testSecret("s3-keys"),
				}
			},
			wantSynced: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := newOperationTestClient(append(tc.objs(), testBackup(tc.state))...)
			p := &countingBackupProvider{}
			r := &backupRuntimeReconciler{client: c, reader: c, provider: p, providerName: "test-provider"}

			key := client.ObjectKey{Namespace: "team-a", Name: "nightly"}
			_, err := r.Reconcile(t.Context(), reconcile.Request{NamespacedName: key})
			require.NoError(t, err)

			got := &backupv1alpha1.Backup{}
			require.NoError(t, c.Get(t.Context(), key, got))
			if tc.wantSynced {
				assert.Equal(t, 1, p.backups)
				return
			}
			assert.Zero(t, p.backups, "the provider must not be called")
			assert.Equal(t, tc.wantState, got.Status.State)
			assert.Equal(t, tc.wantMessage, got.Status.Message)
		})
	}
}

func TestRestoreReconcile_WaitsForInstance(t *testing.T) {
	t.Parallel()

	source := testBackup(backupv1alpha1.BackupStateSucceeded)
	source.Name = "nightly-1"
	restore := &backupv1alpha1.Restore{
		ObjectMeta: metav1.ObjectMeta{Name: "rollback", Namespace: "team-a"},
		Spec: backupv1alpha1.RestoreSpec{
			InstanceRef: common.ObjectRef{Name: "orders"},
			DataSource: backupv1alpha1.DataSource{
				Type:   backupv1alpha1.DataSourceTypeBackup,
				Backup: &backupv1alpha1.DataSourceBackup{BackupRef: common.ObjectRef{Name: "nightly-1"}},
			},
		},
	}
	c := newOperationTestClient(
		testBackupClass(backupv1alpha1.BackupExecutionModeProviderManaged, "test-provider"), source, restore,
	)
	p := &countingBackupProvider{}
	r := &restoreRuntimeReconciler{client: c, reader: c, provider: p, providerName: "test-provider"}

	key := client.ObjectKeyFromObject(restore)
	_, err := r.Reconcile(t.Context(), reconcile.Request{NamespacedName: key})
	require.NoError(t, err)

	got := &backupv1alpha1.Restore{}
	require.NoError(t, c.Get(t.Context(), key, got))
	assert.Equal(t, backupv1alpha1.RestoreStatePending, got.Status.State)
	assert.Equal(t, "instance orders not found in team-a", got.Status.Message)
	assert.Zero(t, p.restores, "the provider must not be called")

	reqs := r.restoresForInstance(t.Context(), testOperationInstance())
	require.Len(t, reqs, 1)
	assert.Equal(t, "rollback", reqs[0].Name)
}

func TestBackupsForReferents(t *testing.T) {
	t.Parallel()

	finished := testBackup(backupv1alpha1.BackupStateSucceeded)
	finished.Name = "finished"
	otherStorage := testBackup("")
	otherStorage.Name = "other-storage"
	otherStorage.Spec.StorageRef.Name = "s3-dev"
	c := newOperationTestClient(
		testBackup(""), finished, otherStorage,
		testBackupStorage("s3-prod", "prod-keys", metav1.ConditionTrue), testBackupStorage("s3-dev", "dev-keys", metav1.ConditionTrue),
	)
	r := &backupRuntimeReconciler{client: c, reader: c, providerName: "test-provider"}

	names := func(reqs []reconcile.Request) []string {
		out := make([]string, 0, len(reqs))
		for _, req := range reqs {
			out = append(out, req.Name)
		}
		return out
	}
	assert.ElementsMatch(t, []string{"nightly", "other-storage"},
		names(r.backupsForInstance(t.Context(), testOperationInstance())),
	)
	assert.ElementsMatch(t, []string{"nightly"},
		names(r.backupsForStorage(t.Context(), testBackupStorage("s3-prod", "", ""))),
	)
	assert.ElementsMatch(t, []string{"other-storage"},
		names(r.backupsForSecret(t.Context(), testSecret("dev-keys"))),
	)
}
