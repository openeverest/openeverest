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

package summary

import (
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	kstatus "sigs.k8s.io/cli-utils/pkg/kstatus/status"

	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

// t0 is when the stories below begin.
func t0() time.Time { return time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC) }

// instance is the status of one Instance as it is carried from one reconcile
// to the next: its phase, and its conditions, where the checks and operations
// are set by the test and the summary conditions by Compute.
type instance struct {
	t          *testing.T
	generation int64
	// start is when the current generation was first reconciled.
	start time.Time
	// converged is true once the current generation has been Ready.
	converged  bool
	phase      v1alpha1.InstancePhase
	conditions []metav1.Condition
}

func newInstance(t *testing.T) *instance {
	t.Helper()

	return &instance{t: t, generation: 1, start: t0()}
}

// provisioned returns an Instance that has been Ready on its current
// generation since long before the story begins.
func provisioned(t *testing.T) *instance {
	t.Helper()
	in := &instance{t: t, generation: 1, start: t0().Add(-72 * time.Hour)}
	in.observe(in.start, Observation{Reported: true, Converged: true})
	require.Equal(t, v1alpha1.InstancePhaseProvisioned, in.phase)

	return in
}

// set records a check or an operation at the time, the way the reconciler
// does: the transition time moves only when the status changes.
func (in *instance) set(at time.Time, condType string, status metav1.ConditionStatus, reason, message string) {
	c := metav1.Condition{
		Type: condType, Status: status, Reason: reason, Message: message,
		LastTransitionTime: metav1.NewTime(at), ObservedGeneration: in.generation,
	}
	if prior := meta.FindStatusCondition(in.conditions, condType); prior != nil && prior.Status == status {
		c.LastTransitionTime = prior.LastTransitionTime
	}
	in.put(c)
}

func (in *instance) put(c metav1.Condition) {
	if i := slices.IndexFunc(in.conditions, func(have metav1.Condition) bool { return have.Type == c.Type }); i >= 0 {
		in.conditions[i] = c

		return
	}
	in.conditions = append(in.conditions, c)
}

// newGeneration is a change to the spec, first reconciled at the time.
func (in *instance) newGeneration(at time.Time) {
	in.generation++
	in.start = at
	in.converged = false
}

// observe runs one reconcile at the time and stores what it writes. Every
// result is checked against what must always hold.
func (in *instance) observe(at time.Time, o Observation) Result {
	in.t.Helper()
	o.Now = at
	o.Generation = in.generation
	o.Phase = in.phase
	o.Conditions = slices.Clone(in.conditions)
	o.GenerationStart = in.start
	o.HadConverged = in.converged

	res := Compute(o)
	requireInvariants(in.t, &o, res)

	in.phase = res.Phase
	for _, c := range res.Conditions {
		in.put(c)
	}
	in.converged = in.converged || res.Ready()

	return res
}

func (in *instance) condition(condType string) metav1.Condition {
	in.t.Helper()
	c := meta.FindStatusCondition(in.conditions, condType)
	require.NotNil(in.t, c, "condition %s is missing", condType)

	return *c
}

// kstatus is what tools built on kstatus, such as Flux and Helm, make of the
// Instance.
func (in *instance) kstatus(deleting bool) kstatus.Status {
	in.t.Helper()
	obj := &v1alpha1.Instance{
		TypeMeta:   metav1.TypeMeta{APIVersion: v1alpha1.GroupVersion.String(), Kind: "Instance"},
		ObjectMeta: metav1.ObjectMeta{Name: "db", Namespace: "ns", Generation: in.generation},
		Status:     v1alpha1.InstanceStatus{ObservedGeneration: in.generation, Phase: in.phase, Conditions: in.conditions},
	}
	if deleting {
		requested := metav1.NewTime(t0())
		obj.DeletionTimestamp = &requested
		obj.Finalizers = []string{"test"}
	}
	content, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	require.NoError(in.t, err)
	res, err := kstatus.Compute(&unstructured.Unstructured{Object: content})
	require.NoError(in.t, err)

	return res.Status
}

// requireInvariants checks what must hold for every result: the summary
// conditions are all there and describe the current generation, a tool that
// reads Reconciling and Stalled never finds both True, and a Ready Instance
// has neither, nor a problem.
func requireInvariants(t *testing.T, o *Observation, res Result) {
	t.Helper()

	status := make(map[string]metav1.ConditionStatus, len(res.Conditions))
	for _, c := range res.Conditions {
		status[c.Type] = c.Status
		assert.Equal(t, o.Generation, c.ObservedGeneration, "%s must carry the generation it describes", c.Type)
		assert.NotEmpty(t, c.Reason, "%s needs a reason", c.Type)
		assert.False(t, c.LastTransitionTime.IsZero(), "%s needs a transition time", c.Type)
	}
	require.Len(t, status, 5)
	for _, condType := range []string{
		v1alpha1.ConditionAvailable, v1alpha1.ConditionReady, v1alpha1.ConditionReconciling,
		v1alpha1.ConditionStalled, v1alpha1.ConditionNeedsAttention,
	} {
		require.Contains(t, status, condType)
	}

	reconciling := status[v1alpha1.ConditionReconciling] == metav1.ConditionTrue
	stalled := status[v1alpha1.ConditionStalled] == metav1.ConditionTrue
	attention := status[v1alpha1.ConditionNeedsAttention] == metav1.ConditionTrue
	if status[v1alpha1.ConditionReady] == metav1.ConditionTrue {
		assert.False(t, reconciling, "a Ready Instance is not Reconciling")
		assert.False(t, stalled, "a Ready Instance is not Stalled")
		assert.False(t, attention, "a Ready Instance needs no attention")
	} else {
		assert.NotEqual(t, reconciling, stalled, "exactly one of Reconciling and Stalled is True while not Ready")
	}

	problem := false
	if !o.Deleting {
		for _, check := range GatingChecks() {
			if c := meta.FindStatusCondition(o.Conditions, check); c != nil && c.Status == metav1.ConditionFalse {
				problem = true
			}
		}
	}
	if stalled || problem {
		assert.True(t, attention, "NeedsAttention is True while a gating check is False or Stalled is True")
	}
	if !stalled && !problem && !o.Deleting {
		assert.False(t, attention, "NeedsAttention is False when nothing is wrong")
	}
}

// want is what a situation must read as.
type want struct {
	phase       v1alpha1.InstancePhase
	available   metav1.ConditionStatus
	ready       metav1.ConditionStatus
	reason      string
	reconciling bool
	stalled     bool
	attention   bool
	kstatus     kstatus.Status
}

func (in *instance) require(w want, deleting bool) {
	in.t.Helper()
	assert.Equal(in.t, w.phase, in.phase, "phase")
	assert.Equal(in.t, w.available, in.condition(v1alpha1.ConditionAvailable).Status, "Available")
	assert.Equal(in.t, w.ready, in.condition(v1alpha1.ConditionReady).Status, "Ready")
	assert.Equal(in.t, w.reason, in.condition(v1alpha1.ConditionReady).Reason, "the status word")
	assert.Equal(in.t, conditionStatus(w.reconciling), in.condition(v1alpha1.ConditionReconciling).Status, "Reconciling")
	assert.Equal(in.t, conditionStatus(w.stalled), in.condition(v1alpha1.ConditionStalled).Status, "Stalled")
	assert.Equal(in.t, conditionStatus(w.attention), in.condition(v1alpha1.ConditionNeedsAttention).Status, "NeedsAttention")
	assert.Equal(in.t, w.kstatus, in.kstatus(deleting), "kstatus")
}

