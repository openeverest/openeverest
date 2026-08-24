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

package validation

import (
	"testing"

	"github.com/stretchr/testify/require"

	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

func TestValidateBackupScheduleNames(t *testing.T) {
	t.Parallel()

	storages := func(s ...corev1alpha1.InstanceBackupStorage) *corev1alpha1.Instance {
		return &corev1alpha1.Instance{
			Spec: corev1alpha1.InstanceSpec{
				Backup: &corev1alpha1.InstanceBackupSpec{
					Storages: s,
				},
			},
		}
	}
	storage := func(name string, scheduleNames ...string) corev1alpha1.InstanceBackupStorage {
		schedules := make([]corev1alpha1.InstanceBackupSchedule, 0, len(scheduleNames))
		for _, n := range scheduleNames {
			schedules = append(schedules, corev1alpha1.InstanceBackupSchedule{
				Name: n,
				Cron: "0 0 * * *",
			})
		}
		return corev1alpha1.InstanceBackupStorage{
			StorageRef: common.ObjectRef{Name: name},
			Schedules:  schedules,
		}
	}

	testCases := []struct {
		name     string
		instance *corev1alpha1.Instance
		wantErr  error
	}{
		{
			name:     "nil instance",
			instance: nil,
		},
		{
			name:     "nil backup spec",
			instance: &corev1alpha1.Instance{},
		},
		{
			name:     "no storages",
			instance: storages(),
		},
		{
			name:     "unique names in one storage",
			instance: storages(storage("s1", "daily", "weekly")),
		},
		{
			name:     "unique names across storages",
			instance: storages(storage("s1", "daily"), storage("s2", "weekly")),
		},
		{
			name:     "duplicate names in one storage",
			instance: storages(storage("s1", "daily", "daily")),
			wantErr:  errDuplicatedSchedules,
		},
		{
			name:     "duplicate names across storages",
			instance: storages(storage("s1", "daily"), storage("s2", "daily")),
			wantErr:  errDuplicatedSchedules,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateBackupScheduleNames(tc.instance)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}
