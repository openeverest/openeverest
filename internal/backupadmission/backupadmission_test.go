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

package backupadmission_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/internal/backupadmission"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
)

const (
	ns       = "db"
	provider = "psmdb"
)

var s3 = &backupv1alpha1.BackupStorageClass{ //nolint:gochecknoglobals // shared test fixture
	ObjectMeta: metav1.ObjectMeta{Name: "s3", UID: "s3-uid", Generation: 1},
}

// class returns a class executed by provider whose verdict for it, computed
// at the class's generation, has compatibility and reason.
func class(name string, compatibility backupv1alpha1.Compatibility, reason string) *backupv1alpha1.BackupClass {
	return &backupv1alpha1.BackupClass{
		ObjectMeta: metav1.ObjectMeta{Name: name, UID: types.UID(name + "-uid"), Generation: 1},
		Spec: backupv1alpha1.BackupClassSpec{
			ExecutionMode: backupv1alpha1.BackupExecutionModeProviderManaged,
			ProviderManaged: &backupv1alpha1.ProviderManagedSpec{
				ProviderRef:  common.ObjectRef{Name: provider},
				SupportsPITR: true,
			},
			SupportsImport: true,
		},
		Status: backupv1alpha1.BackupClassStatus{
			Providers: []backupv1alpha1.BackupClassProviderCompatibility{{
				Name: provider, Compatibility: compatibility, Reason: reason, Message: "detail", ClassGeneration: 1,
			}},
		},
	}
}

// storage returns an s3 storage whose verdict for c, computed from the
// current objects, has compatibility and reason.
func storage(name string, c *backupv1alpha1.BackupClass, compatibility backupv1alpha1.Compatibility, reason string) *backupv1alpha1.BackupStorage {
	return &backupv1alpha1.BackupStorage{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns, Generation: 1},
		Spec:       backupv1alpha1.BackupStorageSpec{BackupStorageClassRef: common.ObjectRef{Name: s3.Name}},
		Status: backupv1alpha1.BackupStorageStatus{
			BackupClasses: []backupv1alpha1.BackupClassVerdict{{
				Name: c.Name, Compatibility: compatibility, Reason: reason, Message: "detail",
				Observed: backupv1alpha1.BackupClassVerdictInputs{
					StorageGeneration: 1, ClassUID: c.UID, ClassGeneration: c.Generation,
					StorageClassUID: s3.UID, StorageClassGeneration: s3.Generation,
				},
			}},
		},
	}
}

func instance(name, providerName string, backup *corev1alpha1.InstanceBackupSpec) *corev1alpha1.Instance {
	return &corev1alpha1.Instance{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Spec:       corev1alpha1.InstanceSpec{ProviderRef: common.ObjectRef{Name: providerName}, Backup: backup},
	}
}

func backup(name, className, storageName string, origin *common.ObjectRef) *backupv1alpha1.Backup {
	b := &backupv1alpha1.Backup{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Spec: backupv1alpha1.BackupSpec{
			Origin:     backupv1alpha1.BackupOrigin{Type: backupv1alpha1.BackupOriginTypeExternal},
			ClassRef:   common.ObjectRef{Name: className},
			StorageRef: common.ObjectRef{Name: storageName},
		},
	}
	if origin != nil {
		b.Spec.Origin = backupv1alpha1.BackupOrigin{Type: backupv1alpha1.BackupOriginTypeInstance, InstanceRef: origin}
	}
	return b
}

func reader(t *testing.T, objs ...client.Object) backupadmission.ClientReader {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, backupv1alpha1.AddToScheme(scheme))
	require.NoError(t, corev1alpha1.AddToScheme(scheme))
	copies := make([]client.Object, 0, len(objs)+1)
	copies = append(copies, s3.DeepCopy())
	for _, o := range objs {
		copies = append(copies, o.DeepCopyObject().(client.Object)) //nolint:forcetypeassert
	}
	return backupadmission.ClientReader{Reader: fake.NewClientBuilder().WithScheme(scheme).WithObjects(copies...).Build()}
}

// outcome is what a test expects of a decision.
type outcome struct {
	denied   error    // sentinel the denial wraps; nil when admitted
	reason   string   // text the denial contains
	missing  []error  // sentinels of the missing objects
	warnings []string // texts of the warnings, in order
}