const (
	pending          = v1alpha1.InstancePhasePending
	provisionedPhase = v1alpha1.InstancePhaseProvisioned
	yes              = metav1.ConditionTrue
	no               = metav1.ConditionFalse
	unknown          = metav1.ConditionUnknown
)

// The situations people meet most, as the runtime writes them. Each story
// sets up what is observed and returns the Instance after one reconcile.
func TestCompute_CommonSituations(t *testing.T) {
	t.Parallel()

	available := Availability{Status: yes}
	unavailable := Availability{Status: no, Reason: "NoPrimary", Message: "No replica set member is primary"}
	crashLoop := func(in *instance, at time.Time) {
		in.set(at, v1alpha1.ConditionPodsReady, no, v1alpha1.ReasonCrashLoopBackOff, "engine: 1 of 3 pods keep crashing")
	}

	tests := []struct {
		name     string
		story    func(t *testing.T) *instance
		deleting bool
		want     want
	}{
		{
			name: "just created",
			story: func(t *testing.T) *instance {
				t.Helper()
				in := newInstance(t)
				in.observe(t0(), Observation{Reported: true, Message: "Waiting for the replica set to initialise"})

				return in
			},
			want: want{phase: pending, available: no, ready: no, reason: v1alpha1.ReasonCreating, reconciling: true, kstatus: kstatus.InProgressStatus},
		},
		{
			name: "created from a backup, data loading",
			story: func(t *testing.T) *instance {
				t.Helper()
				in := newInstance(t)
				in.set(t0(), v1alpha1.ConditionDataSourceReady, unknown, v1alpha1.ReasonDataSourceRestoring, "Restore is running")
				in.set(t0(), v1alpha1.ConditionRestoring, yes, v1alpha1.ReasonSeeding, "Restore is running")
				// Even if the engine answers, what is written now is overwritten.
				in.observe(t0(), Observation{Reported: true, Available: available})

				return in
			},
			want: want{phase: pending, available: no, ready: no, reason: v1alpha1.ConditionRestoring, reconciling: true, kstatus: kstatus.InProgressStatus},
		},
		{
			name: "created with a spec the provider rejects",
			story: func(t *testing.T) *instance {
				t.Helper()
				in := newInstance(t)
				in.observe(t0(), Observation{Rejected: &Cause{Reason: v1alpha1.ReasonValidationFailed, Message: "replicas must be odd"}})

				return in
			},
			want: want{phase: pending, available: no, ready: no, reason: v1alpha1.ReasonValidationFailed, stalled: true, attention: true, kstatus: kstatus.FailedStatus},
		},
		{
			name: "user Secret missing, within the first 10 minutes",
			story: func(t *testing.T) *instance {
				t.Helper()
				in := newInstance(t)
				in.set(t0(), v1alpha1.ConditionReferencesResolved, no, v1alpha1.ReasonUserSecretNotFound, "secret app-users not found")
				in.observe(t0(), Observation{})
				in.observe(t0().Add(DefaultProblemDeadline-time.Second), Observation{})

				return in
			},
			want: want{phase: pending, available: no, ready: no, reason: v1alpha1.ReasonUserSecretNotFound, reconciling: true, attention: true, kstatus: kstatus.InProgressStatus},
		},
		{
			name: "user Secret still missing after 10 minutes",
			story: func(t *testing.T) *instance {
				t.Helper()
				in := newInstance(t)
				in.set(t0(), v1alpha1.ConditionReferencesResolved, no, v1alpha1.ReasonUserSecretNotFound, "secret app-users not found")
				in.observe(t0(), Observation{})
				in.observe(t0().Add(DefaultProblemDeadline), Observation{})

				return in
			},
			want: want{phase: pending, available: no, ready: no, reason: v1alpha1.ReasonUserSecretNotFound, stalled: true, attention: true, kstatus: kstatus.FailedStatus},
		},
		{
			name: "healthy",
			story: func(t *testing.T) *instance {
				t.Helper()
				in := newInstance(t)
				in.set(t0(), v1alpha1.ConditionPodsReady, yes, v1alpha1.ReasonReady, "")
				in.observe(t0(), Observation{Reported: true, Converged: true, Available: available})

				return in
			},
			want: want{phase: provisionedPhase, available: yes, ready: yes, reason: v1alpha1.ReasonReady, kstatus: kstatus.CurrentStatus},
		},
		{
			name: "rolling version upgrade",
			story: func(t *testing.T) *instance {
				t.Helper()
				in := provisioned(t)
				in.newGeneration(t0())
				in.set(t0(), v1alpha1.ConditionUpgrading, yes, "MinorVersionRolling", "1 of 3 pods updated")
				in.set(t0(), v1alpha1.ConditionPodsReady, unknown, v1alpha1.ReasonNotReady, "engine: 1 of 3 pods are not ready")
				in.observe(t0(), Observation{Reported: true, Available: available})

				return in
			},
			want: want{phase: provisionedPhase, available: yes, ready: no, reason: v1alpha1.ConditionUpgrading, reconciling: true, kstatus: kstatus.InProgressStatus},
		},
		{
			name: "any change, on a provider that cannot tell availability",
			story: func(t *testing.T) *instance {
				t.Helper()
				in := provisioned(t)
				in.newGeneration(t0())
				in.observe(t0(), Observation{Reported: true, Message: "Waiting for pods to roll"})

				return in
			},
			want: want{phase: provisionedPhase, available: unknown, ready: no, reason: v1alpha1.ReasonApplyingChanges, reconciling: true, kstatus: kstatus.InProgressStatus},
		},
		{
			name: "a restart that follows a provider upgrade awaits approval",
			story: func(t *testing.T) *instance {
				t.Helper()
				in := provisioned(t)
				in.set(t0(), v1alpha1.ConditionMaintenancePending, yes, v1alpha1.ReasonAwaitingApproval, "1 action(s) require approval to proceed")
				in.observe(t0(), Observation{Reported: true, Converged: true, Available: available})

				return in
			},
			want: want{phase: provisionedPhase, available: yes, ready: yes, reason: v1alpha1.ReasonReady, kstatus: kstatus.CurrentStatus},
		},
		{
			name: "backup storage deleted, within the first 10 minutes",
			story: func(t *testing.T) *instance {
				t.Helper()
				in := provisioned(t)
				in.set(t0(), v1alpha1.ConditionBackupConfigured, no, v1alpha1.ReasonBackupStorageNotFound, "backup storage s3-main not found")
				// The provider is not called, so what it last said about availability stands.
				in.observe(t0(), Observation{Converged: true})
				in.observe(t0().Add(9*time.Minute), Observation{Converged: true})

				return in
			},
			want: want{phase: provisionedPhase, available: yes, ready: no, reason: v1alpha1.ReasonBackupStorageNotFound, reconciling: true, attention: true, kstatus: kstatus.InProgressStatus},
		},
		{
			name: "backup storage still missing after 10 minutes",
			story: func(t *testing.T) *instance {
				t.Helper()
				in := provisioned(t)
				in.set(t0(), v1alpha1.ConditionBackupConfigured, no, v1alpha1.ReasonBackupStorageNotFound, "backup storage s3-main not found")
				in.observe(t0(), Observation{Converged: true})
				in.observe(t0().Add(11*time.Minute), Observation{Converged: true})

				return in
			},
			want: want{phase: provisionedPhase, available: yes, ready: no, reason: v1alpha1.ReasonBackupStorageNotFound, stalled: true, attention: true, kstatus: kstatus.FailedStatus},
		},
		{
			name: "one replica crash-looping, primary serving",
			story: func(t *testing.T) *instance {
				t.Helper()
				in := provisioned(t)
				crashLoop(in, t0())
				in.observe(t0(), Observation{Reported: true, Available: available})

				return in
			},
			want: want{phase: provisionedPhase, available: yes, ready: no, reason: v1alpha1.ReasonCrashLoopBackOff, reconciling: true, attention: true, kstatus: kstatus.InProgressStatus},
		},
		{
			name: "one replica crash-looping for longer than the deadline, primary serving",
			story: func(t *testing.T) *instance {
				t.Helper()
				in := provisioned(t)
				crashLoop(in, t0())
				in.observe(t0(), Observation{Reported: true, Available: available})
				in.observe(t0().Add(DefaultProblemDeadline), Observation{Reported: true, Available: available})

				return in
			},
			want: want{phase: provisionedPhase, available: yes, ready: no, reason: v1alpha1.ReasonCrashLoopBackOff, stalled: true, attention: true, kstatus: kstatus.FailedStatus},
		},
		{
			name: "every replica crash-looping",
			story: func(t *testing.T) *instance {
				t.Helper()
				in := provisioned(t)
				crashLoop(in, t0())
				in.observe(t0(), Observation{Reported: true, Available: unavailable})

				return in
			},
			want: want{phase: provisionedPhase, available: no, ready: no, reason: v1alpha1.ReasonCrashLoopBackOff, reconciling: true, attention: true, kstatus: kstatus.InProgressStatus},
		},
		{
			name: "every replica crash-looping for longer than the deadline",
			story: func(t *testing.T) *instance {
				t.Helper()
				in := provisioned(t)
				crashLoop(in, t0())
				in.observe(t0(), Observation{Reported: true, Available: unavailable})
				in.observe(t0().Add(DefaultProblemDeadline), Observation{Reported: true, Available: unavailable})

				return in
			},
			want: want{phase: provisionedPhase, available: no, ready: no, reason: v1alpha1.ReasonCrashLoopBackOff, stalled: true, attention: true, kstatus: kstatus.FailedStatus},
		},
		{
			name: "in-place restore",
			story: func(t *testing.T) *instance {
				t.Helper()
				in := provisioned(t)
				in.set(t0(), v1alpha1.ConditionRestoring, yes, v1alpha1.ConditionRestoring, "Restore nightly-0412 is running")
				// The spec did not change: the Restore is what someone asked for.
				in.observe(t0(), Observation{
					Reported: true, RestoreInProgress: true,
					Available: Availability{Status: no, Message: "Data is being restored"},
				})

				return in
			},
			want: want{phase: provisionedPhase, available: no, ready: no, reason: v1alpha1.ConditionRestoring, reconciling: true, kstatus: kstatus.InProgressStatus},
		},
		{
			name: "storage resize the engine started, spec unchanged",
			story: func(t *testing.T) *instance {
				t.Helper()
				in := provisioned(t)
				in.set(t0(), v1alpha1.ConditionStorageResizing, yes, "ExpansionTriggered", "Expanding volumes to 50Gi")
				in.observe(t0(), Observation{Reported: true, Converged: true, Available: available})

				return in
			},
			want: want{phase: provisionedPhase, available: yes, ready: yes, reason: v1alpha1.ConditionStorageResizing, kstatus: kstatus.CurrentStatus},
		},
		{
			name: "idle model scaled to zero",
			story: func(t *testing.T) *instance {
				t.Helper()
				in := provisioned(t)
				in.observe(t0(), Observation{
					Reported: true, Converged: true,
					Available: Availability{Status: yes, Reason: "Idle", Message: "The next request starts the model"},
				})
				assert.Equal(t, "Idle", in.condition(v1alpha1.ConditionAvailable).Reason)

				return in
			},
			want: want{phase: provisionedPhase, available: yes, ready: yes, reason: v1alpha1.ReasonReady, kstatus: kstatus.CurrentStatus},
		},
		{
			name: "deleting",
			story: func(t *testing.T) *instance {
				t.Helper()
				in := provisioned(t)
				in.newGeneration(t0())
				in.observe(t0(), Observation{Deleting: true})

				return in
			},
			deleting: true,
			want: want{
				phase: v1alpha1.InstancePhaseTerminating, available: unknown, ready: no, reason: v1alpha1.ReasonDeleting,
				reconciling: true, kstatus: kstatus.TerminatingStatus,
			},
		},
		{
			name: "deleting, waiting for its backups",
			story: func(t *testing.T) *instance {
				t.Helper()
				in := provisioned(t)
				in.newGeneration(t0())
				in.observe(t0(), Observation{
					Deleting: true,
					Waiting:  &Cause{Reason: v1alpha1.ReasonWaitingForBackups, Message: "Waiting for 2 backups to be deleted"},
				})
				assert.Equal(t, "Waiting for 2 backups to be deleted", in.condition(v1alpha1.ConditionReady).Message)

				return in
			},
			deleting: true,
			want: want{
				phase: v1alpha1.InstancePhaseTerminating, available: unknown, ready: no, reason: v1alpha1.ReasonWaitingForBackups,
				reconciling: true, kstatus: kstatus.TerminatingStatus,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.story(t).require(tt.want, tt.deleting)
		})
	}
}

