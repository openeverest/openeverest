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
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
)

// flatProvider is backupTestProvider for an engine that cannot nest.
type flatProvider struct {
	backupTestProvider
}

func (p *flatProvider) BackupPathStyle() backupv1alpha1.BackupPathStyle {
	return backupv1alpha1.BackupPathFlat
}

// flatCountingProvider is countingBackupProvider for an engine that cannot
// nest.
type flatCountingProvider struct {
	countingBackupProvider
}

func (p *flatCountingProvider) BackupPathStyle() backupv1alpha1.BackupPathStyle {
	return backupv1alpha1.BackupPathFlat
}

// TestReconcile_BackupPathStyle checks that the path a provider renders and
// the location the runtime records are laid out in the provider's style.
func TestReconcile_BackupPathStyle(t *testing.T) {
	t.Parallel()

	const (
		nested = "prod/backups/team-a/orders/orders-uid/s3-prod"
		flat   = "prod_backups_team-a_orders_orders-uid_s3-prod"
	)
	tests := []struct {
		name     string
		flat     bool
		recorded string
		want     string
		reason   string
	}{
		{name: "nested", want: nested},
		{name: "flat", flat: true, want: flat},
		{name: "flat, recorded nested before the provider declared the style", flat: true, recorded: nested, want: flat},
		{name: "flat, recorded flat", flat: true, recorded: flat, want: flat},
		{
			name: "flat, recorded elsewhere", flat: true, recorded: "old/backups/team-a/orders/orders-uid/s3-prod",
			want: "old/backups/team-a/orders/orders-uid/s3-prod", reason: v1alpha1.ReasonBackupStorageLocationChanged,
		},
		{
			name: "nested, recorded flat", recorded: flat,
			want: flat, reason: v1alpha1.ReasonBackupStorageLocationChanged,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var rendered []string
			sync := func(c *controller.Context) {
				storages, err := c.BackupStorages()
				if err == nil {
					for _, s := range storages {
						rendered = append(rendered, s.Path)
					}
				}
			}
			var p providerAdapter = &backupTestProvider{readyTestProvider: readyTestProvider{status: controller.Ready()}, sync: sync}
			if tc.flat {
				p = &flatProvider{backupTestProvider: backupTestProvider{readyTestProvider: readyTestProvider{status: controller.Ready()}, sync: sync}}
			}
			in := readyTestInstance(withBackup(true, "s3-prod"))
			in.UID = "orders-uid"
			if tc.recorded != "" {
				withRecordedLocation("s3-prod", tc.recorded)(in)
			}
			r := newReadyTestReconciler(p,
				&v1alpha1.Provider{ObjectMeta: metav1.ObjectMeta{Name: "test-provider"}}, in,
				compatibleBackupClass(), testBackupStorageClass(), testBackupStorage("s3-prod", "s3-prod-keys", metav1.ConditionTrue),
			)

			got, err := reconcileAndGet(t, r)
			require.NoError(t, err)
			wantPath := nested
			if tc.flat {
				wantPath = flat
			}
			assert.Equal(t, []string{wantPath}, rendered, "the path handed to the provider")
			require.NotNil(t, got.Status.Backup)
			require.Len(t, got.Status.Backup.Storages, 1)
			assert.Equal(t, tc.want, got.Status.Backup.Storages[0].Location, "the recorded location")
			assert.Equal(t, tc.reason, got.Status.Backup.Storages[0].Reason)
		})
	}
}

// TestBackupReconcile_PathStyle checks that a Backup records the path of
// its storage entry in the provider's style.
func TestBackupReconcile_PathStyle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		provider controller.BackupProvider
		want     string
	}{
		{name: "nested", provider: &countingBackupProvider{}, want: "prod/backups/team-a/orders/orders-uid/s3-prod"},
		{name: "flat", provider: &flatCountingProvider{}, want: "prod_backups_team-a_orders_orders-uid_s3-prod"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := newOperationTestClient(
				testBackupClass(backupv1alpha1.BackupExecutionModeProviderManaged, "test-provider"), testOperationInstance(),
				testBackupStorage("s3-prod", "s3-keys", metav1.ConditionTrue), testSecret("s3-keys"), testBackup(""),
			)
			r := &backupRuntimeReconciler{client: c, reader: c, provider: tc.provider, providerName: "test-provider"}

			key := client.ObjectKey{Namespace: "team-a", Name: "nightly"}
			_, err := r.Reconcile(t.Context(), reconcile.Request{NamespacedName: key})
			require.NoError(t, err)

			got := &backupv1alpha1.Backup{}
			require.NoError(t, c.Get(t.Context(), key, got))
			assert.Equal(t, tc.want, got.Status.Path)
		})
	}
}
