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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

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

func TestReconcileExternalBackupStatus(t *testing.T) {
	t.Parallel()

	now := metav1.NewTime(time.Now())
	zero := metav1.NewTime(time.Time{})

	tests := []struct {
		name          string
		external      *backupv1alpha1.BackupOriginExternal
		expectedState backupv1alpha1.BackupState
	}{
		{
			name: "success",
			external: &backupv1alpha1.BackupOriginExternal{
				Path:        "backups/a",
				StartedAt:   now,
				CompletedAt: metav1.NewTime(time.Now().Add(time.Minute)),
			},
			expectedState: backupv1alpha1.BackupStateSucceeded,
		},
		{
			name: "missing startedAt",
			external: &backupv1alpha1.BackupOriginExternal{
				Path:        "backups/a",
				StartedAt:   zero,
				CompletedAt: now,
			},
			expectedState: backupv1alpha1.BackupStateFailed,
		},
		{
			name: "missing completedAt",
			external: &backupv1alpha1.BackupOriginExternal{
				Path:        "backups/a",
				StartedAt:   now,
				CompletedAt: zero,
			},
			expectedState: backupv1alpha1.BackupStateFailed,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			s := runtime.NewScheme()
			require.NoError(t, backupv1alpha1.AddToScheme(s))
			require.NoError(t, corev1.AddToScheme(s))

			backup := &backupv1alpha1.Backup{
				ObjectMeta: metav1.ObjectMeta{Name: "imported", Namespace: "ns"},
				Spec: backupv1alpha1.BackupSpec{
					Origin: backupv1alpha1.BackupOrigin{
						Type:     backupv1alpha1.BackupOriginTypeExternal,
						External: tc.external,
					},
					StorageRef: common.ObjectRef{Name: "storage"},
				},
			}

			ReconcileExternalBackupStatus(backup)
			assert.Equal(t, tc.expectedState, backup.Status.State)

			if tc.expectedState == backupv1alpha1.BackupStateSucceeded {
				require.NotNil(t, backup.Status.StartedAt)
				require.NotNil(t, backup.Status.CompletedAt)
				assert.Equal(t, tc.external.StartedAt, *backup.Status.StartedAt)
				assert.Equal(t, tc.external.CompletedAt, *backup.Status.CompletedAt)
			}
		})
	}
}
