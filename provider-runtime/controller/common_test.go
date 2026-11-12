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

func TestStatusHelpers(t *testing.T) {
	t.Parallel()

	details := ConnectionDetails{Type: "mongodb", Host: "db.example", Port: "27017"}
	tests := []struct {
		name string
		got  Status
		want Status
	}{
		{name: "Ready is converged", got: Ready(), want: Status{Converged: true}},
		{
			name: "ReadyWithConnectionDetails is converged, with how to connect",
			got:  ReadyWithConnectionDetails(details),
			want: Status{Converged: true, ConnectionDetails: details},
		},
		{
			name: "Progressing says what it waits for and nothing about availability",
			got:  Progressing("waiting for cluster..."),
			want: Status{Message: "waiting for cluster..."},
		},
		{
			name: "Failed reports a problem that may clear by itself",
			got:  Failed("replica set has no primary"),
			want: Status{
				Message:  "replica set has no primary",
				Problems: []Problem{{Reason: v1alpha1.ReasonEngineError, Message: "replica set has no primary"}},
			},
		},
		{
			name: "the With methods add facts",
			got: Progressing("rolling out").
				WithAvailable(Available).
				WithOperation(v1alpha1.ConditionUpgrading, "MinorVersionRolling", "1 of 3 pods updated").
				WithProblem("ReplicaLagging", "rs0-2 is 40s behind").
				WithTerminalProblem("TopologyChangeUnsupported", "a replica set cannot become sharded"),
			want: Status{
				Message:   "rolling out",
				Available: Available,
				Operations: []Operation{{
					Type: v1alpha1.ConditionUpgrading, Reason: "MinorVersionRolling", Message: "1 of 3 pods updated",
				}},
				Problems: []Problem{
					{Reason: "ReplicaLagging", Message: "rs0-2 is 40s behind"},
					{Reason: "TopologyChangeUnsupported", Message: "a replica set cannot become sharded", Terminal: true},
				},
			},
		},
		{
			name: "WithUnavailable says why clients cannot connect",
			got:  Progressing("electing").WithUnavailable("NoPrimary", "rs0 has no primary"),
			want: Status{
				Message: "electing", Available: Unavailable,
				AvailableReason: "NoPrimary", AvailableMessage: "rs0 has no primary",
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, tc.got)
		})
	}
}

func TestRestoring(t *testing.T) {
	t.Parallel()

	status := Restoring("restoring from nightly")

	assert.False(t, status.Converged)
	assert.Equal(t, "restoring from nightly", status.Message)
	want := []Operation{{
		Type: v1alpha1.ConditionRestoring, Reason: v1alpha1.ConditionRestoring, Message: "restoring from nightly",
	}}
	assert.Equal(t, want, status.Operations, "a restore is a named operation, so it may take as long as it needs")
	assert.Empty(t, status.Problems)
}

// The helpers that named a phase must keep compiling, and must not claim
// anything the phase did not say: they all mean "not converged yet".
func TestDeprecatedStatusHelpers(t *testing.T) {
	t.Parallel()

	for name, helper := range map[string]func(string) Status{
		"Pending":      Pending,
		"Provisioning": Provisioning,
		"Initializing": Initializing,
		"Updating":     Updating,
		"Suspending":   Suspending,
		"Resuming":     Resuming,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, Progressing("waiting"), helper("waiting"))
		})
	}

	suspended := Suspended()
	assert.False(t, suspended.Converged)
	assert.Equal(t, Unavailable, suspended.Available)
	require.Len(t, suspended.Problems, 1)
	assert.False(t, suspended.Problems[0].Terminal)
}

func TestContext_PodLabels(t *testing.T) {
	t.Parallel()

	in := &v1alpha1.Instance{ObjectMeta: metav1.ObjectMeta{Name: "db", Namespace: "ns"}}
	c := NewContext(t.Context(), nil, in, "psmdb")

	assert.Empty(t, c.LabelledComponents())

	want := map[string]string{
		ProviderLabel:  "psmdb",
		InstanceLabel:  "db",
		ComponentLabel: "proxy",
	}
	assert.Equal(t, want, c.PodLabels("proxy"))
	c.PodLabels("engine")
	c.PodLabels("proxy")

	assert.Equal(t, []string{"engine", "proxy"}, c.LabelledComponents())
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
				SourceMetadata: &backupv1alpha1.BackupSourceMetadata{
					EngineVersion: "8.0.12-4",
					BackupType:    "logical",
					Topology:      "replicaSet",
				},
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
				assert.Equal(t, tc.external.SourceMetadata, backup.Status.SourceMetadata)
			}
		})
	}
}
