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

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
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
			if !ok || b.Spec.InstanceName == "" {
				return nil
			}
			return []string{b.Spec.InstanceName}
		}).
		WithIndex(&backupv1alpha1.Restore{}, controller.IndexRestoreInstanceName, func(obj client.Object) []string {
			rs, ok := obj.(*backupv1alpha1.Restore)
			if !ok || rs.Spec.InstanceName == "" {
				return nil
			}
			return []string{rs.Spec.InstanceName}
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
			Provider:       "test-provider",
			DeletionPolicy: corev1alpha1.InstanceDeletionPolicyCascade,
		},
	}
	backup1 := &backupv1alpha1.Backup{
		ObjectMeta: metav1.ObjectMeta{Name: "backup-1", Namespace: "default"},
		Spec: backupv1alpha1.BackupSpec{
			InstanceName:    "my-db",
			BackupClassName: "bc",
			StorageName:     "s3",
		},
	}
	backup2 := &backupv1alpha1.Backup{
		ObjectMeta: metav1.ObjectMeta{Name: "backup-2", Namespace: "default"},
		Spec: backupv1alpha1.BackupSpec{
			InstanceName:    "my-db",
			BackupClassName: "bc",
			StorageName:     "s3",
		},
	}
	backupOther := &backupv1alpha1.Backup{
		ObjectMeta: metav1.ObjectMeta{Name: "backup-other", Namespace: "default"},
		Spec: backupv1alpha1.BackupSpec{
			InstanceName:    "other-db",
			BackupClassName: "bc",
			StorageName:     "s3",
		},
	}
	restore1 := &backupv1alpha1.Restore{
		ObjectMeta: metav1.ObjectMeta{Name: "restore-1", Namespace: "default"},
		Spec: backupv1alpha1.RestoreSpec{
			InstanceName: "my-db",
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
			Provider:       "test-provider",
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