func assertDecision(t *testing.T, want outcome, d backupadmission.Decision, err error) {
	t.Helper()
	require.NoError(t, err)
	if want.denied == nil {
		require.NoError(t, d.Denied)
	} else {
		require.ErrorIs(t, d.Denied, want.denied)
		require.ErrorIs(t, d.Denied, controller.ErrInvalidReference)
		assert.Contains(t, d.Denied.Error(), want.reason)
	}
	require.Len(t, d.Missing, len(want.missing))
	for i, m := range want.missing {
		require.ErrorIs(t, d.Missing[i], m)
		require.ErrorIs(t, d.Missing[i], controller.ErrInvalidReference)
	}
	got := d.Warnings()
	require.Len(t, got, len(want.warnings), "warnings: %q", got)
	for i, w := range want.warnings {
		assert.Contains(t, got[i], w)
	}
}

func TestBackup(t *testing.T) {
	t.Parallel()

	db := instance("mongo", provider, nil)
	pbm := class("pbm", backupv1alpha1.CompatibilityCompatible, "")
	aws := storage("aws", pbm, backupv1alpha1.CompatibilityCompatible, "")
	onDB := &common.ObjectRef{Name: db.Name}

	stalePBM := pbm.DeepCopy()
	stalePBM.Generation = 2
	staleAWS := aws.DeepCopy()
	staleAWS.Generation = 2
	otherLevel := class("pbm", "BackupOnly", "RestoreUnsupported")

	tests := []struct {
		name    string
		backup  *backupv1alpha1.Backup
		objects []client.Object
		want    outcome
	}{
		{
			name:    "compatible",
			backup:  backup("b", "pbm", "aws", onDB),
			objects: []client.Object{db, pbm, aws},
		},
		{
			name:    "class pending",
			backup:  backup("b", "pending", "aws", onDB),
			objects: []client.Object{db, class("pending", backupv1alpha1.CompatibilityPending, backupv1alpha1.ReasonClassNotAccepted), aws},
			want: outcome{warnings: []string{
				`BackupClass "pending" is Pending for Provider "psmdb" (ClassNotAccepted): detail`,
				`BackupStorage "aws" is Pending for BackupClass "pending" (VerdictPending)`,
			}},
		},
		{
			name:    "class incompatible",
			backup:  backup("b", "drift", "aws", onDB),
			objects: []client.Object{db, class("drift", backupv1alpha1.CompatibilityIncompatible, backupv1alpha1.ReasonSpecDrift), aws},
			want: outcome{
				denied: controller.ErrBackupClassIncompatible,
				reason: `BackupClass "drift" is Incompatible with Provider "psmdb" (SpecDrift): detail`,
				// The storage has no verdict for a class nobody claims.
				warnings: []string{`BackupStorage "aws" is Pending for BackupClass "drift"`},
			},
		},
		{
			name:    "class does not run on the instance's provider",
			backup:  backup("b", "pbm", "aws", &common.ObjectRef{Name: "pg"}),
			objects: []client.Object{instance("pg", "percona-postgresql", nil), pbm, aws},
			want: outcome{
				denied: controller.ErrBackupClassIncompatible,
				reason: `Provider "percona-postgresql" (ProviderUnsupported)`,
			},
		},
		{
			name:    "class verdict for an older generation",
			backup:  backup("b", "pbm", "aws", onDB),
			objects: []client.Object{db, stalePBM, aws},
			want: outcome{warnings: []string{
				`BackupClass "pbm" is Pending for Provider "psmdb" (VerdictPending)`,
				`BackupStorage "aws" is Pending for BackupClass "pbm" (VerdictPending)`,
			}},
		},
		{
			name:    "a compatibility level this build does not know",
			backup:  backup("b", "pbm", "aws", onDB),
			objects: []client.Object{db, otherLevel, aws},
			want:    outcome{warnings: []string{`BackupClass "pbm" is BackupOnly for Provider "psmdb" (RestoreUnsupported)`}},
		},
		{
			name:    "storage incompatible",
			backup:  backup("b", "pbm", "gcs", onDB),
			objects: []client.Object{db, pbm, storage("gcs", pbm, backupv1alpha1.CompatibilityIncompatible, backupv1alpha1.ReasonParameterUnsupported)},
			want: outcome{
				denied: controller.ErrBackupStorageIncompatible,
				reason: `BackupStorage "gcs" is Incompatible with BackupClass "pbm" (ParameterUnsupported): detail`,
			},
		},
		{
			name:    "storage pending",
			backup:  backup("b", "pbm", "new", onDB),
			objects: []client.Object{db, pbm, storage("new", pbm, backupv1alpha1.CompatibilityPending, backupv1alpha1.ReasonStorageNotAccepted)},
			want:    outcome{warnings: []string{`BackupStorage "new" is Pending for BackupClass "pbm" (StorageNotAccepted)`}},
		},
		{
			name:    "storage verdict for an older generation",
			backup:  backup("b", "pbm", "aws", onDB),
			objects: []client.Object{db, pbm, staleAWS},
			want:    outcome{warnings: []string{`BackupStorage "aws" is Pending for BackupClass "pbm" (VerdictPending)`}},
		},
		{
			name:   "nothing exists yet",
			backup: backup("b", "pbm", "aws", onDB),
			want: outcome{
				missing: []error{controller.ErrInstanceNotFound, controller.ErrBackupStorageNotFound, controller.ErrBackupClassNotFound},
				warnings: []string{
					"instance 'mongo' does not exist; the Backup waits until it exists",
					"backup storage 'aws' does not exist; the Backup waits until it exists",
					"backup class not found: 'pbm'; the Backup waits until it exists",
				},
			},
		},
		{
			name:    "external backup is judged for the class's executor",
			backup:  backup("b", "drift", "aws", nil),
			objects: []client.Object{class("drift", backupv1alpha1.CompatibilityIncompatible, backupv1alpha1.ReasonSpecDrift), aws},
			want: outcome{
				denied:   controller.ErrBackupClassIncompatible,
				reason:   `Provider "psmdb" (SpecDrift)`,
				warnings: []string{`BackupStorage "aws" is Pending for BackupClass "drift"`},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d, err := backupadmission.Backup(context.Background(), reader(t, tt.objects...), tt.backup)
			assertDecision(t, tt.want, d, err)
		})
	}
}

