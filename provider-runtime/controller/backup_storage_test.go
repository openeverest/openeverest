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

package controller

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/backupstorageclasses"
	"github.com/openeverest/openeverest/v2/backupstorageclasses/s3"
)

func newBackupStorageTestContext(t *testing.T, objs ...client.Object) *Context {
	t.Helper()

	s := runtime.NewScheme()
	require.NoError(t, v1alpha1.AddToScheme(s))
	require.NoError(t, backupv1alpha1.AddToScheme(s))
	require.NoError(t, corev1.AddToScheme(s))
	in := &v1alpha1.Instance{
		ObjectMeta: metav1.ObjectMeta{Name: "orders", Namespace: "ns", UID: "orders-uid"},
		Spec: v1alpha1.InstanceSpec{
			Backup: &v1alpha1.InstanceBackupSpec{
				Enabled:  true,
				ClassRef: common.ObjectRef{Name: "pbm"},
				Storages: []v1alpha1.InstanceBackupStorage{
					{StorageRef: common.ObjectRef{Name: "s3-prod"}},
				},
			},
		},
	}
	// Copies, since parallel subtests share objects and Build writes their
	// resourceVersion.
	copies := make([]client.Object, 0, len(objs)+1)
	for _, obj := range objs {
		copied, ok := obj.DeepCopyObject().(client.Object)
		require.True(t, ok)
		copies = append(copies, copied)
	}
	cl := fake.NewClientBuilder().WithScheme(s).WithObjects(append(copies, in)...).Build()
	return NewContext(context.Background(), cl, in, "test-provider")
}

func acceptedBackupStorage(tls *common.TLSConfig) *backupv1alpha1.BackupStorage {
	return &backupv1alpha1.BackupStorage{
		ObjectMeta: metav1.ObjectMeta{Name: "s3-prod", Namespace: "ns"},
		Spec: backupv1alpha1.BackupStorageSpec{
			BackupStorageClassRef: common.ObjectRef{Name: s3.ClassName},
			Location:              backupv1alpha1.BackupStorageLocation{ObjectStore: &backupv1alpha1.ObjectStoreLocation{Bucket: "backups"}},
			TLS:                   tls,
		},
		Status: backupv1alpha1.BackupStorageStatus{
			Conditions: []metav1.Condition{{
				Type:   backupv1alpha1.BackupStorageConditionAccepted,
				Status: metav1.ConditionTrue,
				Reason: backupv1alpha1.ReasonAccepted,
			}},
		},
	}
}

func TestBackupStorages_PathAndRejection(t *testing.T) {
	t.Parallel()

	c := newBackupStorageTestContext(t, acceptedBackupStorage(nil))

	storages, err := c.BackupStorages()
	require.NoError(t, err)
	require.Len(t, storages, 1)
	assert.True(t, storages[0].Apply())
	assert.Equal(t, "s3-prod", storages[0].Name())
	assert.Equal(t, "backups/ns/orders/orders-uid/s3-prod", storages[0].Path)

	c.RejectBackupStorage("s3-prod", fmt.Errorf("decode: %w", backupstorageclasses.ErrUnsupportedParameter))
	storages, err = c.BackupStorages()
	require.NoError(t, err)
	assert.False(t, storages[0].Apply())
	assert.Equal(t, v1alpha1.ReasonUnsupportedParameter, storages[0].Reason)
	assert.Equal(t, "backup storage s3-prod: decode: unsupported parameter", storages[0].Message)
}

