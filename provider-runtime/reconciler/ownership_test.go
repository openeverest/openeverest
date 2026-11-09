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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

// TestOwnership_OnlyClaimedClasses checks that the runtime executes the
// backups, restores and imports of the classes that name its provider and
// ignores, without an error, those of any other executor (spec 019 §4.8
// invariant 2), even for its own Instances.
func TestOwnership_OnlyClaimedClasses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		class *backupv1alpha1.BackupClass
		want  bool
	}{
		{name: "a class of this provider", class: testBackupClass(backupv1alpha1.BackupExecutionModeProviderManaged, "test-provider"), want: true},
		{name: "a class of another provider", class: testBackupClass(backupv1alpha1.BackupExecutionModeProviderManaged, "other")},
		{name: "a class of another execution mode", class: testBackupClass(otherMode)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.class.Spec.SupportsImport = true
			in := readyTestInstance(nil)
			backup := &backupv1alpha1.Backup{
				ObjectMeta: metav1.ObjectMeta{Name: "nightly", Namespace: in.Namespace},
				Spec: backupv1alpha1.BackupSpec{
					ClassRef:   common.ObjectRef{Name: tc.class.Name},
					StorageRef: common.ObjectRef{Name: "s3-prod"},
					Origin: backupv1alpha1.BackupOrigin{
						Type:        backupv1alpha1.BackupOriginTypeInstance,
						InstanceRef: &common.ObjectRef{Name: in.Name},
					},
				},
			}
			restore := &backupv1alpha1.Restore{
				ObjectMeta: metav1.ObjectMeta{Name: "back", Namespace: in.Namespace},
				Spec: backupv1alpha1.RestoreSpec{
					InstanceRef: common.ObjectRef{Name: in.Name},
					DataSource: backupv1alpha1.DataSource{
						Type:   backupv1alpha1.DataSourceTypeBackup,
						Backup: &backupv1alpha1.DataSourceBackup{BackupRef: common.ObjectRef{Name: backup.Name}},
					},
				},
			}
			imp := backupImport(in.Namespace, "import", tc.class.Name, "s3-prod")
			c := newFakeClient(newTestScheme(), tc.class, in, backup, restore, imp)

			_, _, ours, err := resolveBackupOwnership(t.Context(), c, backup, "test-provider")
			require.NoError(t, err)
			assert.Equal(t, tc.want, ours, "backup")

			_, _, ours, err = resolveRestoreOwnership(t.Context(), c, restore, "test-provider")
			require.NoError(t, err)
			assert.Equal(t, tc.want, ours, "restore")

			ours, err = resolveImportOwnership(t.Context(), c, imp, "test-provider")
			require.NoError(t, err)
			assert.Equal(t, tc.want, ours, "import")
		})
	}
}

// TestOwnership_InstanceOfAnotherProvider checks that a class of this
// provider does not make the backups of another provider's Instance its own.
func TestOwnership_InstanceOfAnotherProvider(t *testing.T) {
	t.Parallel()
	in := readyTestInstance(func(in *v1alpha1.Instance) { in.Spec.ProviderRef.Name = "other" })
	class := testBackupClass(backupv1alpha1.BackupExecutionModeProviderManaged, "test-provider")
	backup := &backupv1alpha1.Backup{
		ObjectMeta: metav1.ObjectMeta{Name: "nightly", Namespace: in.Namespace},
		Spec: backupv1alpha1.BackupSpec{
			ClassRef: common.ObjectRef{Name: class.Name},
			Origin:   backupv1alpha1.BackupOrigin{Type: backupv1alpha1.BackupOriginTypeInstance, InstanceRef: &common.ObjectRef{Name: in.Name}},
		},
	}
	c := newFakeClient(newTestScheme(), class, in, backup)

	_, _, ours, err := resolveBackupOwnership(t.Context(), c, backup, "test-provider")
	require.NoError(t, err)
	assert.False(t, ours)
}
