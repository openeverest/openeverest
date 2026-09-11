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
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
)

const testImportProvider = "test-provider"

// fakeImporter is a stub BackupImporter that returns a canned result so the
// reconciler's dedup/create/status logic can be exercised without touching a
// real BackupStorage.
type fakeImporter struct {
	result controller.BackupImportExecutionStatus
	err    error
}

func (f *fakeImporter) ImportBackups(
	_ context.Context,
	_ *backupv1alpha1.BackupImport,
	_ *backupv1alpha1.BackupStorage,
) (controller.BackupImportExecutionStatus, error) {
	return f.result, f.err
}

//nolint:ireturn // testing only
func newImportFakeClient(objs ...client.Object) client.Client {
	return fake.NewClientBuilder().
		WithScheme(newTestScheme()).
		WithObjects(objs...).
		WithStatusSubresource(&backupv1alpha1.BackupImport{}).
		Build()
}

func providerManagedBackupClass(name string) *backupv1alpha1.BackupClass {
	return &backupv1alpha1.BackupClass{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: backupv1alpha1.BackupClassSpec{
			ExecutionMode:      backupv1alpha1.BackupExecutionModeProviderManaged,
			SupportedProviders: backupv1alpha1.ProviderNameList{testImportProvider},
		},
	}
}

func backupImport(namespace, name, className, storageName string) *backupv1alpha1.BackupImport {
	return &backupv1alpha1.BackupImport{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name, Generation: 1},
		Spec: backupv1alpha1.BackupImportSpec{
			ClassRef:   common.ObjectRef{Name: className},
			StorageRef: common.ObjectRef{Name: storageName},
		},
	}
}

func externalBackup(name, storageName, path string) *backupv1alpha1.Backup {
	now := metav1.Now()
	return &backupv1alpha1.Backup{
		ObjectMeta: metav1.ObjectMeta{Namespace: testImportNamespace, Name: name},
		Spec: backupv1alpha1.BackupSpec{
			ClassRef:   common.ObjectRef{Name: "external-class"},
			StorageRef: common.ObjectRef{Name: storageName},
			Origin: backupv1alpha1.BackupOrigin{
				Type: backupv1alpha1.BackupOriginTypeExternal,
				External: &backupv1alpha1.BackupOriginExternal{
					Path:        path,
					StartedAt:   now,
					CompletedAt: now,
				},
			},
		},
	}
}

const testImportNamespace = "default"

func testStorage() *backupv1alpha1.BackupStorage {
	return &backupv1alpha1.BackupStorage{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testImportNamespace,
			Name:      "storage-1",
		},
	}
}