// What decides the status word when several things are true at once.
func TestCompute_Precedence(t *testing.T) {
	t.Parallel()

	now := t0()
	cond := func(condType string, status metav1.ConditionStatus, reason string) metav1.Condition {
		return metav1.Condition{Type: condType, Status: status, Reason: reason, Message: reason + " message", LastTransitionTime: metav1.NewTime(now)}
	}
	crashLoop := cond(v1alpha1.ConditionPodsReady, no, v1alpha1.ReasonCrashLoopBackOff)
	podsStarting := cond(v1alpha1.ConditionPodsReady, unknown, v1alpha1.ReasonNotReady)
	upgrading := cond(v1alpha1.ConditionUpgrading, yes, "MinorVersionRolling")
	retriesExhausted := cond(v1alpha1.ConditionMaintenancePending, yes, v1alpha1.ReasonRetriesExhausted)
	rejected := &Cause{Reason: v1alpha1.ReasonValidationFailed, Message: "replicas must be odd"}
	terminal := &Cause{Reason: "TopologyChangeUnsupported", Message: "a replica set cannot become sharded"}

	tests := []struct {
		name        string
		o           Observation
		wantReady   metav1.ConditionStatus
		wantReason  string
		wantMessage string
		wantStalled bool
	}{
		{
			name:       "a False check outranks any operation",
			o:          Observation{Reported: true, Conditions: []metav1.Condition{upgrading, crashLoop}},
			wantReady:  no,
			wantReason: v1alpha1.ReasonCrashLoopBackOff, wantMessage: "CrashLoopBackOff message",
		},
		{
			name:       "an operation supplies the word, and a check that is Unknown does not",
			o:          Observation{Reported: true, Conditions: []metav1.Condition{podsStarting, upgrading}},
			wantReady:  no,
			wantReason: v1alpha1.ConditionUpgrading, wantMessage: "MinorVersionRolling message",
		},
		{
			name: "of several False checks the first in order names the problem",
			o: Observation{Reported: true, Conditions: []metav1.Condition{
				cond(v1alpha1.ConditionBackupConfigured, no, v1alpha1.ReasonBackupStorageNotFound),
				crashLoop,
				cond(v1alpha1.ConditionPodsScheduled, no, v1alpha1.ReasonUnschedulable),
			}},
			wantReady:  no,
			wantReason: v1alpha1.ReasonUnschedulable, wantMessage: "Unschedulable message",
		},
		{
			name:       "a check that is Unknown leaves the word alone",
			o:          Observation{Reported: true, Message: "Cluster is being created", Conditions: []metav1.Condition{podsStarting}},
			wantReady:  no,
			wantReason: v1alpha1.ReasonCreating, wantMessage: "Cluster is being created",
		},
		{
			name:       "and keeps an Instance whose provider has converged from being Ready",
			o:          Observation{Reported: true, Converged: true, Conditions: []metav1.Condition{podsStarting}},
			wantReady:  no,
			wantReason: v1alpha1.ReasonApplyingChanges, wantMessage: "NotReady message",
		},
		{
			name:       "it explains the wait when the provider does not",
			o:          Observation{Reported: true, Conditions: []metav1.Condition{podsStarting}},
			wantReady:  no,
			wantReason: v1alpha1.ReasonCreating, wantMessage: "NotReady message",
		},
		{
			name: "also for an operation that says nothing about itself",
			o: Observation{Reported: true, Conditions: []metav1.Condition{
				podsStarting,
				{Type: v1alpha1.ConditionUpgrading, Status: yes, Reason: "Rolling", LastTransitionTime: metav1.NewTime(now)},
			}},
			wantReady:  no,
			wantReason: v1alpha1.ConditionUpgrading, wantMessage: "NotReady message",
		},
		{
			name: "once the database has been available the word is the other generic one",
			o: Observation{
				Reported: true, Phase: provisionedPhase, Message: "Waiting for pods to roll",
				Conditions: []metav1.Condition{podsStarting},
			},
			wantReady:  no,
			wantReason: v1alpha1.ReasonApplyingChanges, wantMessage: "Waiting for pods to roll",
		},
		{
			name: "a provider call that fails on a generation still converging is not a problem yet",
			o: Observation{Conditions: []metav1.Condition{
				cond(v1alpha1.ConditionSynced, unknown, v1alpha1.ReasonReconcileError), podsStarting,
			}},
			wantReady:  no,
			wantReason: v1alpha1.ReasonCreating, wantMessage: "ReconcileError message",
		},
		{
			name: "it is one once the check turns False",
			o: Observation{Conditions: []metav1.Condition{
				cond(v1alpha1.ConditionSynced, no, v1alpha1.ReasonReconcileError), podsStarting,
			}},
			wantReady:  no,
			wantReason: v1alpha1.ReasonReconcileError, wantMessage: "ReconcileError message",
		},
		{
			name:       "with nothing more specific the provider's message explains the wait",
			o:          Observation{Reported: true, Message: "Cluster is being created"},
			wantReady:  no,
			wantReason: v1alpha1.ReasonCreating, wantMessage: "Cluster is being created",
		},
		{
			name:       "a rejected spec outranks a problem",
			o:          Observation{Rejected: rejected, Conditions: []metav1.Condition{crashLoop}},
			wantReady:  no,
			wantReason: v1alpha1.ReasonValidationFailed, wantMessage: "replicas must be odd", wantStalled: true,
		},
		{
			name:       "a rejected spec outranks exhausted retries and a terminal problem",
			o:          Observation{Rejected: rejected, TerminalProblem: terminal, Conditions: []metav1.Condition{retriesExhausted}},
			wantReady:  no,
			wantReason: v1alpha1.ReasonValidationFailed, wantMessage: "replicas must be odd", wantStalled: true,
		},
		{
			name:       "exhausted retries stall at once, and outrank a terminal problem",
			o:          Observation{Reported: true, Converged: true, TerminalProblem: terminal, Conditions: []metav1.Condition{retriesExhausted}},
			wantReady:  no,
			wantReason: v1alpha1.ReasonRetriesExhausted, wantMessage: "RetriesExhausted message", wantStalled: true,
		},
		{
			name: "a problem the provider calls terminal stalls at once",
			o: Observation{Reported: true, TerminalProblem: terminal, Conditions: []metav1.Condition{
				cond(v1alpha1.ConditionEngineHealthy, no, terminal.Reason),
			}},
			wantReady:  no,
			wantReason: terminal.Reason, wantMessage: terminal.Message, wantStalled: true,
		},
		{
			name:       "an operation on a generation that has not converged carries out the spec",
			o:          Observation{Reported: true, Converged: true, Conditions: []metav1.Condition{upgrading}},
			wantReady:  no,
			wantReason: v1alpha1.ConditionUpgrading, wantMessage: "MinorVersionRolling message",
		},
		{
			name:       "an operation on a generation that had converged was not asked for",
			o:          Observation{Reported: true, Converged: true, HadConverged: true, Conditions: []metav1.Condition{upgrading}},
			wantReady:  yes,
			wantReason: v1alpha1.ConditionUpgrading, wantMessage: "MinorVersionRolling message",
		},
		{
			name: "unless a Restore asks for it",
			o: Observation{
				Reported: true, Converged: true, HadConverged: true, RestoreInProgress: true,
				Conditions: []metav1.Condition{cond(v1alpha1.ConditionRestoring, yes, "Restoring")},
			},
			wantReady:  no,
			wantReason: v1alpha1.ConditionRestoring, wantMessage: "Restoring message",
		},
		{
			name: "an operation that was not asked for still supplies the word while a check is Unknown",
			o: Observation{
				Reported: true, Converged: true, HadConverged: true,
				Conditions: []metav1.Condition{podsStarting, cond(v1alpha1.ConditionStorageResizing, yes, "ExpansionTriggered")},
			},
			wantReady:  no,
			wantReason: v1alpha1.ConditionStorageResizing, wantMessage: "ExpansionTriggered message",
		},
		{
			name: "of two operations the one that started last supplies the word",
			o: Observation{Reported: true, Conditions: []metav1.Condition{
				upgrading,
				{
					Type: v1alpha1.ConditionStorageResizing, Status: yes, Reason: "ExpansionTriggered", Message: "Expanding volumes",
					LastTransitionTime: metav1.NewTime(now.Add(time.Minute)),
				},
			}},
			wantReady:  no,
			wantReason: v1alpha1.ConditionStorageResizing, wantMessage: "Expanding volumes",
		},
		{
			name: "an operation that says nothing about itself borrows the message",
			o: Observation{Reported: true, Message: "Rolling out", Conditions: []metav1.Condition{
				{Type: v1alpha1.ConditionUpgrading, Status: yes, Reason: "Rolling", LastTransitionTime: metav1.NewTime(now)},
			}},
			wantReady:  no,
			wantReason: v1alpha1.ConditionUpgrading, wantMessage: "Rolling out",
		},
		{
			name: "also on an Instance that stays Ready",
			o: Observation{Reported: true, Converged: true, HadConverged: true, Conditions: []metav1.Condition{
				{Type: v1alpha1.ConditionStorageResizing, Status: yes, Reason: "ExpansionTriggered", LastTransitionTime: metav1.NewTime(now)},
			}},
			wantReady:  yes,
			wantReason: v1alpha1.ConditionStorageResizing, wantMessage: "The instance is ready",
		},
		{
			name: "a notice does not keep an Instance from being Ready",
			o: Observation{Reported: true, Converged: true, Conditions: []metav1.Condition{
				cond(v1alpha1.ConditionMaintenancePending, yes, v1alpha1.ReasonAwaitingApproval),
				cond(v1alpha1.ConditionComponentVersionDeprecated, yes, v1alpha1.ReasonScheduledForRemoval),
				cond(v1alpha1.ConditionConnectionDetailsReady, no, "NotReported"),
				cond(v1alpha1.ConditionMonitoringReady, no, v1alpha1.ReasonMonitoringNotReady),
			}},
			wantReady:  yes,
			wantReason: v1alpha1.ReasonReady, wantMessage: "The instance is ready",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.o.Now = now
			tt.o.Generation = 3
			tt.o.GenerationStart = now

			res := Compute(tt.o)
			requireInvariants(t, &tt.o, res)

			ready := meta.FindStatusCondition(res.Conditions, v1alpha1.ConditionReady)
			assert.Equal(t, tt.wantReady, ready.Status)
			assert.Equal(t, tt.wantReason, ready.Reason)
			assert.Equal(t, tt.wantMessage, ready.Message)
			assert.Equal(t, tt.wantStalled, meta.IsStatusConditionTrue(res.Conditions, v1alpha1.ConditionStalled))
			if tt.wantReady == no {
				// Whichever of the two is True names the same cause as Ready.
				which := v1alpha1.ConditionReconciling
				if tt.wantStalled {
					which = v1alpha1.ConditionStalled
				}
				c := meta.FindStatusCondition(res.Conditions, which)
				assert.Equal(t, tt.wantReason, c.Reason)
				assert.Equal(t, tt.wantMessage, c.Message)
			}
		})
	}
}

