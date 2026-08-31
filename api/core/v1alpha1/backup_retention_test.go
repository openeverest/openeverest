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

package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/utils/ptr"
)

func TestInstanceBackupScheduleEffectiveRetention(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		schedule *InstanceBackupSchedule
		expected EffectiveRetention
		expectNo bool // expect ok == false
	}{
		{
			name:     "nil schedule keeps all",
			schedule: nil,
			expected: EffectiveRetention{KeepAll: true},
		},
		{
			name:     "no retention and no retentionCopies keeps all",
			schedule: &InstanceBackupSchedule{Name: "daily"},
			expected: EffectiveRetention{KeepAll: true},
		},
		{
			name:     "legacy retentionCopies zero keeps all",
			schedule: &InstanceBackupSchedule{Name: "daily", RetentionCopies: 0},
			expected: EffectiveRetention{KeepAll: true},
		},
		{
			name:     "legacy retentionCopies is honoured unchanged",
			schedule: &InstanceBackupSchedule{Name: "daily", RetentionCopies: 5},
			expected: EffectiveRetention{
				Type:  InstanceBackupScheduleRetentionTypeCount,
				Count: 5,
			},
		},
		{
			name: "explicit count retention",
			schedule: &InstanceBackupSchedule{
				Name: "daily",
				Retention: &InstanceBackupScheduleRetention{
					Type:  InstanceBackupScheduleRetentionTypeCount,
					Count: ptr.To[int32](3),
				},
			},
			expected: EffectiveRetention{
				Type:  InstanceBackupScheduleRetentionTypeCount,
				Count: 3,
			},
		},
		{
			name: "explicit time retention",
			schedule: &InstanceBackupSchedule{
				Name: "daily",
				Retention: &InstanceBackupScheduleRetention{
					Type:     InstanceBackupScheduleRetentionTypeTime,
					Duration: "7d",
				},
			},
			expected: EffectiveRetention{
				Type:     InstanceBackupScheduleRetentionTypeTime,
				Duration: "7d",
			},
		},
		{
			name: "empty type defaults to count",
			schedule: &InstanceBackupSchedule{
				Name: "daily",
				Retention: &InstanceBackupScheduleRetention{
					Count: ptr.To[int32](4),
				},
			},
			expected: EffectiveRetention{
				Type:  InstanceBackupScheduleRetentionTypeCount,
				Count: 4,
			},
		},
		{
			// Enum validation rejects this at admission; the accessor still
			// must not coerce it into a count policy, which would read as
			// "keep 0 backups" and delete everything.
			name: "unknown type is not resolved",
			schedule: &InstanceBackupSchedule{
				Name: "daily",
				Retention: &InstanceBackupScheduleRetention{
					Type: InstanceBackupScheduleRetentionType("weeks"),
				},
			},
			expected: EffectiveRetention{},
			expectNo: true,
		},
		{
			name: "unknown type is not resolved even with retentionCopies set",
			schedule: &InstanceBackupSchedule{
				Name:            "daily",
				RetentionCopies: 7,
				Retention: &InstanceBackupScheduleRetention{
					Type: InstanceBackupScheduleRetentionType("forever"),
				},
			},
			expected: EffectiveRetention{},
			expectNo: true,
		},
		{
			name: "retention count wins over retentionCopies",
			schedule: &InstanceBackupSchedule{
				Name:            "daily",
				RetentionCopies: 9,
				Retention: &InstanceBackupScheduleRetention{
					Type:  InstanceBackupScheduleRetentionTypeCount,
					Count: ptr.To[int32](2),
				},
			},
			expected: EffectiveRetention{
				Type:  InstanceBackupScheduleRetentionTypeCount,
				Count: 2,
			},
		},
		{
			name: "retention time wins over retentionCopies",
			schedule: &InstanceBackupSchedule{
				Name:            "daily",
				RetentionCopies: 9,
				Retention: &InstanceBackupScheduleRetention{
					Type:     InstanceBackupScheduleRetentionTypeTime,
					Duration: "2w",
				},
			},
			expected: EffectiveRetention{
				Type:     InstanceBackupScheduleRetentionTypeTime,
				Duration: "2w",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := tt.schedule.EffectiveRetention()
			assert.Equal(t, !tt.expectNo, ok)
			assert.Equal(t, tt.expected, got)
			if tt.expectNo {
				// An unresolved policy must not look like any usable
				// strategy to a provider.
				assert.False(t, got.IsCount())
				assert.False(t, got.IsTime())
				assert.False(t, got.KeepAll)
			}
		})
	}
}

func TestEffectiveRetentionPredicates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		retention EffectiveRetention
		isCount   bool
		isTime    bool
	}{
		{
			name:      "keep all is neither count nor time",
			retention: EffectiveRetention{KeepAll: true},
		},
		{
			name: "keep all wins even with a populated type",
			retention: EffectiveRetention{
				KeepAll: true,
				Type:    InstanceBackupScheduleRetentionTypeCount,
				Count:   3,
			},
		},
		{
			name: "count policy",
			retention: EffectiveRetention{
				Type:  InstanceBackupScheduleRetentionTypeCount,
				Count: 3,
			},
			isCount: true,
		},
		{
			name: "time policy",
			retention: EffectiveRetention{
				Type:     InstanceBackupScheduleRetentionTypeTime,
				Duration: "6m",
			},
			isTime: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.isCount, tt.retention.IsCount())
			assert.Equal(t, tt.isTime, tt.retention.IsTime())
		})
	}
}