func TestRestore(t *testing.T) {
	t.Parallel()

	pbm := class("pbm", backupv1alpha1.CompatibilityCompatible, "")
	physical := class("physical", backupv1alpha1.CompatibilityCompatible, "")
	noPITR := class("no-pitr", backupv1alpha1.CompatibilityCompatible, "")
	noPITR.Spec.ProviderManaged.SupportsPITR = false
	archiving := func(className, storageName string) *corev1alpha1.InstanceBackupSpec {
		return &corev1alpha1.InstanceBackupSpec{
			Enabled:  true,
			ClassRef: common.ObjectRef{Name: className},
			Storages: []corev1alpha1.InstanceBackupStorage{{
				StorageRef: common.ObjectRef{Name: storageName},
				PITR:       &corev1alpha1.InstanceBackupStoragePITR{Enabled: true},
			}},
		}
	}
	// The target writes with physical; the source Backup was written with pbm.
	target := instance("mongo", provider, archiving("physical", "aws"))
	succeeded := func(b *backupv1alpha1.Backup, state backupv1alpha1.BackupState) *backupv1alpha1.Backup {
		b.Status.State = state
		return b
	}
	fromBackup := func(name string) *backupv1alpha1.Restore {
		return &backupv1alpha1.Restore{
			ObjectMeta: metav1.ObjectMeta{Name: "r", Namespace: ns},
			Spec: backupv1alpha1.RestoreSpec{
				InstanceRef: common.ObjectRef{Name: target.Name},
				DataSource: backupv1alpha1.DataSource{
					Type:   backupv1alpha1.DataSourceTypeBackup,
					Backup: &backupv1alpha1.DataSourceBackup{BackupRef: common.ObjectRef{Name: name}},
				},
			},
		}
	}
	pointInTime := func(source, storageName string) *backupv1alpha1.Restore {
		return &backupv1alpha1.Restore{
			ObjectMeta: metav1.ObjectMeta{Name: "r", Namespace: ns},
			Spec: backupv1alpha1.RestoreSpec{
				InstanceRef: common.ObjectRef{Name: target.Name},
				DataSource: backupv1alpha1.DataSource{
					Type: backupv1alpha1.DataSourceTypePointInTime,
					PointInTime: &backupv1alpha1.DataSourcePointInTime{
						Source:         backupv1alpha1.StreamSource{InstanceRef: &common.ObjectRef{Name: source}, StorageRef: common.ObjectRef{Name: storageName}},
						RecoveryTarget: backupv1alpha1.RecoveryTargetLatest,
					},
				},
			},
		}
	}

	tests := []struct {
		name    string
		restore *backupv1alpha1.Restore
		objects []client.Object
		want    outcome
	}{
		{
			name:    "the class the source backup was written with",
			restore: fromBackup("old"),
			objects: []client.Object{target, physical, pbm, succeeded(backup("old", "pbm", "aws", nil), backupv1alpha1.BackupStateSucceeded)},
		},
		{
			name:    "source backup's class incompatible",
			restore: fromBackup("old"),
			objects: []client.Object{
				target, physical, class("pbm", backupv1alpha1.CompatibilityIncompatible, backupv1alpha1.ReasonUnknownClass),
				succeeded(backup("old", "pbm", "aws", nil), backupv1alpha1.BackupStateSucceeded),
			},
			want: outcome{denied: controller.ErrBackupClassIncompatible, reason: `BackupClass "pbm" is Incompatible with Provider "psmdb" (UnknownClass)`},
		},
		{
			name:    "source backup's class does not run on the target's provider",
			restore: fromBackup("old"),
			objects: []client.Object{
				instance("mongo", "percona-postgresql", nil), pbm,
				succeeded(backup("old", "pbm", "aws", nil), backupv1alpha1.BackupStateSucceeded),
			},
			want: outcome{denied: controller.ErrBackupClassIncompatible, reason: "(ProviderUnsupported)"},
		},
		{
			name:    "source backup's class pending",
			restore: fromBackup("old"),
			objects: []client.Object{
				target, class("pbm", backupv1alpha1.CompatibilityPending, backupv1alpha1.ReasonProviderNotFound),
				succeeded(backup("old", "pbm", "aws", nil), backupv1alpha1.BackupStateSucceeded),
			},
			want: outcome{warnings: []string{`BackupClass "pbm" is Pending for Provider "psmdb" (ProviderNotFound)`}},
		},
		{
			name:    "source backup not succeeded",
			restore: fromBackup("running"),
			objects: []client.Object{target, pbm, succeeded(backup("running", "pbm", "aws", nil), backupv1alpha1.BackupStateRunning)},
			want:    outcome{denied: controller.ErrBackupNotSucceeded, reason: "'running' is in state 'Running'"},
		},
		{
			name:    "source backup and target missing",
			restore: fromBackup("old"),
			want: outcome{
				missing: []error{controller.ErrBackupNotFound, controller.ErrInstanceNotFound},
				warnings: []string{
					"backup not found: 'old' in namespace 'db'; the Restore waits until it exists",
					"instance 'mongo' does not exist; the Restore waits until it exists",
				},
			},
		},
		{
			name:    "point in time from a compatible source storage",
			restore: pointInTime("source", "aws"),
			objects: []client.Object{target, instance("source", provider, archiving("pbm", "aws")), pbm, storage("aws", pbm, backupv1alpha1.CompatibilityCompatible, "")},
		},
		{
			name:    "point in time from an incompatible source storage",
			restore: pointInTime("source", "aws"),
			objects: []client.Object{
				target, instance("source", provider, archiving("pbm", "aws")), pbm,
				storage("aws", pbm, backupv1alpha1.CompatibilityIncompatible, backupv1alpha1.ReasonTLSOptionUnsupported),
			},
			want: outcome{denied: controller.ErrBackupStorageIncompatible, reason: `BackupStorage "aws" is Incompatible with BackupClass "pbm" (TLSOptionUnsupported)`},
		},
		{
			name:    "point in time from a source storage that does not exist yet",
			restore: pointInTime("source", "aws"),
			objects: []client.Object{target, instance("source", provider, archiving("pbm", "aws")), pbm},
			want: outcome{
				missing:  []error{controller.ErrBackupStorageNotFound},
				warnings: []string{"backup storage 'aws' does not exist; the Restore waits until it exists"},
			},
		},
		{
			name:    "point in time of the target's own stream",
			restore: pointInTime("mongo", "aws"),
			objects: []client.Object{target, physical, storage("aws", physical, backupv1alpha1.CompatibilityCompatible, "")},
		},
		{
			name:    "point in time from a storage the stream does not archive to",
			restore: pointInTime("source", "gcs"),
			objects: []client.Object{target, instance("source", provider, archiving("pbm", "aws")), pbm, storage("gcs", pbm, backupv1alpha1.CompatibilityCompatible, "")},
			want:    outcome{denied: controller.ErrRestorePITRUnsupported, reason: `storage "gcs" is not a PITR-enabled storage`},
		},
		{
			name:    "point in time with a class that does not support it",
			restore: pointInTime("source", "aws"),
			objects: []client.Object{target, instance("source", provider, archiving("no-pitr", "aws")), noPITR, storage("aws", noPITR, backupv1alpha1.CompatibilityCompatible, "")},
			want:    outcome{denied: controller.ErrRestorePITRUnsupported, reason: `BackupClass "no-pitr"`},
		},
		{
			name:    "point in time from an instance without a backup class",
			restore: pointInTime("source", "aws"),
			objects: []client.Object{target, instance("source", provider, nil), storage("aws", pbm, backupv1alpha1.CompatibilityCompatible, "")},
			want:    outcome{denied: controller.ErrInvalidReference, reason: "instance 'source' has no backup class configured"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d, err := backupadmission.Restore(context.Background(), reader(t, tt.objects...), tt.restore)
			assertDecision(t, tt.want, d, err)
		})
	}
}