// TestBackupStorages_Verdicts checks that an entry is judged by the verdict
// core publishes on the storage for the Instance's class, read through
// CurrentVerdict.
func TestBackupStorages_Verdicts(t *testing.T) {
	t.Parallel()

	class := &backupv1alpha1.BackupClass{ObjectMeta: metav1.ObjectMeta{Name: "pbm", UID: "pbm-uid", Generation: 3}}
	storageClass := &backupv1alpha1.BackupStorageClass{ObjectMeta: metav1.ObjectMeta{Name: s3.ClassName, UID: "s3-uid", Generation: 2}}
	current := backupv1alpha1.BackupClassVerdictInputs{
		ClassUID: "pbm-uid", ClassGeneration: 3, StorageClassUID: "s3-uid", StorageClassGeneration: 2,
	}
	verdict := func(c backupv1alpha1.Compatibility, reason, message string, mutate ...func(*backupv1alpha1.BackupClassVerdictInputs)) backupv1alpha1.BackupClassVerdict {
		v := backupv1alpha1.BackupClassVerdict{Name: "pbm", Compatibility: c, Reason: reason, Message: message, Observed: current}
		for _, m := range mutate {
			m(&v.Observed)
		}
		return v
	}
	storage := func(verdicts ...backupv1alpha1.BackupClassVerdict) *backupv1alpha1.BackupStorage {
		bs := acceptedBackupStorage(nil)
		bs.Status.BackupClasses = verdicts
		return bs
	}

	tests := []struct {
		name        string
		objs        []client.Object
		wantReason  string
		wantMessage string
	}{
		{
			name: "compatible",
			objs: []client.Object{class, storageClass, storage(verdict(backupv1alpha1.CompatibilityCompatible, "", ""))},
		},
		{
			name: "incompatible",
			objs: []client.Object{class, storageClass, storage(verdict(
				backupv1alpha1.CompatibilityIncompatible, backupv1alpha1.ReasonParameterUnsupported, "implementation: gcs is not allowed",
			))},
			wantReason:  v1alpha1.ReasonBackupStorageIncompatible,
			wantMessage: "backup class pbm cannot write to backup storage s3-prod: ParameterUnsupported: implementation: gcs is not allowed",
		},
		{
			name: "pending",
			objs: []client.Object{class, storageClass, storage(verdict(
				backupv1alpha1.CompatibilityPending, backupv1alpha1.ReasonStorageNotAccepted, "not accepted yet",
			))},
			wantReason:  v1alpha1.ReasonBackupStorageVerdictPending,
			wantMessage: "backup storage s3-prod is Pending for backup class pbm: StorageNotAccepted: not accepted yet",
		},
		{
			name:       "no verdict for the class yet",
			objs:       []client.Object{class, storageClass, storage()},
			wantReason: v1alpha1.ReasonBackupStorageVerdictPending,
		},
		{
			name: "verdict for an older generation of the class",
			objs: []client.Object{class, storageClass, storage(verdict(backupv1alpha1.CompatibilityCompatible, "", "",
				func(o *backupv1alpha1.BackupClassVerdictInputs) { o.ClassGeneration = 2 },
			))},
			wantReason: v1alpha1.ReasonBackupStorageVerdictPending,
		},
		{
			name: "verdict for a class recreated under the same name",
			objs: []client.Object{class, storageClass, storage(verdict(backupv1alpha1.CompatibilityCompatible, "", "",
				func(o *backupv1alpha1.BackupClassVerdictInputs) { o.ClassUID = "old-uid" },
			))},
			wantReason: v1alpha1.ReasonBackupStorageVerdictPending,
		},
		{
			name: "verdict for an older generation of the storage class",
			objs: []client.Object{class, storageClass, storage(verdict(backupv1alpha1.CompatibilityCompatible, "", "",
				func(o *backupv1alpha1.BackupClassVerdictInputs) { o.StorageClassGeneration = 1 },
			))},
			wantReason: v1alpha1.ReasonBackupStorageVerdictPending,
		},
		{
			name:       "missing storage class",
			objs:       []client.Object{class, storage(verdict(backupv1alpha1.CompatibilityCompatible, "", ""))},
			wantReason: v1alpha1.ReasonBackupStorageVerdictPending,
		},
		{
			name: "no class: nothing to judge against",
			objs: []client.Object{storageClass, storage()},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := newBackupStorageTestContext(t, tc.objs...)

			storages, err := c.BackupStorages()
			require.NoError(t, err)
			require.Len(t, storages, 1)
			assert.Equal(t, tc.wantReason == "", storages[0].Apply())
			assert.Equal(t, tc.wantReason, storages[0].Reason)
			if tc.wantMessage != "" {
				assert.Equal(t, tc.wantMessage, storages[0].Message)
			}
		})
	}
}

