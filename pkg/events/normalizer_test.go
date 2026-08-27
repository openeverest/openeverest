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

package events

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
)

func TestNormalizeBackup(t *testing.T) {
	t.Parallel()

	const (
		kind = "Backup"
		name = "b1"
		ns   = "team-a"
		uid  = "backup-uid"
	)

	backup := func(state backupv1alpha1.BackupState) *backupv1alpha1.Backup {
		return &backupv1alpha1.Backup{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: ns,
				UID:       types.UID(uid),
			},
			Status: backupv1alpha1.BackupStatus{State: state},
		}
	}

	tests := []struct {
		name      string
		eventType watch.EventType
		obj       *backupv1alpha1.Backup
		old       *backupv1alpha1.Backup
		wantType  Type
		wantEmpty bool
	}{
		{
			name:      "Added emits backup.started",
			eventType: watch.Added,
			obj:       backup(backupv1alpha1.BackupStatePending),
			wantType:  BackupStarted,
		},
		{
			name:      "Modified into Succeeded emits backup.completed",
			eventType: watch.Modified,
			old:       backup(backupv1alpha1.BackupStateRunning),
			obj:       backup(backupv1alpha1.BackupStateSucceeded),
			wantType:  BackupCompleted,
		},
		{
			name:      "repeated Succeeded emits nothing",
			eventType: watch.Modified,
			old:       backup(backupv1alpha1.BackupStateSucceeded),
			obj:       backup(backupv1alpha1.BackupStateSucceeded),
			wantEmpty: true,
		},
		{
			name:      "Modified into Failed emits backup.failed",
			eventType: watch.Modified,
			old:       backup(backupv1alpha1.BackupStateRunning),
			obj:       backup(backupv1alpha1.BackupStateFailed),
			wantType:  BackupFailed,
		},
		{
			name:      "repeated Failed emits nothing",
			eventType: watch.Modified,
			old:       backup(backupv1alpha1.BackupStateFailed),
			obj:       backup(backupv1alpha1.BackupStateFailed),
			wantEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := NormalizeBackup(watch.Event{Type: tt.eventType, Object: tt.obj}, tt.old)
			assertNormalizeEnvelope(t, got, tt.wantEmpty, tt.wantType, kind, name, ns, uid)
		})
	}
}

func TestNormalizeRestore(t *testing.T) {
	t.Parallel()

	const (
		kind = "Restore"
		name = "r1"
		ns   = "team-a"
		uid  = "restore-uid"
	)

	restore := func(state backupv1alpha1.RestoreState) *backupv1alpha1.Restore {
		return &backupv1alpha1.Restore{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: ns,
				UID:       types.UID(uid),
			},
			Status: backupv1alpha1.RestoreStatus{State: state},
		}
	}

	tests := []struct {
		name      string
		eventType watch.EventType
		obj       *backupv1alpha1.Restore
		old       *backupv1alpha1.Restore
		wantType  Type
		wantEmpty bool
	}{
		{
			name:      "Added emits restore.started",
			eventType: watch.Added,
			obj:       restore(backupv1alpha1.RestoreStatePending),
			wantType:  RestoreStarted,
		},
		{
			name:      "Modified into Succeeded emits restore.completed",
			eventType: watch.Modified,
			old:       restore(backupv1alpha1.RestoreStateRunning),
			obj:       restore(backupv1alpha1.RestoreStateSucceeded),
			wantType:  RestoreCompleted,
		},
		{
			name:      "repeated Succeeded emits nothing",
			eventType: watch.Modified,
			old:       restore(backupv1alpha1.RestoreStateSucceeded),
			obj:       restore(backupv1alpha1.RestoreStateSucceeded),
			wantEmpty: true,
		},
		{
			name:      "Modified into Failed emits restore.failed",
			eventType: watch.Modified,
			old:       restore(backupv1alpha1.RestoreStateRunning),
			obj:       restore(backupv1alpha1.RestoreStateFailed),
			wantType:  RestoreFailed,
		},
		{
			name:      "repeated Failed emits nothing",
			eventType: watch.Modified,
			old:       restore(backupv1alpha1.RestoreStateFailed),
			obj:       restore(backupv1alpha1.RestoreStateFailed),
			wantEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := NormalizeRestore(watch.Event{Type: tt.eventType, Object: tt.obj}, tt.old)
			assertNormalizeEnvelope(t, got, tt.wantEmpty, tt.wantType, kind, name, ns, uid)
		})
	}
}

func assertNormalizeEnvelope(t *testing.T, got []Event, wantEmpty bool, wantType Type, wantKind, wantName, wantNS, wantUID string) {
	t.Helper()
	if wantEmpty {
		assert.Empty(t, got)
		return
	}
	require.Len(t, got, 1)
	assert.Equal(t, wantType, got[0].Type)
	assert.Equal(t, wantKind, got[0].Resource.Kind)
	assert.Equal(t, wantName, got[0].Resource.Name)
	assert.Equal(t, wantUID, got[0].Resource.UID)
	assert.Equal(t, wantNS, got[0].Namespace)
}
