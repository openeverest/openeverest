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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

func newReferentTestClient(t *testing.T, objs ...client.Object) client.Client { //nolint:ireturn
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, backupv1alpha1.AddToScheme(scheme))
	require.NoError(t, corev1alpha1.AddToScheme(scheme))
	return fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(objs...).
		WithStatusSubresource(&backupv1alpha1.Backup{}, &backupv1alpha1.Restore{}, &backupv1alpha1.BackupImport{}).
		Build()
}

func TestBackupReconcile_WaitsForClass(t *testing.T) {
	t.Parallel()
	backup := &backupv1alpha1.Backup{
		ObjectMeta: metav1.ObjectMeta{Name: "nightly", Namespace: "team-a"},
		Spec: backupv1alpha1.BackupSpec{
			Origin: backupv1alpha1.BackupOrigin{
				Type:        backupv1alpha1.BackupOriginTypeInstance,
				InstanceRef: &common.ObjectRef{Name: "orders"},
			},
			ClassRef:   common.ObjectRef{Name: "pbm"},
			StorageRef: common.ObjectRef{Name: "s3-prod"},
		},
	}
	c := newReferentTestClient(t, backup)
	r := &BackupReconciler{Client: c}

	_, err := r.Reconcile(t.Context(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(backup)})
	require.NoError(t, err)

	got := &backupv1alpha1.Backup{}
	require.NoError(t, c.Get(t.Context(), client.ObjectKeyFromObject(backup), got))
	assert.Equal(t, backupv1alpha1.BackupStatePending, got.Status.State)
	assert.Equal(t, "backup class pbm not found", got.Status.Message)
}

func TestRestoreReconcile_WaitsForSource(t *testing.T) {
	t.Parallel()
	restore := func(state backupv1alpha1.RestoreState) *backupv1alpha1.Restore {
		return &backupv1alpha1.Restore{
			ObjectMeta: metav1.ObjectMeta{Name: "rollback", Namespace: "team-a"},
			Spec: backupv1alpha1.RestoreSpec{
				InstanceRef: common.ObjectRef{Name: "orders"},
				DataSource: backupv1alpha1.DataSource{
					Type:   backupv1alpha1.DataSourceTypeBackup,
					Backup: &backupv1alpha1.DataSourceBackup{BackupRef: common.ObjectRef{Name: "nightly-1"}},
				},
			},
			Status: backupv1alpha1.RestoreStatus{State: state},
		}
	}

	tests := []struct {
		name        string
		state       backupv1alpha1.RestoreState
		wantState   backupv1alpha1.RestoreState
		wantMessage string
	}{
		{
			name:        "a restore that has not started waits for its source",
			wantState:   backupv1alpha1.RestoreStatePending,
			wantMessage: "backup nightly-1 not found in team-a",
		},
		{
			name:      "a running restore is left to its owner",
			state:     backupv1alpha1.RestoreStateRunning,
			wantState: backupv1alpha1.RestoreStateRunning,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rs := restore(tc.state)
			c := newReferentTestClient(t, rs)
			r := &RestoreReconciler{Client: c}

			_, err := r.Reconcile(t.Context(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(rs)})
			require.NoError(t, err)

			got := &backupv1alpha1.Restore{}
			require.NoError(t, c.Get(t.Context(), client.ObjectKeyFromObject(rs), got))
			assert.Equal(t, tc.wantState, got.Status.State)
			assert.Equal(t, tc.wantMessage, got.Status.Message)
		})
	}
}

func TestBackupImportReconcile_WaitsForClass(t *testing.T) {
	t.Parallel()
	imp := &backupv1alpha1.BackupImport{
		ObjectMeta: metav1.ObjectMeta{Name: "import", Namespace: "team-a"},
		Spec: backupv1alpha1.BackupImportSpec{
			ClassRef:   common.ObjectRef{Name: "pbm"},
			StorageRef: common.ObjectRef{Name: "s3-prod"},
		},
	}
	c := newReferentTestClient(t, imp)
	r := &BackupImportReconciler{Client: c}

	_, err := r.Reconcile(t.Context(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(imp)})
	require.NoError(t, err)

	got := &backupv1alpha1.BackupImport{}
	require.NoError(t, c.Get(t.Context(), client.ObjectKeyFromObject(imp), got))
	assert.Equal(t, backupv1alpha1.BackupImportStatePending, got.Status.State)
	assert.Equal(t, "backup class pbm not found", got.Status.Message)
}

func TestRestoreReferentSelectors(t *testing.T) {
	t.Parallel()
	pitr := &backupv1alpha1.Restore{
		Spec: backupv1alpha1.RestoreSpec{
			InstanceRef: common.ObjectRef{Name: "orders-copy"},
			DataSource: backupv1alpha1.DataSource{
				Type: backupv1alpha1.DataSourceTypePointInTime,
				PointInTime: &backupv1alpha1.DataSourcePointInTime{
					Source: backupv1alpha1.StreamSource{InstanceRef: &common.ObjectRef{Name: "orders"}},
				},
			},
		},
	}
	instance := func(name string) client.Object {
		return &corev1alpha1.Instance{ObjectMeta: metav1.ObjectMeta{Name: name}}
	}
	assert.True(t, restoreNamesInstance(pitr, instance("orders")), "the stream's Instance")
	assert.True(t, restoreNamesInstance(pitr, instance("orders-copy")), "the target Instance")
	assert.False(t, restoreNamesInstance(pitr, instance("billing")))
	assert.False(t, restoreNamesBackup(pitr, &backupv1alpha1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "nightly-1"}}))
}
