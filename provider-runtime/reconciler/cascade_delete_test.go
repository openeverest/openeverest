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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
)

func newTestScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	_ = corev1alpha1.AddToScheme(s)
	_ = backupv1alpha1.AddToScheme(s)
	return s
}

func newFakeClient(scheme *runtime.Scheme, objs ...client.Object) client.Client {
	return fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(objs...).
		WithIndex(&backupv1alpha1.Backup{}, controller.IndexBackupInstanceName, func(obj client.Object) []string {
			b, ok := obj.(*backupv1alpha1.Backup)
			if !ok || b.Spec.Origin.InstanceRef == nil {
				return nil
			}
			return []string{b.Spec.Origin.InstanceRef.Name}
		}).
		WithIndex(&backupv1alpha1.Restore{}, controller.IndexRestoreInstanceName, func(obj client.Object) []string {
			rs, ok := obj.(*backupv1alpha1.Restore)
			if !ok || rs.Spec.InstanceRef.Name == "" {
				return nil
			}
			return []string{rs.Spec.InstanceRef.Name}
		}).
		Build()
}

// testLogger implements the logger interface used by handleDeletion / cascadeDeleteChildren.
type testLogger struct {
	t *testing.T
}

func (l *testLogger) Info(msg string, keysAndValues ...interface{}) {
	l.t.Logf("INFO: %s %v", msg, keysAndValues)
}

func TestCascadeDeleteChildren_DeletesBackupsAndRestores(t *testing.T) {
	scheme := newTestScheme()
	instance := &corev1alpha1.Instance{
		ObjectMeta: metav1.ObjectMeta{Name: "my-db", Namespace: "default"},
		Spec: corev1alpha1.InstanceSpec{
			ProviderRef:    common.ObjectRef{Name: "test-provider"},
			DeletionPolicy: corev1alpha1.InstanceDeletionPolicyCascade,
		},
	}
	backup1 := &backupv1alpha1.Backup{
		ObjectMeta: metav1.ObjectMeta{Name: "backup-1", Namespace: "default"},
		Spec: backupv1alpha1.BackupSpec{
			Origin: backupv1alpha1.BackupOrigin{
				Type:        backupv1alpha1.BackupOriginTypeInstance,
				InstanceRef: &common.ObjectRef{Name: "my-db"},
			},
			ClassRef:   common.ObjectRef{Name: "bc"},
			StorageRef: common.ObjectRef{Name: "s3"},
		},
	}
	backup2 := &backupv1alpha1.Backup{
		ObjectMeta: metav1.ObjectMeta{Name: "backup-2", Namespace: "default"},
		Spec: backupv1alpha1.BackupSpec{
			Origin: backupv1alpha1.BackupOrigin{
				Type:        backupv1alpha1.BackupOriginTypeInstance,
				InstanceRef: &common.ObjectRef{Name: "my-db"},
			},
			ClassRef:   common.ObjectRef{Name: "bc"},
			StorageRef: common.ObjectRef{Name: "s3"},
		},
	}
	backupOther := &backupv1alpha1.Backup{
		ObjectMeta: metav1.ObjectMeta{Name: "backup-other", Namespace: "default"},
		Spec: backupv1alpha1.BackupSpec{
			Origin: backupv1alpha1.BackupOrigin{
				Type:        backupv1alpha1.BackupOriginTypeInstance,
				InstanceRef: &common.ObjectRef{Name: "other-db"},
			},
			ClassRef:   common.ObjectRef{Name: "bc"},
			StorageRef: common.ObjectRef{Name: "s3"},
		},
	}
	restore1 := &backupv1alpha1.Restore{
		ObjectMeta: metav1.ObjectMeta{Name: "restore-1", Namespace: "default"},
		Spec: backupv1alpha1.RestoreSpec{
			InstanceRef: common.ObjectRef{Name: "my-db"},
		},
	}

	c := newFakeClient(scheme, instance, backup1, backup2, backupOther, restore1)
	r := &ProviderReconciler{Client: c}
	inCtx := controller.NewContext(context.Background(), c, instance, "test-provider")
	logger := &testLogger{t: t}

	remaining, err := r.cascadeDeleteChildren(context.Background(), inCtx, logger)
	require.NoError(t, err)
	// We expect 3 items (2 backups + 1 restore for "my-db") to be in the remaining count
	// because the list is done before deletion and the objects still exist at count time.
	assert.Equal(t, 3, remaining)

	// After deletion, re-listing should show them gone (fake client honors Delete immediately).
	backups := &backupv1alpha1.BackupList{}
	err = c.List(context.Background(), backups, client.InNamespace("default"),
		client.MatchingFields{controller.IndexBackupInstanceName: "my-db"})
	require.NoError(t, err)
	assert.Empty(t, backups.Items)

	restores := &backupv1alpha1.RestoreList{}
	err = c.List(context.Background(), restores, client.InNamespace("default"),
		client.MatchingFields{controller.IndexRestoreInstanceName: "my-db"})
	require.NoError(t, err)
	assert.Empty(t, restores.Items)

	// Verify "other-db" backup was NOT deleted.
	otherBackups := &backupv1alpha1.BackupList{}
	err = c.List(context.Background(), otherBackups, client.InNamespace("default"),
		client.MatchingFields{controller.IndexBackupInstanceName: "other-db"})
	require.NoError(t, err)
	assert.Len(t, otherBackups.Items, 1)
}

