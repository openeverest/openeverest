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
	"fmt"
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
	"github.com/openeverest/openeverest/v2/backupstorageclasses"
	"github.com/openeverest/openeverest/v2/backupstorageclasses/s3"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
)

type backupTestProvider struct {
	readyTestProvider
	sync func(*controller.Context)
}

func (p *backupTestProvider) Sync(c *controller.Context) error {
	if p.sync != nil {
		p.sync(c)
	}
	return nil
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

// otherMode is an execution mode the provider runtime does not execute.
const otherMode = backupv1alpha1.BackupExecutionMode("Other")

// testBackupClass returns a class of mode at generation 1, executed by
// provider when one is given. Core has not judged it yet.
func testBackupClass(mode backupv1alpha1.BackupExecutionMode, provider ...string) *backupv1alpha1.BackupClass {
	bc := &backupv1alpha1.BackupClass{
		ObjectMeta: metav1.ObjectMeta{Name: "pbm", UID: "pbm-uid", Generation: 1},
		Spec: backupv1alpha1.BackupClassSpec{
			ExecutionMode: mode,
			BackupStorageClasses: []backupv1alpha1.BackupStorageClassSupport{
				{Name: s3.ClassName, AuthModes: []string{s3.AuthModeAccessKey}},
			},
		},
	}
	if len(provider) > 0 {
		bc.Spec.ProviderManaged = &backupv1alpha1.ProviderManagedSpec{ProviderRef: common.ObjectRef{Name: provider[0]}}
	}
	return bc
}

// judged returns bc with core's verdict for provider published at bc's
// generation.
func judged(bc *backupv1alpha1.BackupClass, provider string, c backupv1alpha1.Compatibility, reason string) *backupv1alpha1.BackupClass {
	bc.Status.Providers = append(bc.Status.Providers, backupv1alpha1.BackupClassProviderCompatibility{
		Name: provider, Compatibility: c, Reason: reason, ClassGeneration: bc.Generation,
	})
	return bc
}

// compatibleBackupClass returns the class test-provider executes, which
// core published as Compatible with it.
func compatibleBackupClass() *backupv1alpha1.BackupClass {
	return judged(
		testBackupClass(backupv1alpha1.BackupExecutionModeProviderManaged, "test-provider"),
		"test-provider", backupv1alpha1.CompatibilityCompatible, "",
	)
}

// testBackupStorageClass returns the s3 BackupStorageClass at generation 1.
func testBackupStorageClass() *backupv1alpha1.BackupStorageClass {
	return &backupv1alpha1.BackupStorageClass{ObjectMeta: metav1.ObjectMeta{Name: s3.ClassName, UID: "s3-uid", Generation: 1}}
}

// withStorageVerdict returns bs with core's verdict for the class of
// testBackupClass, computed from the current storage, class and storage
// class.
func withStorageVerdict(bs *backupv1alpha1.BackupStorage, c backupv1alpha1.Compatibility, reason, message string) *backupv1alpha1.BackupStorage {
	bs.Status.BackupClasses = []backupv1alpha1.BackupClassVerdict{{
		Name: "pbm", Compatibility: c, Reason: reason, Message: message,
		Observed: backupv1alpha1.BackupClassVerdictInputs{
			StorageGeneration: bs.Generation,
			ClassUID:          "pbm-uid", ClassGeneration: 1,
			StorageClassUID: "s3-uid", StorageClassGeneration: 1,
		},
	}}
	return bs
}

// testBackupStorage returns an s3 storage with an Accepted condition of the
// given status, or none when accepted is empty. An accepted storage carries
// core's Compatible verdict for the class of testBackupClass.
func testBackupStorage(name, secret string, accepted metav1.ConditionStatus) *backupv1alpha1.BackupStorage {
	bs := &backupv1alpha1.BackupStorage{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "team-a"},
		Spec: backupv1alpha1.BackupStorageSpec{
			BackupStorageClassRef: common.ObjectRef{Name: s3.ClassName},
			Location:              backupv1alpha1.BackupStorageLocation{ObjectStore: &backupv1alpha1.ObjectStoreLocation{Bucket: "backups", Prefix: "prod"}},
			Authentication: common.Authentication{
				Mode:                 s3.AuthModeAccessKey,
				CredentialsSecretRef: &common.SecretRef{Name: secret},
			},
		},
	}
	if accepted != "" {
		reason := backupv1alpha1.ReasonAccepted
		if accepted != metav1.ConditionTrue {
			reason = backupv1alpha1.ReasonCredentialsNotFound
		}
		bs.Status.Conditions = []metav1.Condition{{
			Type:   backupv1alpha1.BackupStorageConditionAccepted,
			Status: accepted,
			Reason: reason,
		}}
	}
	if accepted == metav1.ConditionTrue {
		return withStorageVerdict(bs, backupv1alpha1.CompatibilityCompatible, "", "")
	}
	return bs
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

func withRecordedLocation(name, location string) func(*v1alpha1.Instance) {
	return func(in *v1alpha1.Instance) {
		in.Status.Backup = &v1alpha1.InstanceBackupStatus{
			Storages: []v1alpha1.InstanceBackupStorageStatus{{Name: name, Location: location}},
		}
	}
}

func instanceWith(mutate ...func(*v1alpha1.Instance)) func(*v1alpha1.Instance) {
	return func(in *v1alpha1.Instance) {
		for _, m := range mutate {
			m(in)
		}
	}
}

// TestReconcile_BackupClassReferent covers the class verdicts, which keep the
// provider from being called at all.
func TestReconcile_BackupClassReferent(t *testing.T) {
	t.Parallel()

	storage := func() client.Object { return testBackupStorage("s3-prod", "s3-prod-keys", metav1.ConditionTrue) }

	tests := []struct {
		name        string
		objs        func() []client.Object
		wantReason  string
		wantMessage string
	}{
		{
			name:        "missing class",
			objs:        func() []client.Object { return []client.Object{storage()} },
			wantReason:  v1alpha1.ReasonBackupClassNotFound,
			wantMessage: "backup class pbm not found",
		},
		{
			name: "class not judged yet",
			objs: func() []client.Object {
				return []client.Object{testBackupClass(backupv1alpha1.BackupExecutionModeProviderManaged, "test-provider"), storage()}
			},
			wantReason: v1alpha1.ReasonBackupClassPending,
			wantMessage: `backup class pbm is pending for provider test-provider: VerdictPending: ` +
				`BackupClass "pbm" has not been judged for Provider "test-provider" at generation 1 yet`,
		},
		{
			name: "class not claimed yet",
			objs: func() []client.Object {
				return []client.Object{judged(
					testBackupClass(backupv1alpha1.BackupExecutionModeProviderManaged, "test-provider"),
					"test-provider", backupv1alpha1.CompatibilityPending, backupv1alpha1.ReasonClassNotAccepted,
				), storage()}
			},
			wantReason:  v1alpha1.ReasonBackupClassPending,
			wantMessage: "backup class pbm is pending for provider test-provider: ClassNotAccepted",
		},
		{
			name: "verdict for an older generation",
			objs: func() []client.Object {
				bc := compatibleBackupClass()
				bc.Generation = 2
				return []client.Object{bc, storage()}
			},
			wantReason: v1alpha1.ReasonBackupClassPending,
			wantMessage: `backup class pbm is pending for provider test-provider: VerdictPending: ` +
				`the verdict on BackupClass "pbm" for Provider "test-provider" predates generation 2 of the class`,
		},
		{
			name: "drifted class",
			objs: func() []client.Object {
				return []client.Object{judged(
					testBackupClass(backupv1alpha1.BackupExecutionModeProviderManaged, "test-provider"),
					"test-provider", backupv1alpha1.CompatibilityIncompatible, backupv1alpha1.ReasonSpecDrift,
				), storage()}
			},
			wantReason:  v1alpha1.ReasonBackupClassIncompatible,
			wantMessage: "backup class pbm is incompatible with provider test-provider: SpecDrift",
		},
		{
			name: "class for another provider",
			objs: func() []client.Object {
				return []client.Object{judged(
					testBackupClass(backupv1alpha1.BackupExecutionModeProviderManaged, "other"),
					"other", backupv1alpha1.CompatibilityCompatible, "",
				), storage()}
			},
			wantReason: v1alpha1.ReasonBackupClassIncompatible,
			wantMessage: `backup class pbm is incompatible with provider test-provider: ProviderUnsupported: ` +
				`BackupClass "pbm" does not run on Provider "test-provider"`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := &backupTestProvider{readyTestProvider: readyTestProvider{status: controller.Ready()}}
			objs := []client.Object{
				&v1alpha1.Provider{ObjectMeta: metav1.ObjectMeta{Name: "test-provider"}},
				readyTestInstance(withBackup(true, "s3-prod")),
			}
			objs = append(objs, tc.objs()...)
			r := newReadyTestReconciler(p, objs...)

			got, err := reconcileAndGet(t, r)
			require.NoError(t, err)
			c := requireReady(t, got, metav1.ConditionFalse, tc.wantReason)
			assert.Equal(t, tc.wantMessage, c.Message)
			assert.Zero(t, p.calls, "the provider must not be called")
			bc := meta.FindStatusCondition(got.Status.Conditions, v1alpha1.ConditionBackupConfigured)
			require.NotNil(t, bc)
			assert.Equal(t, tc.wantReason, bc.Reason)
		})
	}
}

