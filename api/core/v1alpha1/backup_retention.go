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

// EffectiveRetention is the resolved retention policy for a backup schedule,
// with the precedence between .retention and the legacy .retentionCopies
// already applied. Providers should consume this instead of reading either
// field directly, so that precedence is implemented exactly once.
//
// It is a computed view, never serialised as part of any API object, so it is
// excluded from deepcopy generation.
// +kubebuilder:object:generate=false
type EffectiveRetention struct {
	// Type is the resolved strategy. It is only meaningful when the policy
	// resolved successfully and KeepAll is false.
	Type InstanceBackupScheduleRetentionType
	// Count is the number of backups to keep when Type is "count".
	Count int32
	// Duration is the retention window (e.g. "7d") when Type is "time".
	Duration string
	// KeepAll reports that the schedule requests unbounded retention: no
	// .retention block and .retentionCopies zero or unset. Providers should
	// leave the engine's retention unconfigured in that case.
	KeepAll bool
}

// IsCount reports whether the resolved policy is bounded and count-based.
func (r EffectiveRetention) IsCount() bool {
	return !r.KeepAll && r.Type == InstanceBackupScheduleRetentionTypeCount
}

// IsTime reports whether the resolved policy is bounded and time-based.
func (r EffectiveRetention) IsTime() bool {
	return !r.KeepAll && r.Type == InstanceBackupScheduleRetentionTypeTime
}

// EffectiveRetention resolves the schedule's retention policy.
//
// The second return value reports whether the policy could be resolved at
// all. It is false only for a .retention block carrying a .type this build
// does not know about — which CRD enum validation already rejects, so it can
// normally only happen when an object was written by a newer API version.
// Callers must check it: on false the returned EffectiveRetention is the zero
// value and carries no meaning, and providers must not treat it as "keep 0".
// Surface it as a condition on the Instance instead.
//
// Precedence, matching the field documentation on InstanceBackupSchedule:
//
//  1. .retention, when set, always wins — including when .retentionCopies is
//     also set, in which case .retentionCopies is ignored.
//  2. otherwise .retentionCopies applies, preserving its historical meaning.
//  3. a .retentionCopies of zero (or unset) with no .retention means "keep
//     all", reported as KeepAll.
//
// A .retention block with an *empty* .type is treated as "count", mirroring
// the CRD default, so that objects created before the default was applied
// resolve the same way. An unknown non-empty .type is not: it returns ok
// false rather than being silently coerced to a count policy.
func (s *InstanceBackupSchedule) EffectiveRetention() (EffectiveRetention, bool) {
	if s == nil {
		return EffectiveRetention{KeepAll: true}, true
	}

	if r := s.Retention; r != nil {
		switch r.Type {
		case InstanceBackupScheduleRetentionTypeTime:
			return EffectiveRetention{
				Type:     InstanceBackupScheduleRetentionTypeTime,
				Duration: r.Duration,
			}, true
		case InstanceBackupScheduleRetentionTypeCount, "":
			// An empty type mirrors the CRD default of "count".
			var count int32
			if r.Count != nil {
				count = *r.Count
			}
			return EffectiveRetention{
				Type:  InstanceBackupScheduleRetentionTypeCount,
				Count: count,
			}, true
		default:
			// Unknown strategy: refuse to guess rather than let a caller
			// read the zero value as "keep 0 backups".
			return EffectiveRetention{}, false
		}
	}

	if s.RetentionCopies > 0 {
		return EffectiveRetention{
			Type:  InstanceBackupScheduleRetentionTypeCount,
			Count: s.RetentionCopies,
		}, true
	}

	return EffectiveRetention{KeepAll: true}, true
}
