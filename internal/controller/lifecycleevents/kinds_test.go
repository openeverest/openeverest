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

package lifecycleevents

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/pkg/events"
)

// observedAt says which generation the Instance's status describes.
func observedAt(in *corev1alpha1.Instance, generation int64) *corev1alpha1.Instance {
	in.Status.ObservedGeneration = generation
	return in
}

// unattended adds core's condition saying that nobody attends to in.
func unattended(in *corev1alpha1.Instance, status metav1.ConditionStatus, reason string) *corev1alpha1.Instance {
	meta.SetStatusCondition(&in.Status.Conditions, metav1.Condition{
		Type: corev1alpha1.ConditionUnattended, Status: status, Reason: reason,
	})
	return in
}

func TestInstanceStatusWord(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		in            *corev1alpha1.Instance
		wantWord      string
		wantAttention bool
	}{
		{
			name:     "never reconciled",
			in:       observedAt(instance(1, "", time.Minute), -1),
			wantWord: "",
		},
		{
			name:     "the provider's word",
			in:       observedAt(instance(1, corev1alpha1.ReasonCreating, time.Minute), 1),
			wantWord: corev1alpha1.ReasonCreating,
		},
		{
			name:          "a new Instance nobody responds to",
			in:            unattended(observedAt(instance(1, "", 10*time.Minute), -1), metav1.ConditionUnknown, corev1alpha1.ReasonNotResponding),
			wantWord:      corev1alpha1.ReasonNotResponding,
			wantAttention: true,
		},
		{
			name: "an edit whose Provider is not installed",
			in: unattended(observedAt(instance(3, corev1alpha1.ReasonReady, time.Hour), 2),
				metav1.ConditionTrue, corev1alpha1.ReasonProviderNotFound,
			),
			wantWord:      corev1alpha1.ReasonProviderNotFound,
			wantAttention: true,
		},
		{
			name: "the condition is answered once the status is current",
			in: unattended(observedAt(instance(3, corev1alpha1.ReasonReady, time.Hour), 3),
				metav1.ConditionUnknown, corev1alpha1.ReasonNotResponding,
			),
			wantWord: corev1alpha1.ReasonReady,
		},
		{
			name: "core never writes it False",
			in: unattended(observedAt(instance(3, corev1alpha1.ReasonReady, time.Hour), 2),
				metav1.ConditionFalse, corev1alpha1.ReasonNotResponding,
			),
			wantWord: corev1alpha1.ReasonReady,
		},
		{
			name:          "a problem the provider reports",
			in:            needingAttention(observedAt(instance(1, "CrashLoopBackOff", time.Hour), 1)),
			wantWord:      "CrashLoopBackOff",
			wantAttention: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.wantWord, instanceStatusWord(tc.in))
			assert.Equal(t, tc.wantAttention, instanceNeedsAttention(tc.in))
		})
	}
}

// Core's verdict is a status word like any other: recorded as a warning on
// the transition to it, and recorded again once a provider answers it.
func TestRecorder_Unattended(t *testing.T) {
	t.Parallel()

	r, emitter := newTestRecorder(t)
	handler := instanceHandler(t, r)
	created := observedAt(instance(1, "", 10*time.Minute), -1)
	verdict := unattended(created.DeepCopy(), metav1.ConditionUnknown, corev1alpha1.ReasonNotResponding)
	verdict.ResourceVersion = "101"
	answered := observedAt(instance(1, corev1alpha1.ReasonCreating, 10*time.Minute), 1)
	answered.ResourceVersion = "102"

	handler.OnUpdate(created, verdict)
	handler.OnUpdate(verdict, verdict.DeepCopy())
	handler.OnUpdate(verdict, answered)

	require.Len(t, emitter.records, 2)
	assert.Equal(t, []events.Reason{events.ReasonPhaseChanged, events.ReasonPhaseChanged}, emitter.reasons())
	assert.Equal(t, corev1alpha1.ReasonNotResponding, emitter.records[0].PhaseTo)
	assert.True(t, emitter.records[0].Warning)
	assert.Equal(t, corev1alpha1.ReasonNotResponding, emitter.records[1].PhaseFrom)
	assert.Equal(t, corev1alpha1.ReasonCreating, emitter.records[1].PhaseTo)
	assert.False(t, emitter.records[1].Warning)
}