func TestCompute_Available(t *testing.T) {
	t.Parallel()

	now := t0()
	cond := func(condType string, status metav1.ConditionStatus, reason, message string) metav1.Condition {
		return metav1.Condition{Type: condType, Status: status, Reason: reason, Message: message, LastTransitionTime: metav1.NewTime(now)}
	}
	seeding := cond(v1alpha1.ConditionDataSourceReady, unknown, v1alpha1.ReasonDataSourceRestoring, "Restore is running")
	wasAvailable := cond(v1alpha1.ConditionAvailable, yes, v1alpha1.ReasonAvailable, "")

	tests := []struct {
		name        string
		o           Observation
		wantStatus  metav1.ConditionStatus
		wantReason  string
		wantMessage string
	}{
		{
			name:       "a provider that does not say is available once it has converged",
			o:          Observation{Reported: true, Converged: true},
			wantStatus: yes, wantReason: v1alpha1.ReasonAvailable,
		},
		{
			name:       "and not available until the database has first been available",
			o:          Observation{Reported: true, Phase: pending},
			wantStatus: no, wantReason: v1alpha1.ReasonNotCreated, wantMessage: messageNotCreated,
		},
		{
			name:       "afterwards nobody can tell: no outage is claimed that the provider cannot see",
			o:          Observation{Reported: true, Phase: provisionedPhase, Conditions: []metav1.Condition{wasAvailable}},
			wantStatus: unknown, wantReason: v1alpha1.ReasonNotReported, wantMessage: messageNotReported,
		},
		{
			name:       "what the provider says is taken as it is",
			o:          Observation{Reported: true, Phase: provisionedPhase, Available: Availability{Status: no, Reason: "NoPrimary", Message: "rs0 has no primary"}},
			wantStatus: no, wantReason: "NoPrimary", wantMessage: "rs0 has no primary",
		},
		{
			name:       "an answer without a reason gets one",
			o:          Observation{Reported: true, Phase: provisionedPhase, Available: Availability{Status: no}},
			wantStatus: no, wantReason: v1alpha1.ReasonNotAvailable,
		},
		{
			name:       "the provider may say it cannot tell",
			o:          Observation{Reported: true, Converged: true, Available: Availability{Status: unknown}},
			wantStatus: unknown, wantReason: v1alpha1.ReasonNotReported,
		},
		{
			name:       "a rolling change on a cluster that stays up",
			o:          Observation{Reported: true, Phase: provisionedPhase, Available: Availability{Status: yes}},
			wantStatus: yes, wantReason: v1alpha1.ReasonAvailable,
		},
		{
			name:       "loading seed data overrides the provider",
			o:          Observation{Reported: true, Converged: true, Available: Availability{Status: yes}, Conditions: []metav1.Condition{seeding}},
			wantStatus: no, wantReason: v1alpha1.ReasonLoadingData, wantMessage: "Restore is running",
		},
		{
			name: "seeding that failed still counts as loading",
			o: Observation{Reported: true, Conditions: []metav1.Condition{
				cond(v1alpha1.ConditionDataSourceReady, no, v1alpha1.ReasonDataSourceFailed, ""),
			}},
			wantStatus: no, wantReason: v1alpha1.ReasonLoadingData, wantMessage: messageLoadingData,
		},
		{
			name: "seeding that succeeded no longer does",
			o: Observation{Reported: true, Converged: true, Conditions: []metav1.Condition{
				cond(v1alpha1.ConditionDataSourceReady, yes, v1alpha1.ReasonDataSourceSucceeded, ""),
				cond(v1alpha1.ConditionAvailable, no, v1alpha1.ReasonLoadingData, "Restore is running"),
			}},
			wantStatus: yes, wantReason: v1alpha1.ReasonAvailable,
		},
		{
			name:       "deletion overrides everything",
			o:          Observation{Deleting: true, Reported: true, Converged: true, Available: Availability{Status: yes}, Conditions: []metav1.Condition{wasAvailable}},
			wantStatus: unknown, wantReason: v1alpha1.ReasonDeleting, wantMessage: messageDeleting,
		},
		{
			name: "a pass that does not ask the provider leaves the last observation",
			o: Observation{Phase: provisionedPhase, Conditions: []metav1.Condition{
				cond(v1alpha1.ConditionAvailable, no, "NoPrimary", "rs0 has no primary"),
			}},
			wantStatus: no, wantReason: "NoPrimary", wantMessage: "rs0 has no primary",
		},
		{
			name:       "a pass that does not ask the provider and has nothing to go by: never been up",
			o:          Observation{},
			wantStatus: no, wantReason: v1alpha1.ReasonNotCreated, wantMessage: messageNotCreated,
		},
		{
			name: "a forced value is not carried over once its cause is gone",
			o: Observation{Phase: pending, Conditions: []metav1.Condition{
				cond(v1alpha1.ConditionAvailable, no, v1alpha1.ReasonLoadingData, "Restore is running"),
			}},
			wantStatus: no, wantReason: v1alpha1.ReasonNotCreated, wantMessage: messageNotCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.o.Now = now
			tt.o.Generation = 2
			tt.o.GenerationStart = now

			res := Compute(tt.o)
			requireInvariants(t, &tt.o, res)

			c := meta.FindStatusCondition(res.Conditions, v1alpha1.ConditionAvailable)
			assert.Equal(t, tt.wantStatus, c.Status)
			assert.Equal(t, tt.wantReason, c.Reason)
			assert.Equal(t, tt.wantMessage, c.Message)
		})
	}
}