func TestCascadeDeleteChildren_NoChildren(t *testing.T) {
	scheme := newTestScheme()
	instance := &corev1alpha1.Instance{
		ObjectMeta: metav1.ObjectMeta{Name: "empty-db", Namespace: "default"},
		Spec: corev1alpha1.InstanceSpec{
			ProviderRef:    common.ObjectRef{Name: "test-provider"},
			DeletionPolicy: corev1alpha1.InstanceDeletionPolicyCascade,
		},
	}

	c := newFakeClient(scheme, instance)
	r := &ProviderReconciler{Client: c}
	inCtx := controller.NewContext(context.Background(), c, instance, "test-provider")
	logger := &testLogger{t: t}

	remaining, err := r.cascadeDeleteChildren(context.Background(), inCtx, logger)
	require.NoError(t, err)
	assert.Equal(t, 0, remaining)
}

// A Backup or Restore kept (Orphan) from an earlier Instance must survive the
// cascade delete of a later Instance that reuses the name.
func TestCascadeDeleteChildren_SkipsChildrenOfEarlierInstanceWithSameName(t *testing.T) {
	t.Parallel()

	scheme := newTestScheme()
	instance := &corev1alpha1.Instance{
		ObjectMeta: metav1.ObjectMeta{Name: "my-db", Namespace: "default", UID: "uid-later"},
		Spec: corev1alpha1.InstanceSpec{
			ProviderRef:    common.ObjectRef{Name: "test-provider"},
			DeletionPolicy: corev1alpha1.InstanceDeletionPolicyCascade,
		},
	}
	backup := func(name, claimedUID string) *backupv1alpha1.Backup {
		b := &backupv1alpha1.Backup{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"},
			Spec: backupv1alpha1.BackupSpec{
				Origin: backupv1alpha1.BackupOrigin{
					Type:        backupv1alpha1.BackupOriginTypeInstance,
					InstanceRef: &common.ObjectRef{Name: "my-db"},
				},
				ClassRef:   common.ObjectRef{Name: "bc"},
				StorageRef: common.ObjectRef{Name: "s3"},
			},
		}
		if claimedUID != "" {
			b.Labels = map[string]string{controller.InstanceUIDLabel: claimedUID}
		}
		return b
	}
	restore := func(name, claimedUID string) *backupv1alpha1.Restore {
		return &backupv1alpha1.Restore{
			ObjectMeta: metav1.ObjectMeta{
				Name: name, Namespace: "default",
				Labels: map[string]string{controller.InstanceUIDLabel: claimedUID},
			},
			Spec: backupv1alpha1.RestoreSpec{InstanceRef: common.ObjectRef{Name: "my-db"}},
		}
	}

	c := newFakeClient(scheme, instance,
		backup("own", "uid-later"),
		backup("unclaimed", ""),
		backup("kept", "uid-earlier"),
		restore("own-restore", "uid-later"),
		restore("kept-restore", "uid-earlier"),
	)
	r := &ProviderReconciler{Client: c}
	inCtx := controller.NewContext(context.Background(), c, instance, "test-provider")

	remaining, err := r.cascadeDeleteChildren(context.Background(), inCtx, &testLogger{t: t})
	require.NoError(t, err)
	assert.Equal(t, 3, remaining, "only the later Instance's children are waited for")

	exists := func(obj client.Object) bool {
		err := c.Get(context.Background(), client.ObjectKeyFromObject(obj), obj)
		return err == nil
	}
	assert.True(t, exists(backup("kept", "")), "Backup of the earlier Instance was deleted")
	assert.True(t, exists(restore("kept-restore", "")), "Restore of the earlier Instance was deleted")
	assert.False(t, exists(backup("own", "")))
	assert.False(t, exists(backup("unclaimed", "")))
	assert.False(t, exists(restore("own-restore", "")))
}

func TestBackupAndRestoreReconcile_ClaimRecordsInstanceUID(t *testing.T) {
	t.Parallel()

	instance := testOperationInstance()
	instance.UID = "uid-orders"
	backup := testBackup("")
	backup.Finalizers = nil
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
		testBackupClass(backupv1alpha1.BackupExecutionModeProviderManaged, "test-provider"),
		instance, backup, source, restore,
	)
	p := &countingBackupProvider{}

	br := &backupRuntimeReconciler{client: c, reader: c, provider: p, providerName: "test-provider"}
	_, err := br.Reconcile(t.Context(), reconcile.Request{NamespacedName: client.ObjectKeyFromObject(backup)})
	require.NoError(t, err)
	gotBackup := &backupv1alpha1.Backup{}
	require.NoError(t, c.Get(t.Context(), client.ObjectKeyFromObject(backup), gotBackup))
	assert.Equal(t, "uid-orders", gotBackup.Labels[controller.InstanceUIDLabel])

	rr := &restoreRuntimeReconciler{client: c, reader: c, provider: p, providerName: "test-provider"}
	_, err = rr.Reconcile(t.Context(), reconcile.Request{NamespacedName: client.ObjectKeyFromObject(restore)})
	require.NoError(t, err)
	gotRestore := &backupv1alpha1.Restore{}
	require.NoError(t, c.Get(t.Context(), client.ObjectKeyFromObject(restore), gotRestore))
	assert.Equal(t, "uid-orders", gotRestore.Labels[controller.InstanceUIDLabel])
}
