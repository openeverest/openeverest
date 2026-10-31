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
	"github.com/openeverest/openeverest/v2/api/backup/v1alpha1/storageclasses"
	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
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
				Enabled: true,
				Storages: []v1alpha1.InstanceBackupStorage{
					{StorageRef: common.ObjectRef{Name: "s3-prod"}},
				},
			},
		},
	}
	cl := fake.NewClientBuilder().WithScheme(s).WithObjects(append(objs, in)...).Build()
	return NewContext(context.Background(), cl, in, "test-provider")
}

func acceptedBackupStorage(tls *common.TLSConfig) *backupv1alpha1.BackupStorage {
	return &backupv1alpha1.BackupStorage{
		ObjectMeta: metav1.ObjectMeta{Name: "s3-prod", Namespace: "ns"},
		Spec: backupv1alpha1.BackupStorageSpec{
			BackupStorageClassRef: common.ObjectRef{Name: storageclasses.ClassS3},
			Location:              backupv1alpha1.BackupStorageLocation{Bucket: "backups"},
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

	c.RejectBackupStorage("s3-prod", fmt.Errorf("decode: %w", storageclasses.ErrUnsupportedParameter))
	storages, err = c.BackupStorages()
	require.NoError(t, err)
	assert.False(t, storages[0].Apply())
	assert.Equal(t, v1alpha1.ReasonUnsupportedParameter, storages[0].Reason)
	assert.Equal(t, "backup storage s3-prod: decode: unsupported parameter", storages[0].Message)
}

func TestRejectBackupStorage_Reasons(t *testing.T) {
	t.Parallel()

	tests := []struct {
		err  error
		want string
	}{
		{err: storageclasses.ErrUnsupportedParameter, want: v1alpha1.ReasonUnsupportedParameter},
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