func TestCompute_Phase(t *testing.T) {
	t.Parallel()

	t.Run("Pending until the database is first available, then Provisioned for good", func(t *testing.T) {
		t.Parallel()
		in := newInstance(t)

		in.observe(t0(), Observation{Reported: true})
		assert.Equal(t, pending, in.phase)

		in.observe(t0().Add(time.Minute), Observation{Reported: true, Available: Availability{Status: yes}})
		assert.Equal(t, provisionedPhase, in.phase, "available before it has converged")
		assert.Equal(t, v1alpha1.ReasonApplyingChanges, in.condition(v1alpha1.ConditionReady).Reason)

		in.observe(t0().Add(2*time.Minute), Observation{Reported: true, Available: Availability{Status: no}})
		assert.Equal(t, provisionedPhase, in.phase, "the phase never moves back")
	})

	t.Run("a provider that reports only convergence gets there too", func(t *testing.T) {
		t.Parallel()
		in := newInstance(t)

		in.observe(t0(), Observation{Reported: true, Converged: true})

		assert.Equal(t, provisionedPhase, in.phase)
	})

	t.Run("a status that was lost is rebuilt from what is observed", func(t *testing.T) {
		t.Parallel()
		in := provisioned(t)
		in.phase, in.conditions = "", nil

		in.observe(t0(), Observation{Reported: true})
		assert.Equal(t, pending, in.phase, "nothing says it has been up")
		assert.Equal(t, v1alpha1.ReasonCreating, in.condition(v1alpha1.ConditionReady).Reason)

		in.observe(t0().Add(time.Minute), Observation{Reported: true, Converged: true})
		assert.Equal(t, provisionedPhase, in.phase)
	})

	t.Run("a phase this version does not know is rebuilt as well", func(t *testing.T) {
		t.Parallel()
		in := newInstance(t)
		in.phase = "Ready"

		in.observe(t0(), Observation{Reported: true})

		assert.Equal(t, pending, in.phase)
	})

	t.Run("Terminating once deletion is requested", func(t *testing.T) {
		t.Parallel()
		in := newInstance(t)

		in.observe(t0(), Observation{Deleting: true})

		assert.Equal(t, v1alpha1.InstancePhaseTerminating, in.phase)
	})
}

