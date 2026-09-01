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
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

func TestStatus_ToV2Alpha1(t *testing.T) {
	t.Parallel()

	status := Provisioning("waiting for cluster...").ToV2Alpha1()

	assert.Equal(t, v1alpha1.InstancePhaseProvisioning, status.Phase)
	assert.Equal(t, "waiting for cluster...", status.Message)
}

func newExternalBackupScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	require.NoError(t, backupv1alpha1.AddToScheme(s))
	require.NoError(t, corev1.AddToScheme(s))
	return s
}

func TestReconcileExternalBackupStatus_Success(t *testing.T) {
	t.Parallel()

	started := metav1.NewTime(time.Now().Add(-time.Hour))
	completed := metav1.NewTime(time.Now())

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<ListBucketResult><KeyCount>1</KeyCount></ListBucketResult>`))
	}))
	t.Cleanup(srv.Close)

	credentials := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "creds", Namespace: "ns"},
		Data: map[string][]byte{
			"AWS_ACCESS_KEY_ID":     []byte("key"),
			"AWS_SECRET_ACCESS_KEY": []byte("secret"),
		},
	}
	storage := &backupv1alpha1.BackupStorage{
		ObjectMeta: metav1.ObjectMeta{Name: "storage", Namespace: "ns"},
		Spec: backupv1alpha1.BackupStorageSpec{
			Type: backupv1alpha1.BackupStorageTypeS3,
			S3: &backupv1alpha1.BackupStorageS3Spec{
				Bucket:               "bucket",
				Region:               "us-east-1",
				EndpointURL:          srv.URL,
				ForcePathStyle:       new(true),
				CredentialsSecretRef: common.SecretRef{Name: "creds"},
			},
		},
	}
	backup := externalBackup(&backupv1alpha1.BackupOriginExternal{
		Path:        "backups/a",
		StartedAt:   started,
		CompletedAt: completed,
	})
	c := fake.NewClientBuilder().
		WithScheme(newExternalBackupScheme(t)).
		WithObjects(storage, credentials).
		Build()

	require.NoError(t, ReconcileExternalBackupStatus(context.Background(), c, backup))

	assert.Equal(t, backupv1alpha1.BackupStateSucceeded, backup.Status.State)
	require.NotNil(t, backup.Status.StartedAt)
	require.NotNil(t, backup.Status.CompletedAt)
	assert.Equal(t, started, *backup.Status.StartedAt)
	assert.Equal(t, completed, *backup.Status.CompletedAt)
}

func externalBackup(external *backupv1alpha1.BackupOriginExternal) *backupv1alpha1.Backup {
	return &backupv1alpha1.Backup{
		ObjectMeta: metav1.ObjectMeta{Name: "imported", Namespace: "ns"},
		Spec: backupv1alpha1.BackupSpec{
			Origin: backupv1alpha1.BackupOrigin{
				Type:     backupv1alpha1.BackupOriginTypeExternal,
				External: external,
			},
			StorageRef: common.ObjectRef{Name: "storage"},
		},
	}
}

func TestReconcileExternalBackupStatus_Failure(t *testing.T) {
	t.Parallel()

	now := metav1.NewTime(time.Now())
	zero := metav1.NewTime(time.Time{})

	tests := []struct {
		name      string
		external  *backupv1alpha1.BackupOriginExternal
		wantState backupv1alpha1.BackupState
	}{
		{
			name: "missing startedAt",
			external: &backupv1alpha1.BackupOriginExternal{
				Path:        "backups/a",
				StartedAt:   zero,
				CompletedAt: now,
			},
			wantState: backupv1alpha1.BackupStateFailed,
		},
		{
			name: "missing completedAt",
			external: &backupv1alpha1.BackupOriginExternal{
				Path:        "backups/a",
				StartedAt:   now,
				CompletedAt: zero,
			},
			wantState: backupv1alpha1.BackupStateFailed,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			backup := externalBackup(tc.external)
			c := fake.NewClientBuilder().
				WithScheme(newExternalBackupScheme(t)).
				Build()

			err := ReconcileExternalBackupStatus(context.Background(), c, backup)

			require.NoError(t, err)
			assert.Equal(t, tc.wantState, backup.Status.State)
		})
	}
}