func TestRejectBackupStorage_Reasons(t *testing.T) {
	t.Parallel()

	tests := []struct {
		err  error
		want string
	}{
		{err: backupstorageclasses.ErrUnsupportedParameter, want: v1alpha1.ReasonUnsupportedParameter},
		{err: fmt.Errorf("gcs: %w", ErrImplementationUnsupported), want: v1alpha1.ReasonImplementationUnsupported},
		{err: fmt.Errorf("ca: %w", ErrTLSOptionUnsupported), want: v1alpha1.ReasonTLSOptionUnsupported},
		{err: errors.New("anything else"), want: v1alpha1.ReasonBackupStorageNotAccepted},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()

			// A rejection staged before the first BackupStorages call is
			// applied when it resolves.
			c := newBackupStorageTestContext(t, acceptedBackupStorage(nil))
			c.RejectBackupStorage("s3-prod", tt.err)
			storages, err := c.BackupStorages()
			require.NoError(t, err)
			assert.Equal(t, tt.want, storages[0].Reason)
		})
	}
}

func TestBackupStorageTLS(t *testing.T) {
	t.Parallel()

	caSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "s3-ca", Namespace: "ns"},
		Data:       map[string][]byte{"ca.crt": []byte("PEM")},
	}
	withCA := func(key string) *common.TLSConfig {
		return &common.TLSConfig{CA: &common.CABundleSource{SecretKeyRef: &common.SecretKeyRef{Name: "s3-ca", Key: key}}}
	}
	everything := TLSSupport{CustomCA: true, SkipVerify: true}

	tests := []struct {
		name      string
		tls       *common.TLSConfig
		supported TLSSupport
		want      StorageTLS
		wantErr   error
	}{
		{name: "system trust store", supported: TLSSupport{}},
		{
			name:      "custom CA",
			tls:       withCA("ca.crt"),
			supported: everything,
			want:      StorageTLS{CA: []byte("PEM"), CASecretKeyRef: &common.SecretKeyRef{Name: "s3-ca", Key: "ca.crt"}},
		},
		{name: "custom CA the engine cannot load", tls: withCA("ca.crt"), supported: TLSSupport{SkipVerify: true}, wantErr: ErrTLSOptionUnsupported},
		{name: "skip verification", tls: &common.TLSConfig{InsecureSkipVerify: true}, supported: everything, want: StorageTLS{InsecureSkipVerify: true}},
		{
			name:      "skip verification the engine cannot do",
			tls:       &common.TLSConfig{InsecureSkipVerify: true},
			supported: TLSSupport{CustomCA: true},
			wantErr:   ErrTLSOptionUnsupported,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := newBackupStorageTestContext(t, caSecret)
			got, err := c.BackupStorageTLS(acceptedBackupStorage(tt.tls), tt.supported)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}

	t.Run("CA key missing", func(t *testing.T) {
		t.Parallel()

		c := newBackupStorageTestContext(t, caSecret)
		_, err := c.BackupStorageTLS(acceptedBackupStorage(withCA("other.crt")), everything)
		require.ErrorContains(t, err, `has no key "other.crt"`)
	})
}

func TestBackupLocation(t *testing.T) {
	t.Parallel()

	bs := acceptedBackupStorage(nil)
	bs.Spec.Location.ObjectStore.Prefix = "team-a"

	taken := &backupv1alpha1.Backup{Status: backupv1alpha1.BackupStatus{Path: "team-a/backups/ns/orders/uid/s3-prod"}}
	assert.Equal(t, "team-a/backups/ns/orders/uid/s3-prod", BackupLocation(taken, bs))

	imported := &backupv1alpha1.Backup{Spec: backupv1alpha1.BackupSpec{Origin: backupv1alpha1.BackupOrigin{
		External: &backupv1alpha1.BackupOriginExternal{Path: "legacy/2026-01-01"},
	}}}
	assert.Equal(t, "team-a/legacy/2026-01-01", BackupLocation(imported, bs))
}

// flatStyleProvider declares the flat backup path style.
type flatStyleProvider struct{}

func (flatStyleProvider) BackupPathStyle() backupv1alpha1.BackupPathStyle {
	return backupv1alpha1.BackupPathFlat
}

// emptyStyleProvider declares no style.
type emptyStyleProvider struct{}

func (emptyStyleProvider) BackupPathStyle() backupv1alpha1.BackupPathStyle { return "" }

func TestBackupPathStyleOf(t *testing.T) {
	t.Parallel()
	assert.Equal(t, backupv1alpha1.BackupPathFlat, BackupPathStyleOf(flatStyleProvider{}))
	assert.Equal(t, backupv1alpha1.BackupPathNested, BackupPathStyleOf(emptyStyleProvider{}))
	assert.Equal(t, backupv1alpha1.BackupPathNested, BackupPathStyleOf(&BaseProvider{}))
}