func TestCompute_ProblemDeadline(t *testing.T) {
	t.Parallel()

	secretMissing := func(in *instance, at time.Time) {
		in.set(at, v1alpha1.ConditionReferencesResolved, no, v1alpha1.ReasonUserSecretNotFound, "secret app-users not found")
	}
	stalled := func(in *instance) bool {
		return in.condition(v1alpha1.ConditionStalled).Status == yes
	}

	t.Run("it counts from when the problem began, and says when to look again", func(t *testing.T) {
		t.Parallel()
		in := provisioned(t)

		began := t0().Add(time.Hour)
		secretMissing(in, began)
		res := in.observe(began, Observation{})
		assert.False(t, stalled(in))
		assert.Equal(t, DefaultProblemDeadline, res.Recheck)

		res = in.observe(began.Add(4*time.Minute), Observation{})
		assert.False(t, stalled(in))
		assert.Equal(t, 6*time.Minute, res.Recheck)

		res = in.observe(began.Add(DefaultProblemDeadline), Observation{})
		assert.True(t, stalled(in))
		assert.Zero(t, res.Recheck, "nothing is left to wait for")
		assert.Equal(t, began, in.condition(v1alpha1.ConditionNeedsAttention).LastTransitionTime.Time, "the problem began then")
		assert.Equal(t, began.Add(DefaultProblemDeadline), in.condition(v1alpha1.ConditionStalled).LastTransitionTime.Time)
	})

	t.Run("a problem that clears and comes back starts over", func(t *testing.T) {
		t.Parallel()
		in := newInstance(t)

		secretMissing(in, t0())
		in.observe(t0(), Observation{})
		in.set(t0().Add(8*time.Minute), v1alpha1.ConditionReferencesResolved, yes, v1alpha1.ReasonResolved, "")
		in.observe(t0().Add(8*time.Minute), Observation{Reported: true})
		assert.Equal(t, no, in.condition(v1alpha1.ConditionNeedsAttention).Status)

		secretMissing(in, t0().Add(9*time.Minute))
		in.observe(t0().Add(9*time.Minute), Observation{})
		in.observe(t0().Add(18*time.Minute), Observation{})
		assert.False(t, stalled(in), "9 minutes into the second time")

		in.observe(t0().Add(19*time.Minute), Observation{})
		assert.True(t, stalled(in))
	})

	t.Run("a fix committed while stalled reads as in progress", func(t *testing.T) {
		t.Parallel()
		in := newInstance(t)
		secretMissing(in, t0())
		in.observe(t0(), Observation{})
		in.observe(t0().Add(time.Hour), Observation{})
		require.True(t, stalled(in))
		require.Equal(t, kstatus.FailedStatus, in.kstatus(false))

		// The new spec names another Secret, which does not exist either.
		// The deadline counts from the new generation, not from the hour
		// the old one spent stalled.
		fixed := t0().Add(time.Hour + time.Minute)
		in.newGeneration(fixed)
		res := in.observe(fixed, Observation{})

		assert.False(t, stalled(in))
		assert.Equal(t, yes, in.condition(v1alpha1.ConditionReconciling).Status)
		assert.Equal(t, yes, in.condition(v1alpha1.ConditionNeedsAttention).Status, "the problem is still shown")
		assert.Equal(t, kstatus.InProgressStatus, in.kstatus(false))
		assert.Equal(t, DefaultProblemDeadline, res.Recheck)

		in.observe(fixed.Add(DefaultProblemDeadline), Observation{})
		assert.True(t, stalled(in), "and stalls again if the fix did not help")
	})

	t.Run("a fix committed to a rejected spec reads as in progress", func(t *testing.T) {
		t.Parallel()
		in := newInstance(t)
		in.observe(t0(), Observation{Rejected: &Cause{Reason: v1alpha1.ReasonValidationFailed, Message: "replicas must be odd"}})
		require.True(t, stalled(in))

		fixed := t0().Add(2 * time.Hour)
		in.newGeneration(fixed)
		in.observe(fixed, Observation{Reported: true, Message: "Cluster is being created"})

		assert.False(t, stalled(in))
		assert.Equal(t, v1alpha1.ReasonCreating, in.condition(v1alpha1.ConditionReady).Reason)
		assert.Equal(t, no, in.condition(v1alpha1.ConditionNeedsAttention).Status)
		assert.Equal(t, kstatus.InProgressStatus, in.kstatus(false))
	})

	t.Run("recovery clears the stall", func(t *testing.T) {
		t.Parallel()
		in := provisioned(t)
		in.set(t0(), v1alpha1.ConditionPodsReady, no, v1alpha1.ReasonCrashLoopBackOff, "engine: 1 of 3 pods keep crashing")
		in.observe(t0(), Observation{Reported: true})
		in.observe(t0().Add(time.Hour), Observation{Reported: true})
		require.True(t, stalled(in))

		in.set(t0().Add(2*time.Hour), v1alpha1.ConditionPodsReady, yes, v1alpha1.ReasonReady, "")
		in.observe(t0().Add(2*time.Hour), Observation{Reported: true, Converged: true})

		assert.Equal(t, yes, in.condition(v1alpha1.ConditionReady).Status)
		assert.Equal(t, kstatus.CurrentStatus, in.kstatus(false))
	})

	t.Run("the deadline can be shortened", func(t *testing.T) {
		t.Parallel()
		in := newInstance(t)
		secretMissing(in, t0())

		res := in.observe(t0(), Observation{ProblemDeadline: 30 * time.Second})
		assert.False(t, stalled(in))
		assert.Equal(t, 30*time.Second, res.Recheck)

		in.observe(t0().Add(30*time.Second), Observation{ProblemDeadline: 30 * time.Second})
		assert.True(t, stalled(in))
	})
}