func TestImport(t *testing.T) {
	t.Parallel()

	pbm := class("pbm", backupv1alpha1.CompatibilityCompatible, "")
	noImport := class("no-import", backupv1alpha1.CompatibilityCompatible, "")
	noImport.Spec.SupportsImport = false
	stalePBM := pbm.DeepCopy()
	stalePBM.Generation = 2
	newImport := func(className, storageName string) *backupv1alpha1.BackupImport {
		return &backupv1alpha1.BackupImport{
			ObjectMeta: metav1.ObjectMeta{Name: "i", Namespace: ns},
			Spec: backupv1alpha1.BackupImportSpec{
				ClassRef:   common.ObjectRef{Name: className},
				StorageRef: common.ObjectRef{Name: storageName},
			},
		}
	}

	tests := []struct {
		name    string
		imp     *backupv1alpha1.BackupImport
		objects []client.Object
		want    outcome
	}{
		{
			name:    "compatible",
			imp:     newImport("pbm", "aws"),
			objects: []client.Object{pbm, storage("aws", pbm, backupv1alpha1.CompatibilityCompatible, "")},
		},
		{
			name:    "class does not support import",
			imp:     newImport("no-import", "aws"),
			objects: []client.Object{noImport, storage("aws", noImport, backupv1alpha1.CompatibilityCompatible, "")},
			want:    outcome{denied: controller.ErrBackupImportUnsupported, reason: "class 'no-import'"},
		},
		{
			name:    "class incompatible for its executor",
			imp:     newImport("drift", "aws"),
			objects: []client.Object{class("drift", backupv1alpha1.CompatibilityIncompatible, backupv1alpha1.ReasonSpecDrift), storage("aws", pbm, backupv1alpha1.CompatibilityCompatible, "")},
			want: outcome{
				denied:   controller.ErrBackupClassIncompatible,
				reason:   `BackupClass "drift" is Incompatible with Provider "psmdb" (SpecDrift)`,
				warnings: []string{`BackupStorage "aws" is Pending for BackupClass "drift"`},
			},
		},
		{
			name:    "class verdict for an older generation",
			imp:     newImport("pbm", "aws"),
			objects: []client.Object{stalePBM, storage("aws", pbm, backupv1alpha1.CompatibilityCompatible, "")},
			want: outcome{warnings: []string{
				`BackupClass "pbm" is Pending for Provider "psmdb" (VerdictPending)`,
				`BackupStorage "aws" is Pending for BackupClass "pbm" (VerdictPending)`,
			}},
		},
		{
			name:    "storage incompatible",
			imp:     newImport("pbm", "gcs"),
			objects: []client.Object{pbm, storage("gcs", pbm, backupv1alpha1.CompatibilityIncompatible, backupv1alpha1.ReasonParameterUnsupported)},
			want:    outcome{denied: controller.ErrBackupStorageIncompatible, reason: "(ParameterUnsupported)"},
		},
		{
			name: "class and storage missing",
			imp:  newImport("pbm", "aws"),
			want: outcome{
				missing: []error{controller.ErrBackupClassNotFound, controller.ErrBackupStorageNotFound},
				warnings: []string{
					"backup class not found: 'pbm'; the BackupImport waits until it exists",
					"backup storage 'aws' does not exist; the BackupImport waits until it exists",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d, err := backupadmission.Import(context.Background(), reader(t, tt.objects...), tt.imp)
			assertDecision(t, tt.want, d, err)
		})
	}
}

// A read that fails for any other reason than a missing object fails the
// decision, which is then neither admitted nor denied.
func TestReadError(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, backupv1alpha1.AddToScheme(scheme))
	unavailable := errors.New("etcd unavailable")
	c := fake.NewClientBuilder().WithScheme(scheme).WithInterceptorFuncs(interceptor.Funcs{
		Get: func(context.Context, client.WithWatch, client.ObjectKey, client.Object, ...client.GetOption) error {
			return unavailable
		},
	}).Build()

	_, err := backupadmission.Import(context.Background(), backupadmission.ClientReader{Reader: c}, &backupv1alpha1.BackupImport{
		ObjectMeta: metav1.ObjectMeta{Name: "i", Namespace: ns},
		Spec:       backupv1alpha1.BackupImportSpec{ClassRef: common.ObjectRef{Name: "pbm"}},
	})
	require.ErrorIs(t, err, unavailable)
	require.NotErrorIs(t, err, controller.ErrInvalidReference)
	assert.Contains(t, err.Error(), "failed to get backup class 'pbm'")
}
