// everest
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

package instancestatus

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

// fixtures is testdata/display.json: the cases every implementation of the
// display function has to pass, in whatever language it is written.
type fixtures struct {
	Labels map[string]string `json:"labels"`
	Cases  []fixtureCase     `json:"cases"`
}

type fixtureCase struct {
	Name     string          `json:"name"`
	About    string          `json:"about"`
	Now      time.Time       `json:"now"`
	Instance json.RawMessage `json:"instance"`
	Want     struct {
		Tone      Tone         `json:"tone"`
		Reason    string       `json:"reason"`
		Label     string       `json:"label"`
		Available Availability `json:"available"`
		Sentence  string       `json:"sentence"`
		Since     *time.Time   `json:"since"`
		Kstatus   string       `json:"kstatus"`
		Wait      Outcome      `json:"wait"`
	} `json:"want"`
}

func loadFixtures(t *testing.T) fixtures {
	t.Helper()
	raw, err := os.ReadFile("testdata/display.json")
	require.NoError(t, err)
	var f fixtures
	require.NoError(t, json.Unmarshal(raw, &f))
	require.NotEmpty(t, f.Cases)
	require.NotEmpty(t, f.Labels)
	return f
}

func TestDescribe_Fixtures(t *testing.T) {
	t.Parallel()

	for _, tc := range loadFixtures(t).Cases {
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			var inst corev1alpha1.Instance
			require.NoError(t, json.Unmarshal(tc.Instance, &inst))
			in := FromInstance(&inst)

			got := Describe(in, tc.Now)

			assert.Equal(t, tc.Want.Tone, got.Tone, "tone")
			assert.Equal(t, tc.Want.Reason, got.Reason, "reason")
			assert.Equal(t, tc.Want.Label, got.Label, "label")
			assert.Equal(t, tc.Want.Available, got.Available, "available")
			assert.Equal(t, tc.Want.Sentence, got.Sentence, "sentence")
			assert.Equal(t, formatTime(tc.Want.Since), formatTime(got.Since), "since")
			assert.Equal(t, tc.Want.Wait, WaitOutcome(in), "wait")
		})
	}
}

func TestLabels_MatchFixtures(t *testing.T) {
	t.Parallel()

	want := loadFixtures(t).Labels
	for reason, label := range want {
		got, ok := labels[reason]
		if assert.True(t, ok, "reason %q is in the fixtures but not in the dictionary", reason) {
			assert.Equal(t, label, got, "label of %q", reason)
		}
	}
	for reason := range labels {
		assert.Contains(t, want, reason, "reason %q is in the dictionary but not in the fixtures", reason)
	}
}

func TestLabel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		reason string
		want   string
	}{
		{reason: "CrashLoopBackOff", want: "Crash loop"},
		{reason: "RebalancingShards", want: "Rebalancing shards"},
		{reason: "WaitingForBackups", want: "Waiting for backups"},
		{reason: "PodsReady", want: "Pods ready"},
		{reason: "PVCResizeFailed", want: "PVC resize failed"},
		{reason: "S3Unreachable", want: "S3 unreachable"},
		{reason: "PlanB", want: "Plan B"},
		{reason: "Paused", want: "Paused"},
		{reason: "", want: ""},
	}
	for _, tc := range tests {
		t.Run(tc.reason, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, Label(tc.reason))
		})
	}
}