func TestCompute_ProgressDeadline(t *testing.T) {
	t.Parallel()

	creating := Observation{Reported: true, Message: "Waiting for the replica set to initialise"}
	stalled := func(in *instance) bool {
		return in.condition(v1alpha1.ConditionStalled).Status == yes
	}

	t.Run("an unnamed wait stalls, and the stall holds across reconciles", func(t *testing.T) {
		t.Parallel()
		in := newInstance(t)

		res := in.observe(t0(), creating)
		assert.Equal(t, DefaultProgressDeadline, res.Recheck)
		res = in.observe(t0().Add(29*time.Minute), creating)
		assert.False(t, stalled(in))
		assert.Equal(t, time.Minute, res.Recheck)

		due := t0().Add(DefaultProgressDeadline)
		res = in.observe(due, creating)
		require.True(t, stalled(in))
		assert.Zero(t, res.Recheck)
		ready := in.condition(v1alpha1.ConditionReady)
		assert.Equal(t, v1alpha1.ReasonProgressDeadlineExceeded, ready.Reason)
		assert.Equal(t, "No progress for 30m0s: Waiting for the replica set to initialise", ready.Message)
		assert.Equal(t, yes, in.condition(v1alpha1.ConditionNeedsAttention).Status)
		assert.Equal(t, kstatus.FailedStatus, in.kstatus(false))

		// Stalling turned Reconciling False just now. Were the deadline
		// counted from that, the next pass would find the wait young again
		// and undo the stall.
		for _, later := range []time.Duration{time.Second, time.Minute, time.Hour, 48 * time.Hour} {
			in.observe(due.Add(later), creating)
			require.True(t, stalled(in), "still stalled %s later", later)
			assert.Equal(t, due, in.condition(v1alpha1.ConditionStalled).LastTransitionTime.Time, "and since the same time")
		}
		assert.Equal(t, t0(), in.condition(v1alpha1.ConditionReady).LastTransitionTime.Time, "Ready's time did not move")
	})

	t.Run("convergence ends the stall", func(t *testing.T) {
		t.Parallel()
		in := newInstance(t)
		in.observe(t0(), creating)
		in.observe(t0().Add(time.Hour), creating)
		require.True(t, stalled(in))

		in.observe(t0().Add(2*time.Hour), Observation{Reported: true, Converged: true})

		assert.Equal(t, yes, in.condition(v1alpha1.ConditionReady).Status)
		assert.Equal(t, kstatus.CurrentStatus, in.kstatus(false))
	})

	t.Run("a fix committed while stalled reads as in progress", func(t *testing.T) {
		t.Parallel()
		in := newInstance(t)
		in.observe(t0(), creating)
		in.observe(t0().Add(time.Hour), creating)
		require.True(t, stalled(in))

		fixed := t0().Add(90 * time.Minute)
		in.newGeneration(fixed)
		res := in.observe(fixed, creating)

		assert.False(t, stalled(in))
		assert.Equal(t, v1alpha1.ReasonCreating, in.condition(v1alpha1.ConditionReady).Reason)
		assert.Equal(t, no, in.condition(v1alpha1.ConditionNeedsAttention).Status)
		assert.Equal(t, kstatus.InProgressStatus, in.kstatus(false))
		assert.Equal(t, DefaultProgressDeadline, res.Recheck)
	})

	t.Run("a named operation is exempt, and what follows it gets the whole deadline", func(t *testing.T) {
		t.Parallel()
		in := provisioned(t)
		restoring := Observation{Reported: true, RestoreInProgress: true, Available: Availability{Status: no}}

		in.set(t0(), v1alpha1.ConditionRestoring, yes, v1alpha1.ConditionRestoring, "Restore is running")
		res := in.observe(t0(), restoring)
		assert.Zero(t, res.Recheck, "no deadline applies to a named operation")
		in.observe(t0().Add(3*time.Hour), restoring)
		require.False(t, stalled(in), "a restore may take hours")

		// The restore finished three hours after Ready was lost. The pods
		// restart, and the Instance converges a few minutes later.
		ended := t0().Add(3*time.Hour + time.Minute)
		in.set(ended, v1alpha1.ConditionRestoring, no, v1alpha1.ReasonSucceeded, "")
		res = in.observe(ended, Observation{Reported: true, Message: "Waiting for pods"})
		assert.False(t, stalled(in), "straight after the restore")
		assert.Equal(t, v1alpha1.ReasonApplyingChanges, in.condition(v1alpha1.ConditionReady).Reason)
		assert.Equal(t, DefaultProgressDeadline, res.Recheck)

		in.observe(ended.Add(5*time.Minute), Observation{Reported: true, Converged: true})
		assert.Equal(t, yes, in.condition(v1alpha1.ConditionReady).Status)
	})

	t.Run("a problem during a named operation still stalls", func(t *testing.T) {
		t.Parallel()
		in := provisioned(t)
		restoring := Observation{Reported: true, RestoreInProgress: true}
		in.set(t0(), v1alpha1.ConditionRestoring, yes, v1alpha1.ConditionRestoring, "Restore is running")
		in.observe(t0(), restoring)

		began := t0().Add(time.Hour)
		in.set(began, v1alpha1.ConditionPodsReady, no, v1alpha1.ReasonCrashLoopBackOff, "engine: 3 of 3 pods keep crashing")
		in.observe(began, restoring)
		assert.False(t, stalled(in))
		assert.Equal(t, v1alpha1.ReasonCrashLoopBackOff, in.condition(v1alpha1.ConditionReady).Reason)

		in.observe(began.Add(DefaultProblemDeadline), restoring)
		assert.True(t, stalled(in))
	})

	t.Run("recovery from a problem that had stalled gets the whole deadline", func(t *testing.T) {
		t.Parallel()
		in := newInstance(t)
		in.set(t0(), v1alpha1.ConditionReferencesResolved, no, v1alpha1.ReasonUserSecretNotFound, "secret app-users not found")
		in.observe(t0(), Observation{})
		in.observe(t0().Add(2*time.Hour), Observation{})
		require.True(t, stalled(in))

		// The Secret arrives two hours late. Creation only starts now.
		arrived := t0().Add(2*time.Hour + time.Minute)
		in.set(arrived, v1alpha1.ConditionReferencesResolved, yes, v1alpha1.ReasonResolved, "")
		res := in.observe(arrived, creating)
		assert.False(t, stalled(in))
		assert.Equal(t, v1alpha1.ReasonCreating, in.condition(v1alpha1.ConditionReady).Reason)
		assert.Equal(t, DefaultProgressDeadline, res.Recheck)

		in.observe(arrived.Add(29*time.Minute), creating)
		assert.False(t, stalled(in))
		in.observe(arrived.Add(DefaultProgressDeadline), creating)
		assert.True(t, stalled(in))
	})

	t.Run("a cause that comes and goes still stalls, under its own name", func(t *testing.T) {
		t.Parallel()
		in := newInstance(t)
		// A pod is reported as crash-looping for a few minutes at a time,
		// never long enough to outlast the problem deadline.
		for minute := range 30 {
			at := t0().Add(time.Duration(minute) * time.Minute)
			if minute%8 < 4 {
				in.set(at, v1alpha1.ConditionPodsReady, no, v1alpha1.ReasonCrashLoopBackOff, "engine: 1 of 3 pods keep crashing")
			} else {
				in.set(at, v1alpha1.ConditionPodsReady, unknown, v1alpha1.ReasonNotReady, "engine: 1 of 3 pods are not ready")
			}
			in.observe(at, creating)
			require.False(t, stalled(in), "minute %d", minute)
		}

		at := t0().Add(DefaultProgressDeadline)
		in.set(at, v1alpha1.ConditionPodsReady, no, v1alpha1.ReasonCrashLoopBackOff, "engine: 1 of 3 pods keep crashing")
		in.observe(at, creating)

		require.True(t, stalled(in))
		assert.Equal(t, v1alpha1.ReasonCrashLoopBackOff, in.condition(v1alpha1.ConditionReady).Reason, "the word stays the problem's")
		assert.Equal(t, v1alpha1.ReasonProgressDeadlineExceeded, in.condition(v1alpha1.ConditionStalled).Reason)
		assert.Equal(t, "No progress for 30m0s: engine: 1 of 3 pods keep crashing", in.condition(v1alpha1.ConditionStalled).Message)

		later := at.Add(time.Minute)
		in.set(later, v1alpha1.ConditionPodsReady, unknown, v1alpha1.ReasonNotReady, "engine: 1 of 3 pods are not ready")
		in.observe(later, creating)
		assert.True(t, stalled(in), "and holds when the cause goes away again")
	})

	t.Run("a deletion that makes no progress stalls", func(t *testing.T) {
		t.Parallel()
		in := provisioned(t)
		in.newGeneration(t0())
		deleting := Observation{Deleting: true, Message: "waiting for volumes to be released"}

		res := in.observe(t0(), deleting)
		assert.Equal(t, DefaultProgressDeadline, res.Recheck)
		in.observe(t0().Add(DefaultProgressDeadline), deleting)

		require.True(t, stalled(in))
		assert.Equal(t, v1alpha1.ReasonProgressDeadlineExceeded, in.condition(v1alpha1.ConditionReady).Reason)
		assert.Equal(t, "No progress for 30m0s: waiting for volumes to be released", in.condition(v1alpha1.ConditionReady).Message)
		assert.Equal(t, kstatus.TerminatingStatus, in.kstatus(true))
	})

	t.Run("a cleanup that keeps failing stalls a deletion sooner", func(t *testing.T) {
		t.Parallel()
		in := provisioned(t)
		in.set(in.start, v1alpha1.ConditionPodsReady, no, v1alpha1.ReasonCrashLoopBackOff, "left over from before")
		in.newGeneration(t0())
		in.observe(t0(), Observation{Deleting: true})
		assert.Equal(t, v1alpha1.ReasonDeleting, in.condition(v1alpha1.ConditionReady).Reason, "the engine's checks no longer matter")

		in.set(t0().Add(time.Minute), v1alpha1.ConditionSynced, no, v1alpha1.ReasonReconcileError, "delete cluster: forbidden")
		in.observe(t0().Add(time.Minute), Observation{Deleting: true})
		assert.False(t, stalled(in))
		assert.Equal(t, v1alpha1.ReasonReconcileError, in.condition(v1alpha1.ConditionReady).Reason)

		in.observe(t0().Add(time.Minute+DefaultProblemDeadline), Observation{Deleting: true})
		assert.True(t, stalled(in))
	})

	t.Run("the deadline can be shortened", func(t *testing.T) {
		t.Parallel()
		in := newInstance(t)
		short := creating
		short.ProgressDeadline = time.Minute

		res := in.observe(t0(), short)
		assert.Equal(t, time.Minute, res.Recheck)
		in.observe(t0().Add(time.Minute), short)

		require.True(t, stalled(in))
		assert.Equal(t, "No progress for 1m0s: Waiting for the replica set to initialise", in.condition(v1alpha1.ConditionReady).Message)
	})
}