// TestReconcile_BackupStorageEntries covers the per-entry verdicts: the
// provider is always called, and each entry reports its own reason.
func TestReconcile_BackupStorageEntries(t *testing.T) {
	t.Parallel()

	compatible := func() client.Object { return compatibleBackupClass() }
	storage := func(name string, accepted metav1.ConditionStatus) func() client.Object {
		return func() client.Object { return testBackupStorage(name, name+"-keys", accepted) }
	}
	judgedStorage := func(c backupv1alpha1.Compatibility, reason, message string) func() client.Object {
		return func() client.Object {
			return withStorageVerdict(testBackupStorage("s3-prod", "s3-prod-keys", metav1.ConditionTrue), c, reason, message)
		}
	}
	staleStorage := func() client.Object {
		bs := testBackupStorage("s3-prod", "s3-prod-keys", metav1.ConditionTrue)
		bs.Status.BackupClasses[0].Observed.ClassGeneration = 0
		return bs
	}
	const prodPath = "prod/backups/team-a/orders/orders-uid/s3-prod"

	type want struct {
		reason   string
		message  string
		location string
	}
	tests := []struct {
		name     string
		instance func(*v1alpha1.Instance)
		objs     []func() client.Object
		reject   map[string]error
		want     map[string]want
		ready    string
	}{
		{
			name:     "every entry usable",
			instance: withBackup(true, "s3-prod"),
			objs:     []func() client.Object{compatible, storage("s3-prod", metav1.ConditionTrue)},
			want:     map[string]want{"s3-prod": {location: prodPath}},
			ready:    v1alpha1.ReasonReady,
		},
		{
			name:     "missing storage",
			instance: withBackup(true, "s3-prod"),
			objs:     []func() client.Object{compatible},
			want: map[string]want{"s3-prod": {
				reason:  v1alpha1.ReasonBackupStorageNotFound,
				message: "backup storage s3-prod not found in team-a",
			}},
			ready: v1alpha1.ReasonBackupStorageNotFound,
		},
		{
			name:     "storage not validated yet",
			instance: withBackup(true, "s3-prod"),
			objs:     []func() client.Object{compatible, storage("s3-prod", "")},
			want: map[string]want{"s3-prod": {
				reason:  v1alpha1.ReasonBackupStorageNotAccepted,
				message: "backup storage s3-prod has not been validated yet",
			}},
			ready: v1alpha1.ReasonBackupStorageNotAccepted,
		},
		{
			name:     "storage not accepted",
			instance: withBackup(true, "s3-prod"),
			objs:     []func() client.Object{compatible, storage("s3-prod", metav1.ConditionFalse)},
			want: map[string]want{"s3-prod": {
				reason:  v1alpha1.ReasonBackupStorageNotAccepted,
				message: "backup storage s3-prod is not accepted: CredentialsNotFound: ",
			}},
			ready: v1alpha1.ReasonBackupStorageNotAccepted,
		},
		{
			name:     "backup class cannot write to the storage",
			instance: withBackup(true, "s3-prod"),
			objs: []func() client.Object{compatible, judgedStorage(
				backupv1alpha1.CompatibilityIncompatible, backupv1alpha1.ReasonAuthModeUnsupported, "only AccessKey",
			)},
			want: map[string]want{"s3-prod": {
				reason:  v1alpha1.ReasonBackupStorageIncompatible,
				message: "backup class pbm cannot write to backup storage s3-prod: AuthModeUnsupported: only AccessKey",
			}},
			ready: v1alpha1.ReasonBackupStorageIncompatible,
		},
		{
			name:     "storage verdict pending",
			instance: withBackup(true, "s3-prod"),
			objs: []func() client.Object{compatible, judgedStorage(
				backupv1alpha1.CompatibilityPending, backupv1alpha1.ReasonStorageNotAccepted, "not yet",
			)},
			want: map[string]want{"s3-prod": {
				reason:  v1alpha1.ReasonBackupStorageVerdictPending,
				message: "backup storage s3-prod is Pending for backup class pbm: StorageNotAccepted: not yet",
			}},
			ready: v1alpha1.ReasonBackupStorageVerdictPending,
		},
		{
			name:     "storage verdict for an older class",
			instance: withBackup(true, "s3-prod"),
			objs:     []func() client.Object{compatible, staleStorage},
			want:     map[string]want{"s3-prod": {reason: v1alpha1.ReasonBackupStorageVerdictPending}},
			ready:    v1alpha1.ReasonBackupStorageVerdictPending,
		},
		{
			name:     "a held entry keeps its recorded location",
			instance: instanceWith(withBackup(true, "s3-prod"), withRecordedLocation("s3-prod", "old/backups/team-a/orders/orders-uid/s3-prod")),
			objs:     []func() client.Object{compatible, storage("s3-prod", metav1.ConditionTrue)},
			want: map[string]want{"s3-prod": {
				reason:   v1alpha1.ReasonBackupStorageLocationChanged,
				location: "old/backups/team-a/orders/orders-uid/s3-prod",
			}},
			ready: v1alpha1.ReasonBackupStorageLocationChanged,
		},
		{
			name:     "the provider rejects an entry",
			instance: withBackup(true, "s3-prod"),
			objs:     []func() client.Object{compatible, storage("s3-prod", metav1.ConditionTrue)},
			reject:   map[string]error{"s3-prod": fmt.Errorf("decode: %w", backupstorageclasses.ErrUnsupportedParameter)},
			want: map[string]want{"s3-prod": {
				reason:  v1alpha1.ReasonUnsupportedParameter,
				message: "backup storage s3-prod: decode: unsupported parameter",
			}},
			ready: v1alpha1.ReasonUnsupportedParameter,
		},
		{
			name:     "one bad entry among good ones",
			instance: withBackup(true, "s3-prod", "s3-dr"),
			objs:     []func() client.Object{compatible, storage("s3-prod", metav1.ConditionTrue)},
			want: map[string]want{
				"s3-prod": {location: prodPath},
				"s3-dr":   {reason: v1alpha1.ReasonBackupStorageNotFound, message: "backup storage s3-dr not found in team-a"},
			},
			ready: v1alpha1.ReasonBackupStorageNotFound,
		},
		{
			name:     "several bad entries",
			instance: withBackup(true, "s3-prod", "s3-dr"),
			objs:     []func() client.Object{compatible, storage("s3-prod", "")},
			want: map[string]want{
				"s3-prod": {reason: v1alpha1.ReasonBackupStorageNotAccepted, message: "backup storage s3-prod has not been validated yet"},
				"s3-dr":   {reason: v1alpha1.ReasonBackupStorageNotFound, message: "backup storage s3-dr not found in team-a"},
			},
			ready: v1alpha1.ReasonBackupStoragesUnusable,
		},
		{
			name:     "backups disabled are not checked",
			instance: withBackup(false, "s3-prod"),
			ready:    v1alpha1.ReasonReady,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := &backupTestProvider{
				readyTestProvider: readyTestProvider{status: controller.Ready()},
				sync: func(c *controller.Context) {
					for name, err := range tc.reject {
						c.RejectBackupStorage(name, err)
					}
				},
			}
			in := readyTestInstance(tc.instance)
			in.UID = "orders-uid"
			objs := []client.Object{&v1alpha1.Provider{ObjectMeta: metav1.ObjectMeta{Name: "test-provider"}}, in, testBackupStorageClass()}
			for _, o := range tc.objs {
				objs = append(objs, o())
			}
			r := newReadyTestReconciler(p, objs...)

			got, err := reconcileAndGet(t, r)
			require.NoError(t, err)
			readyStatus := metav1.ConditionFalse
			if tc.ready == v1alpha1.ReasonReady {
				readyStatus = metav1.ConditionTrue
			}
			requireReady(t, got, readyStatus, tc.ready)
			assert.Equal(t, 1, p.calls, "the provider is called whatever its storages")

			if tc.want == nil {
				assert.Nil(t, got.Status.Backup)
				return
			}
			require.NotNil(t, got.Status.Backup)
			gotEntries := make(map[string]want, len(got.Status.Backup.Storages))
			for _, s := range got.Status.Backup.Storages {
				gotEntries[s.Name] = want{reason: s.Reason, message: s.Message, location: s.Location}
			}
			for name, w := range tc.want {
				g, ok := gotEntries[name]
				require.True(t, ok, "status entry %s missing", name)
				assert.Equal(t, w.reason, g.reason, name)
				assert.Equal(t, w.location, g.location, name)
				if w.message != "" {
					assert.Equal(t, w.message, g.message, name)
				}
			}
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
		testBackupStorage("s3-prod", "prod-keys", metav1.ConditionTrue),
		testBackupStorage("s3-dev", "dev-keys", metav1.ConditionTrue),
	)

	names := func(reqs []reconcile.Request) []string {
		out := make([]string, 0, len(reqs))
		for _, req := range reqs {
			out = append(out, req.Namespace+"/"+req.Name)
		}
		return out
	}

	assert.ElementsMatch(t, []string{"team-a/uses-prod"},
		names(r.instancesForBackupStorage(t.Context(), testBackupStorage("s3-prod", "", ""))),
	)
	assert.ElementsMatch(t, []string{"team-a/uses-prod", "team-a/uses-dev", "team-b/elsewhere"},
		names(r.instancesForBackupClass(t.Context(), testBackupClass(backupv1alpha1.BackupExecutionModeProviderManaged))),
	)
	assert.ElementsMatch(t, []string{"team-a/uses-dev"},
		names(r.instancesForBackupStorageSecret(t.Context(), testSecret("dev-keys"))),
	)
}
