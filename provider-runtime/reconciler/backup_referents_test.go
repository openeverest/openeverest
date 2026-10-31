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
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
)

type backupTestProvider struct {
	readyTestProvider
}

func (p *backupTestProvider) SyncBackup(*controller.Context, *backupv1alpha1.Backup) (controller.BackupExecutionStatus, error) {
	return controller.BackupExecutionStatus{}, nil
}

func (p *backupTestProvider) SyncRestore(*controller.Context, *backupv1alpha1.Restore) (controller.RestoreExecutionStatus, error) {
	return controller.RestoreExecutionStatus{}, nil
}

func (p *backupTestProvider) CleanupBackup(*controller.Context, *backupv1alpha1.Backup) (bool, error) {
	return true, nil
}

func (p *backupTestProvider) CleanupRestore(*controller.Context, *backupv1alpha1.Restore) (bool, error) {
	return true, nil
}

func testBackupClass(mode backupv1alpha1.BackupExecutionMode, providers ...string) *backupv1alpha1.BackupClass {
	return &backupv1alpha1.BackupClass{
		ObjectMeta: metav1.ObjectMeta{Name: "pbm"},
		Spec: backupv1alpha1.BackupClassSpec{
			ExecutionMode:      mode,
			SupportedProviders: providers,
		},
	}
}

func testBackupStorage(name, secret string) *backupv1alpha1.BackupStorage {
	return &backupv1alpha1.BackupStorage{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "team-a"},
		Spec: backupv1alpha1.BackupStorageSpec{
			BackupStorageClassRef: common.ObjectRef{Name: "s3"},
			Location:              backupv1alpha1.BackupStorageLocation{Bucket: "backups"},
			Authentication: common.Authentication{
				Mode:                 "AccessKey",
				CredentialsSecretRef: &common.SecretRef{Name: secret},
			},
		},
	}
}

func testSecret(name string) *corev1.Secret {
	return &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "team-a"}}
}

func withBackup(enabled bool, storages ...string) func(*v1alpha1.Instance) {
	return func(in *v1alpha1.Instance) {
		in.Spec.Backup = &v1alpha1.InstanceBackupSpec{
			Enabled:  enabled,
			ClassRef: common.ObjectRef{Name: "pbm"},
		}
		for _, s := range storages {
			in.Spec.Backup.Storages = append(
				in.Spec.Backup.Storages,
				v1alpha1.InstanceBackupStorage{StorageRef: common.ObjectRef{Name: s}},
			)
		}
	}
}