// The cases below are the ones the shared fixtures leave open.
func TestDescribe(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	ago := func(d time.Duration) metav1.Time { return metav1.NewTime(now.Add(-d)) }
	at := func(d time.Duration) *time.Time {
		t := now.Add(-d)
		return &t
	}
	cond := func(conditionType string, status metav1.ConditionStatus, reason string, since time.Duration) metav1.Condition {
		return metav1.Condition{
			Type: conditionType, Status: status, Reason: reason,
			Message: reason + " message", LastTransitionTime: ago(since),
		}
	}
	// healthy is the status of a converged, available Instance.
	healthy := func() []metav1.Condition {
		return []metav1.Condition{
			cond(corev1alpha1.ConditionAvailable, metav1.ConditionTrue, corev1alpha1.ReasonAvailable, 24*time.Hour),
			cond(corev1alpha1.ConditionReady, metav1.ConditionTrue, corev1alpha1.ReasonReady, time.Hour),
			cond(corev1alpha1.ConditionReconciling, metav1.ConditionFalse, corev1alpha1.ReasonNoProblems, time.Hour),
			cond(corev1alpha1.ConditionStalled, metav1.ConditionFalse, corev1alpha1.ReasonNoProblems, time.Hour),
			cond(corev1alpha1.ConditionNeedsAttention, metav1.ConditionFalse, corev1alpha1.ReasonNoProblems, time.Hour),
		}
	}
	// written records a write to the object or to one of its subresources.
	written := func(subresource string, since time.Duration) metav1.ManagedFieldsEntry {
		when := ago(since)
		return metav1.ManagedFieldsEntry{Subresource: subresource, Time: &when}
	}
	deletedAt := ago(time.Second)

	tests := []struct {
		name         string
		in           Input
		wantTone     Tone
		wantReason   string
		wantSentence string
		wantMessage  string
		wantSince    *time.Time
		wantStale    bool
		wantWait     Outcome
	}{
		{
			name:         "no metadata and no status",
			in:           Input{},
			wantTone:     ToneProgress,
			wantReason:   corev1alpha1.ReasonCreating,
			wantSentence: "Creating",
			wantWait:     Pending,
		},
		{
			name: "reconciled, but no Ready condition yet",
			in: Input{
				Meta:               &metav1.ObjectMeta{Generation: 1, CreationTimestamp: ago(time.Hour)},
				ObservedGeneration: 1,
				Phase:              string(corev1alpha1.InstancePhasePending),
			},
			wantTone:     ToneProgress,
			wantReason:   corev1alpha1.ReasonCreating,
			wantSentence: "Creating",
			wantWait:     Pending,
		},
		{
			name: "spec edited a moment ago: the old status decides nothing",
			in: Input{
				Meta: &metav1.ObjectMeta{
					Generation:    2,
					ManagedFields: []metav1.ManagedFieldsEntry{written("", time.Minute), written("status", time.Hour)},
				},
				ObservedGeneration: 1,
				Phase:              string(corev1alpha1.InstancePhaseProvisioned),
				Conditions:         healthy(),
			},
			wantTone:     ToneProgress,
			wantReason:   corev1alpha1.ReasonApplyingChanges,
			wantSentence: "Up · Applying changes",
			wantStale:    true,
			wantWait:     Pending,
		},
		{
			name: "spec edited long ago and never observed: nobody is reconciling",
			in: Input{
				Meta: &metav1.ObjectMeta{
					Generation:    2,
					ManagedFields: []metav1.ManagedFieldsEntry{written("", 6*time.Minute), written("", time.Hour)},
				},
				ObservedGeneration: 1,
				Phase:              string(corev1alpha1.InstancePhaseProvisioned),
				Conditions:         healthy(),
			},
			wantTone:     ToneUnknown,
			wantReason:   ReasonNotPickedUp,
			wantSentence: "Up · Not picked up by provider",
			wantStale:    true,
			wantWait:     Pending,
		},
		{
			name: "a status write does not start the clock again",
			in: Input{
				Meta: &metav1.ObjectMeta{
					Generation:    2,
					ManagedFields: []metav1.ManagedFieldsEntry{written("", 6*time.Minute), written("status", time.Second)},
				},
				ObservedGeneration: 1,
				Phase:              string(corev1alpha1.InstancePhaseProvisioned),
				Conditions:         healthy(),
			},
			wantTone:     ToneUnknown,
			wantReason:   ReasonNotPickedUp,
			wantSentence: "Up · Not picked up by provider",
			wantStale:    true,
			wantWait:     Pending,
		},
		{
			name: "deletion requested, not observed yet: the old status word is not shown",
			in: Input{
				Meta:               &metav1.ObjectMeta{Generation: 3, DeletionTimestamp: &deletedAt},
				ObservedGeneration: 2,
				Phase:              string(corev1alpha1.InstancePhaseProvisioned),
				Conditions:         healthy(),
			},
			wantTone:     ToneDeleting,
			wantReason:   corev1alpha1.ReasonDeleting,
			wantSentence: "Deleting",
			wantSince:    at(time.Second),
			wantStale:    true,
			wantWait:     Failed,
		},
		{
			name: "being resumed: still stopped, and the change is in progress",
			in: Input{
				Meta:               &metav1.ObjectMeta{Generation: 3},
				ObservedGeneration: 2,
				Phase:              string(corev1alpha1.InstancePhaseProvisioned),
				Conditions: []metav1.Condition{
					cond(corev1alpha1.ConditionAvailable, metav1.ConditionFalse, "Suspended", time.Hour),
					cond(corev1alpha1.ConditionReady, metav1.ConditionTrue, "Suspended", time.Hour),
					cond(conditionSuspended, metav1.ConditionTrue, "Suspended", time.Hour),
				},
			},
			wantTone:     ToneStopped,
			wantReason:   corev1alpha1.ReasonApplyingChanges,
			wantSentence: "Applying changes",
			wantStale:    true,
			wantWait:     Pending,
		},
		{
			name: "a stall wins over a problem",
			in: Input{
				Meta:               &metav1.ObjectMeta{Generation: 1},
				ObservedGeneration: 1,
				Phase:              string(corev1alpha1.InstancePhaseProvisioned),
				Conditions: []metav1.Condition{
					cond(corev1alpha1.ConditionAvailable, metav1.ConditionFalse, corev1alpha1.ReasonNotAvailable, time.Hour),
					cond(corev1alpha1.ConditionReady, metav1.ConditionFalse, "DiskFull", time.Hour),
					cond(corev1alpha1.ConditionStalled, metav1.ConditionTrue, "DiskFull", 20*time.Minute),
					cond(corev1alpha1.ConditionNeedsAttention, metav1.ConditionTrue, "DiskFull", 30*time.Minute),
				},
			},
			wantTone:     ToneError,
			wantReason:   "DiskFull",
			wantSentence: "Down · Disk full",
			wantMessage:  "DiskFull message",
			wantSince:    at(20 * time.Minute),
			wantWait:     Failed,
		},
		{
			name: "a finished operation no longer supplies the time",
			in: Input{
				Meta:               &metav1.ObjectMeta{Generation: 1},
				ObservedGeneration: 1,
				Phase:              string(corev1alpha1.InstancePhaseProvisioned),
				Conditions: append(
					healthy(),
					cond(corev1alpha1.ConditionUpgrading, metav1.ConditionFalse, corev1alpha1.ReasonSucceeded, time.Minute),
				),
			},
			wantTone:     ToneOK,
			wantReason:   corev1alpha1.ReasonReady,
			wantSentence: "Up · Ready",
			wantMessage:  "Ready message",
			wantSince:    at(time.Hour),
			wantWait:     Succeeded,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := Describe(tc.in, now)

			assert.Equal(t, tc.wantTone, got.Tone, "tone")
			assert.Equal(t, tc.wantReason, got.Reason, "reason")
			assert.Equal(t, tc.wantSentence, got.Sentence, "sentence")
			assert.Equal(t, tc.wantMessage, got.Message, "message")
			assert.Equal(t, formatTime(tc.wantSince), formatTime(got.Since), "since")
			assert.Equal(t, tc.wantStale, got.Stale, "stale")
			assert.Equal(t, tc.wantWait, WaitOutcome(tc.in), "wait")
		})
	}
}

// formatTime makes two instants comparable whatever their location.
func formatTime(t *time.Time) string {
	if t == nil {
		return "unknown"
	}
	return t.UTC().Format(time.RFC3339)
}