// Pods go not ready and come back many times while a change rolls. That is
// progress, not a cause, so the status word must stay what it is: a word that
// alternates on every pass reads like trouble, and it would be written to the
// status each time.
func TestCompute_ChecksInProgressLeaveTheStatusWordAlone(t *testing.T) {
	t.Parallel()

	rolling := func(in *instance, at time.Time, podsReady bool) {
		if podsReady {
			in.set(at, v1alpha1.ConditionPodsReady, yes, v1alpha1.ReasonReady, "")

			return
		}
		in.set(at, v1alpha1.ConditionPodsReady, unknown, v1alpha1.ReasonNotReady, "engine: 1 of 3 pods are not ready")
	}

	tests := []struct {
		name string
		// start returns the Instance and what the provider reports while it
		// has not converged.
		start       func(t *testing.T) (*instance, Observation)
		wantReason  string
		wantMessage func(podsReady bool) string
	}{
		{
			name: "while it is created",
			start: func(t *testing.T) (*instance, Observation) {
				t.Helper()

				return newInstance(t), Observation{Reported: true, Message: "Cluster is being created"}
			},
			wantReason:  v1alpha1.ReasonCreating,
			wantMessage: func(bool) string { return "Cluster is being created" },
		},
		{
			name: "while a change is applied",
			start: func(t *testing.T) (*instance, Observation) {
				t.Helper()
				in := provisioned(t)
				in.newGeneration(t0())

				return in, Observation{Reported: true, Message: "Waiting for pods to roll", Available: Availability{Status: yes}}
			},
			wantReason:  v1alpha1.ReasonApplyingChanges,
			wantMessage: func(bool) string { return "Waiting for pods to roll" },
		},
		{
			name: "while a provider that explains nothing applies a change: only the sentence follows the pods",
			start: func(t *testing.T) (*instance, Observation) {
				t.Helper()
				in := provisioned(t)
				in.newGeneration(t0())

				return in, Observation{Reported: true}
			},
			wantReason: v1alpha1.ReasonApplyingChanges,
			wantMessage: func(podsReady bool) string {
				if podsReady {
					return messageApplying
				}

				return "engine: 1 of 3 pods are not ready"
			},
		},
		{
			name: "while a named operation runs",
			start: func(t *testing.T) (*instance, Observation) {
				t.Helper()
				in := provisioned(t)
				in.newGeneration(t0())
				in.set(t0(), v1alpha1.ConditionUpgrading, yes, "MinorVersionRolling", "Rolling out 8.0.12")

				return in, Observation{Reported: true, Message: "Waiting for pods to roll"}
			},
			wantReason:  v1alpha1.ConditionUpgrading,
			wantMessage: func(bool) string { return "Rolling out 8.0.12" },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			in, report := tt.start(t)

			for pass := range 12 {
				at := t0().Add(time.Duration(pass) * 5 * time.Second)
				podsReady := pass%2 == 1
				rolling(in, at, podsReady)
				in.observe(at, report)

				ready := in.condition(v1alpha1.ConditionReady)
				require.Equal(t, no, ready.Status, "pass %d", pass)
				require.Equal(t, tt.wantReason, ready.Reason, "pass %d", pass)
				require.Equal(t, tt.wantMessage(podsReady), ready.Message, "pass %d", pass)
				require.Equal(t, tt.wantReason, in.condition(v1alpha1.ConditionReconciling).Reason, "pass %d", pass)
				require.Equal(t, t0(), ready.LastTransitionTime.Time, "pass %d", pass)
				require.Equal(t, kstatus.InProgressStatus, in.kstatus(false), "pass %d", pass)
			}
		})
	}
}

// A condition's transition time is when its status last changed. Clients
// show it as "since", and the deadlines count from it.
func TestCompute_TransitionTimesMoveOnlyWithTheStatus(t *testing.T) {
	t.Parallel()
	in := newInstance(t)

	in.observe(t0(), Observation{Reported: true, Message: "Creating the cluster"})
	in.observe(t0().Add(time.Minute), Observation{Reported: true, Message: "Waiting for the replica set to initialise"})
	ready := in.condition(v1alpha1.ConditionReady)
	assert.Equal(t, "Waiting for the replica set to initialise", ready.Message, "the message follows")
	assert.Equal(t, t0(), ready.LastTransitionTime.Time, "the time does not")

	converged := t0().Add(5 * time.Minute)
	in.observe(converged, Observation{Reported: true, Converged: true})
	for _, condType := range []string{v1alpha1.ConditionReady, v1alpha1.ConditionReconciling, v1alpha1.ConditionAvailable} {
		assert.Equal(t, converged, in.condition(condType).LastTransitionTime.Time, condType)
	}
	assert.Equal(t, t0(), in.condition(v1alpha1.ConditionStalled).LastTransitionTime.Time, "Stalled was False all along")

	in.newGeneration(converged.Add(time.Hour))
	in.observe(converged.Add(time.Hour), Observation{Reported: true, Converged: true})
	ready = in.condition(v1alpha1.ConditionReady)
	assert.Equal(t, int64(2), ready.ObservedGeneration, "the generation follows")
	assert.Equal(t, converged, ready.LastTransitionTime.Time)
}