// TestContextBackupLocation checks where a provider of each style finds a
// backup's data: what the runtime recorded, laid out in the style, or for
// an imported backup its path under the style's key prefix.
func TestContextBackupLocation(t *testing.T) {
	t.Parallel()

	bs := acceptedBackupStorage(nil)
	bs.Spec.Location.ObjectStore.Prefix = "team-a"
	taken := func(path string) *backupv1alpha1.Backup {
		return &backupv1alpha1.Backup{Status: backupv1alpha1.BackupStatus{Path: path}}
	}
	imported := func(path string) *backupv1alpha1.Backup {
		return &backupv1alpha1.Backup{Spec: backupv1alpha1.BackupSpec{Origin: backupv1alpha1.BackupOrigin{
			External: &backupv1alpha1.BackupOriginExternal{Path: path},
		}}}
	}

	tests := []struct {
		name   string
		style  backupv1alpha1.BackupPathStyle
		backup *backupv1alpha1.Backup
		want   string
	}{
		{name: "nested, taken", backup: taken("team-a/backups/ns/orders/uid/s3-prod"), want: "team-a/backups/ns/orders/uid/s3-prod"},
		{name: "nested, imported", backup: imported("legacy/2026-01-01"), want: "team-a/legacy/2026-01-01"},
		{
			name: "flat, taken", style: backupv1alpha1.BackupPathFlat,
			backup: taken("team-a_backups_ns_orders_uid_s3-prod"), want: "team-a_backups_ns_orders_uid_s3-prod",
		},
		{
			name: "flat, taken before the provider declared the style", style: backupv1alpha1.BackupPathFlat,
			backup: taken("team-a/backups/ns/orders/uid/s3-prod"), want: "team-a_backups_ns_orders_uid_s3-prod",
		},
		{
			name: "flat, imported", style: backupv1alpha1.BackupPathFlat,
			backup: imported("backups_ns_orders_uid_s3-prod/node-1/2026-01-01"), want: "team-a_backups_ns_orders_uid_s3-prod/node-1/2026-01-01",
		},
		{name: "not recorded", style: backupv1alpha1.BackupPathFlat, backup: taken("")},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := newBackupStorageTestContext(t)
			if tc.style != "" {
				c.SetBackupPathStyle(tc.style)
			}
			assert.Equal(t, tc.want, c.BackupLocation(tc.backup, bs))
		})
	}
}

// TestBackupStorages_PathStyle checks that the path of each entry is laid
// out in the provider's style, and that a location recorded in the nested
// layout is not a change once the provider declares the flat one.
func TestBackupStorages_PathStyle(t *testing.T) {
	t.Parallel()

	const nested = "backups/ns/orders/orders-uid/s3-prod"
	tests := []struct {
		name       string
		style      backupv1alpha1.BackupPathStyle
		recorded   string
		wantPath   string
		wantReason string
	}{
		{name: "nested by default", wantPath: nested},
		{name: "flat", style: backupv1alpha1.BackupPathFlat, wantPath: "backups_ns_orders_orders-uid_s3-prod"},
		{
			name: "flat, recorded nested", style: backupv1alpha1.BackupPathFlat, recorded: nested,
			wantPath: "backups_ns_orders_orders-uid_s3-prod",
		},
		{
			name: "flat, recorded under another prefix", style: backupv1alpha1.BackupPathFlat, recorded: "old/" + nested,
			wantPath: "backups_ns_orders_orders-uid_s3-prod", wantReason: v1alpha1.ReasonBackupStorageLocationChanged,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := newBackupStorageTestContext(t, acceptedBackupStorage(nil))
			c.SetBackupPathStyle(tc.style)
			if tc.recorded != "" {
				c.in.Status.Backup = &v1alpha1.InstanceBackupStatus{Storages: []v1alpha1.InstanceBackupStorageStatus{{Name: "s3-prod", Location: tc.recorded}}}
			}

			storages, err := c.BackupStorages()
			require.NoError(t, err)
			require.Len(t, storages, 1)
			assert.Equal(t, tc.wantPath, storages[0].Path)
			assert.Equal(t, tc.wantReason, storages[0].Reason)
		})
	}
}