func TestBackupImportReconcile_Discovery(t *testing.T) {
	t.Parallel()

	importName := types.NamespacedName{Namespace: testImportNamespace, Name: "import-1"}

	tests := []struct {
		name             string
		existing         []*backupv1alpha1.Backup
		discovered       controller.BackupImportExecutionStatus
		wantCreated      int
		wantDiscovered   int
		wantTotalBackups int
	}{
		{
			name: "already exists",
			existing: []*backupv1alpha1.Backup{
				externalBackup("existing", "storage-1", "shared/path"),
			},
			discovered: controller.BackupImportExecutionStatus{
				State: backupv1alpha1.BackupImportStateSucceeded,
				Backups: []*backupv1alpha1.Backup{
					externalBackup("discovered-dup", "storage-1", "shared/path"),
				},
			},
			wantCreated:      0,
			wantDiscovered:   1,
			wantTotalBackups: 1,
		},
		{
			name: "backups sharing a path",
			discovered: controller.BackupImportExecutionStatus{
				State: backupv1alpha1.BackupImportStateSucceeded,
				Backups: []*backupv1alpha1.Backup{
					externalBackup("discovered-a", "storage-1", "same/path"),
					externalBackup("discovered-b", "storage-1", "same/path"),
				},
			},
			wantCreated:      1,
			wantDiscovered:   2,
			wantTotalBackups: 1,
		},
		{
			name: "backups different paths",
			discovered: controller.BackupImportExecutionStatus{
				State: backupv1alpha1.BackupImportStateSucceeded,
				Backups: []*backupv1alpha1.Backup{
					externalBackup("discovered-a", "storage-1", "path/a"),
					externalBackup("discovered-b", "storage-1", "path/b"),
				},
			},
			wantCreated:      2,
			wantDiscovered:   2,
			wantTotalBackups: 2,
		},
		{
			name: "backups different storage",
			discovered: controller.BackupImportExecutionStatus{
				State: backupv1alpha1.BackupImportStateSucceeded,
				Backups: []*backupv1alpha1.Backup{
					externalBackup("discovered-a", "storage-1", "path/a"),
					externalBackup("discovered-b", "storage-2", "path/a"),
				},
			},
			wantCreated:      2,
			wantDiscovered:   2,
			wantTotalBackups: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			storage := testStorage()
			imp := backupImport(testImportNamespace, importName.Name, "class-1", storage.Name)

			objs := []client.Object{imp, providerManagedBackupClass("class-1"), storage}
			for _, b := range tt.existing {
				objs = append(objs, b)
			}

			c := newImportFakeClient(objs...)
			importer := &fakeImporter{result: tt.discovered}

			r := &backupImportReconciler{
				client:       c,
				importer:     importer,
				providerName: testImportProvider,
			}
			_, err := r.Reconcile(t.Context(), reconcile.Request{NamespacedName: importName})
			require.NoError(t, err)

			bi := &backupv1alpha1.BackupImport{}
			err = c.Get(t.Context(), importName, bi)
			require.NoError(t, err)
			assert.Equal(t, backupv1alpha1.BackupImportStateSucceeded, bi.Status.State)
			assert.EqualValues(t, tt.wantDiscovered, bi.Status.DiscoveredCount)
			assert.EqualValues(t, tt.wantCreated, bi.Status.CreatedCount)
			backups := &backupv1alpha1.BackupList{}
			err = c.List(t.Context(), backups, client.InNamespace(testImportNamespace))
			require.NoError(t, err)
			assert.Len(t, backups.Items, tt.wantTotalBackups)
		})
	}
}

func TestBackupImportReconcile_Status(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		withStorage   bool
		initialStatus backupv1alpha1.BackupImportStatus
		wantErr       bool
		wantState     backupv1alpha1.BackupImportState
	}{
		{
			name:        "succeeded status short-circuits the reconcile",
			withStorage: true,
			initialStatus: backupv1alpha1.BackupImportStatus{
				State: backupv1alpha1.BackupImportStateSucceeded,
			},
			wantState: backupv1alpha1.BackupImportStateSucceeded,
		},
		{
			name:        "failed status short-circuits the reconcile",
			withStorage: true,
			initialStatus: backupv1alpha1.BackupImportStatus{
				State: backupv1alpha1.BackupImportStateFailed,
			},
			wantState: backupv1alpha1.BackupImportStateFailed,
		},
		{
			name:        "storage fetch failure lands in Error state",
			withStorage: false,
			wantErr:     true,
			wantState:   backupv1alpha1.BackupImportStateError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			objs := []client.Object{providerManagedBackupClass("class-1")}

			storageRef := "missing-storage"
			if tt.withStorage {
				storage := testStorage()
				storageRef = storage.Name
				objs = append(objs, storage)
			}

			importName := types.NamespacedName{
				Namespace: testImportNamespace,
				Name:      "import-1",
			}

			imp := backupImport(testImportNamespace, importName.Name, "class-1", storageRef)
			imp.Status = tt.initialStatus
			objs = append(objs, imp)

			c := newImportFakeClient(objs...)
			importer := &fakeImporter{}

			r := &backupImportReconciler{
				client:       c,
				importer:     importer,
				providerName: testImportProvider,
			}
			_, err := r.Reconcile(t.Context(), reconcile.Request{NamespacedName: importName})
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			got := &backupv1alpha1.BackupImport{}
			err = c.Get(t.Context(), importName, got)
			require.NoError(t, err)
			assert.Equal(t, tt.wantState, got.Status.State)
		})
	}
}
