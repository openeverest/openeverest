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

	"github.com/stretchr/testify/assert"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
)

func TestShouldRetainBackupData(t *testing.T) {
	tests := []struct {
		name   string
		backup *backupv1alpha1.Backup
		want   bool
	}{
		{
			name:   "nil backup returns false",
			backup: nil,
			want:   false,
		},
		{
			name: "Delete policy returns false",
			backup: &backupv1alpha1.Backup{
				Spec: backupv1alpha1.BackupSpec{
					DeletionPolicy: backupv1alpha1.BackupDeletionPolicyDelete,
				},
			},
			want: false,
		},
		{
			name: "Retain policy returns true",
			backup: &backupv1alpha1.Backup{
				Spec: backupv1alpha1.BackupSpec{
					DeletionPolicy: backupv1alpha1.BackupDeletionPolicyRetain,
				},
			},
			want: true,
		},
		{
			name: "empty policy returns false (zero value)",
			backup: &backupv1alpha1.Backup{
				Spec: backupv1alpha1.BackupSpec{},
			},
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// ShouldRetainBackupData only reads backup.Spec.DeletionPolicy,
			// so we can construct a Context with zero-value fields.
			c := &Context{}
			got := c.ShouldRetainBackupData(tc.backup)
			assert.Equal(t, tc.want, got)
		})
	}
}