func TestReconcile_BackupReferents(t *testing.T) {
	t.Parallel()

	// Objects are built per subtest: the fake client writes to the ones it is given.
	compatible := func() client.Object {
		return testBackupClass(backupv1alpha1.BackupExecutionModeProviderManaged, "test-provider")
	}
	storage := func() client.Object { return testBackupStorage("s3-prod", "s3-prod-keys") }
	keys := func() client.Object { return testSecret("s3-prod-keys") }

	tests := []struct {
		name        string
		backup      func(*v1alpha1.Instance)
		objs        func() []client.Object
		wantReady   metav1.ConditionStatus
		wantReason  string
		wantMessage string
	}{
		{
			name:        "missing class",
			backup:      withBackup(true, "s3-prod"),
			objs:        func() []client.Object { return []client.Object{storage(), keys()} },
			wantReady:   metav1.ConditionFalse,
			wantReason:  v1alpha1.ReasonBackupClassNotFound,
			wantMessage: "backup class pbm not found",
		},
		{
			name:   "Job-mode class",
			backup: withBackup(true, "s3-prod"),
			objs: func() []client.Object {
				return []client.Object{testBackupClass(backupv1alpha1.BackupExecutionModeJob), storage(), keys()}
			},
			wantReady:   metav1.ConditionFalse,
			wantReason:  v1alpha1.ReasonBackupClassIncompatible,
			wantMessage: "backup class pbm runs backups as Jobs and can't be an instance's backup class",
		},
		{
			name:   "class for another provider",
			backup: withBackup(true, "s3-prod"),
			objs: func() []client.Object {
				return []client.Object{testBackupClass(backupv1alpha1.BackupExecutionModeProviderManaged, "other"), storage(), keys()}
			},
			wantReady:   metav1.ConditionFalse,
			wantReason:  v1alpha1.ReasonBackupClassIncompatible,
			wantMessage: "backup class pbm does not support provider test-provider",
		},
		{
			name:        "missing storage",
			backup:      withBackup(true, "s3-prod"),
			objs:        func() []client.Object { return []client.Object{compatible()} },
			wantReady:   metav1.ConditionFalse,
			wantReason:  v1alpha1.ReasonBackupStorageNotFound,
			wantMessage: "backup storage s3-prod not found in team-a",
		},
		{
			name:        "missing storage credentials",
			backup:      withBackup(true, "s3-prod"),
			objs:        func() []client.Object { return []client.Object{compatible(), storage()} },
			wantReady:   metav1.ConditionFalse,
			wantReason:  v1alpha1.ReasonBackupStorageCredentialsNotFound,
			wantMessage: "secret s3-prod-keys with the credentials of backup storage s3-prod not found in team-a",
		},
		{
			name:       "every referent present",
			backup:     withBackup(true, "s3-prod"),
			objs:       func() []client.Object { return []client.Object{compatible(), storage(), keys()} },
			wantReady:  metav1.ConditionTrue,
			wantReason: v1alpha1.ReasonReady,
		},
		{
			name:       "backups disabled are not checked",
			backup:     withBackup(false, "s3-prod"),
			wantReady:  metav1.ConditionTrue,
			wantReason: v1alpha1.ReasonReady,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := &backupTestProvider{readyTestProvider{status: controller.Ready()}}
			objs := []client.Object{
				&v1alpha1.Provider{ObjectMeta: metav1.ObjectMeta{Name: "test-provider"}},
				readyTestInstance(tc.backup),
			}
			if tc.objs != nil {
				objs = append(objs, tc.objs()...)
			}
			r := newReadyTestReconciler(p, objs...)

			got, err := reconcileAndGet(t, r)
			require.NoError(t, err)
			c := requireReady(t, got, tc.wantReady, tc.wantReason)
			if tc.wantReady == metav1.ConditionTrue {
				assert.Equal(t, 1, p.calls)
				return
			}
			assert.Equal(t, tc.wantMessage, c.Message)
			assert.Zero(t, p.calls, "the provider must not be called")
			bc := meta.FindStatusCondition(got.Status.Conditions, v1alpha1.ConditionBackupConfigured)
			require.NotNil(t, bc)
			assert.Equal(t, tc.wantReason, bc.Reason)
		})
	}
}

func TestBackupReferentMapping(t *testing.T) {
	t.Parallel()
	named := func(name, ns string, mutate func(*v1alpha1.Instance)) *v1alpha1.Instance {
		in := readyTestInstance(mutate)
		in.Name = name
		in.Namespace = ns
		return in
	}
	r := newReadyTestReconciler(&backupTestProvider{},
		named("uses-prod", "team-a", withBackup(true, "s3-prod")),
		named("uses-dev", "team-a", withBackup(true, "s3-dev")),
		named("no-backup", "team-a", nil),
		named("elsewhere", "team-b", withBackup(true, "s3-prod")),
		testBackupStorage("s3-prod", "prod-keys"),
		testBackupStorage("s3-dev", "dev-keys"),
	)

	names := func(reqs []reconcile.Request) []string {
		out := make([]string, 0, len(reqs))
		for _, req := range reqs {
			out = append(out, req.Namespace+"/"+req.Name)
		}
		return out
	}

	assert.ElementsMatch(t, []string{"team-a/uses-prod"},
		names(r.instancesForBackupStorage(t.Context(), testBackupStorage("s3-prod", ""))),
	)
	assert.ElementsMatch(t, []string{"team-a/uses-prod", "team-a/uses-dev", "team-b/elsewhere"},
		names(r.instancesForBackupClass(t.Context(), testBackupClass(backupv1alpha1.BackupExecutionModeProviderManaged))),
	)
	assert.ElementsMatch(t, []string{"team-a/uses-dev"},
		names(r.instancesForSecret(t.Context(), testSecret("dev-keys"))),
	)
}
