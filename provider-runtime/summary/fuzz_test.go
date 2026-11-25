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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

// FuzzCompute holds every result of Compute to
// contractsummary.CheckConditions and the other invariants, whatever was
// observed and whatever the status held before. Compute runs on what the
// arguments describe, and again a little later on what it wrote, so a
// deadline may pass in between.
func FuzzCompute(f *testing.F) {
	for _, seed := range fuzzSeeds() {
		in := seed.in
		f.Add(in.checks, in.operation, in.flags, in.available, in.prior, in.started, in.since, in.operationAge, in.later)
	}

	f.Fuzz(func(t *testing.T, checks, operation uint8, flags uint16, available, prior uint8, started, since, operationAge, later uint16) {
		in := fuzzInput{
			checks: checks, operation: operation, flags: flags, available: available, prior: prior,
			started: started, since: since, operationAge: operationAge, later: later,
		}
		in.run(t)
	})
}

// The seeds must keep reaching the situations they are named after, or the
// fuzzer would start from fewer of them.
func TestFuzzSeeds(t *testing.T) {
	t.Parallel()

	for _, seed := range fuzzSeeds() {
		t.Run(seed.name, func(t *testing.T) {
			t.Parallel()
			inst := seed.in.run(t)

			assert.Equal(t, conditionStatus(seed.ends == outcomeReady), inst.condition(v1alpha1.ConditionReady).Status, "Ready")
			assert.Equal(t, conditionStatus(seed.ends == outcomeReconciling), inst.condition(v1alpha1.ConditionReconciling).Status, "Reconciling")
			stalled := inst.condition(v1alpha1.ConditionStalled)
			assert.Equal(t, conditionStatus(seed.ends == outcomeStalled), stalled.Status, "Stalled")
			if seed.stall != "" {
				assert.Equal(t, seed.stall, stalled.Reason, "the stall")
			}
			if seed.in.flags&fuzzDeleting != 0 {
				assert.Equal(t, v1alpha1.InstancePhaseTerminating, inst.phase)
			}
		})
	}
}

type fuzzSeed struct {
	name string
	in   fuzzInput
	// ends is the outcome of the second reconcile, and stall the reason of
	// Stalled when that is stalled, if it matters.
	ends  outcome
	stall string
}

func fuzzSeeds() []fuzzSeed {
	ready := fuzzWasReady | fuzzWasNotReconciling | fuzzWasNotStalled | fuzzHadNoProblem
	working := fuzzWasNotReady | fuzzWasReconciling | fuzzWasNotStalled

	return []fuzzSeed{
		{
			name: "ready", ends: outcomeReady,
			in: fuzzInput{
				checks: fuzzCheck(fuzzPodsReady, yes), flags: fuzzReported | fuzzConverged | fuzzHadConverged | fuzzPhaseProvisioned,
				available: fuzzReportsAvailable | fuzzWasAvailable, prior: ready, started: 7200, since: 7200, later: 60,
			},
		},
		{
			name: "an operation the engine started on a spec that had converged", ends: outcomeReady,
			in: fuzzInput{
				operation: fuzzResizing, flags: fuzzReported | fuzzConverged | fuzzHadConverged | fuzzPhaseProvisioned,
				available: fuzzReportsAvailable | fuzzWasAvailable, prior: ready, started: 7200, since: 7200, later: 60,
			},
		},
		{
			name: "being created", ends: outcomeReconciling,
			in: fuzzInput{flags: fuzzReported | fuzzMessage, later: 60},
		},
		{
			name: "seeded from a backup", ends: outcomeReconciling,
			in: fuzzInput{
				checks: fuzzCheck(fuzzDataSourceReady, unknown), operation: fuzzRestoring, flags: fuzzReported | fuzzPhasePending,
				available: fuzzWasLoadingData, prior: working | fuzzHadNoProblem, since: 60, operationAge: 60, later: 60,
			},
		},
		{
			name: "a restore someone asked for, which no deadline applies to", ends: outcomeReconciling,
			in: fuzzInput{
				checks: fuzzCheck(fuzzPodsReady, unknown), operation: fuzzRestoring,
				flags: fuzzReported | fuzzHadConverged | fuzzRestoreInProgress | fuzzPhaseProvisioned,
				prior: working | fuzzHadNoProblem, started: 7200, since: 3600, operationAge: 3600, later: 3600,
			},
		},
		{
			name: "what follows an operation gets the whole progress deadline", ends: outcomeReconciling,
			in: fuzzInput{
				operation: fuzzUpgrading | fuzzOperationEnded, flags: fuzzReported | fuzzPhaseProvisioned,
				prior: working | fuzzHadNoProblem, started: 7200, since: 7200, operationAge: 1, later: 1798,
			},
		},
		{
			name: "a problem within its deadline", ends: outcomeReconciling,
			in: fuzzInput{
				checks: fuzzCheck(fuzzReferencesResolved, no), prior: working | fuzzHadAProblem, started: 598, since: 598, later: 1,
			},
		},
		{
			name: "a problem that outlasts its deadline", ends: outcomeStalled, stall: fuzzCheckReason(v1alpha1.ConditionReferencesResolved, no),
			in: fuzzInput{
				checks: fuzzCheck(fuzzReferencesResolved, no), prior: working | fuzzHadAProblem, started: 599, since: 599, later: 1,
			},
		},
		{
			name: "a rejected spec", ends: outcomeStalled, stall: v1alpha1.ReasonValidationFailed,
			in: fuzzInput{flags: fuzzRejected},
		},
		{
			name: "recovery from a stall", ends: outcomeReady,
			in: fuzzInput{
				checks: fuzzCheck(fuzzPodsReady, yes), flags: fuzzReported | fuzzConverged | fuzzPhaseProvisioned,
				available: fuzzReportsAvailable | fuzzWasAvailable,
				prior:     fuzzWasNotReady | fuzzWasNotReconciling | fuzzWasStalled | fuzzHadAProblem, started: 7200, since: 3600,
			},
		},
		{
			name: "converging past the progress deadline", ends: outcomeStalled, stall: v1alpha1.ReasonProgressDeadlineExceeded,
			in: fuzzInput{flags: fuzzReported | fuzzMessage, prior: working | fuzzHadNoProblem, started: 1799, since: 1799, later: 1},
		},
		{
			name: "held past the progress deadline", ends: outcomeStalled, stall: v1alpha1.ReasonProgressDeadlineExceeded,
			in: fuzzInput{
				flags:   fuzzReported | fuzzMessage,
				prior:   fuzzWasNotReady | fuzzWasNotReconciling | fuzzWasStalledByDeadline | fuzzHadAProblem,
				started: 3600, since: 1800, later: 3600,
			},
		},
		{
			name: "deleting", ends: outcomeReconciling,
			in: fuzzInput{flags: fuzzDeleting | fuzzPhaseProvisioned, prior: ready, since: 7200, later: 60},
		},
		{
			name: "deleting, waiting for its backups", ends: outcomeReconciling,
			in: fuzzInput{
				flags: fuzzDeleting | fuzzWaiting | fuzzPhaseTerminating, prior: working | fuzzHadNoProblem,
				available: fuzzWasDeleting, since: 60, later: 60,
			},
		},
		{
			name: "a deletion past the progress deadline", ends: outcomeStalled, stall: v1alpha1.ReasonProgressDeadlineExceeded,
			in: fuzzInput{
				flags: fuzzDeleting | fuzzPhaseTerminating, prior: working | fuzzHadNoProblem,
				available: fuzzWasDeleting, started: 1799, since: 1799, later: 1,
			},
		},
		{
			name: "a deletion whose cleanup keeps failing", ends: outcomeStalled, stall: fuzzCheckReason(v1alpha1.ConditionSynced, no),
			in: fuzzInput{
				checks: fuzzCheck(fuzzSynced, no), flags: fuzzDeleting | fuzzPhaseTerminating, prior: working | fuzzHadAProblem,
				available: fuzzWasDeleting, started: 900, since: 599, later: 1,
			},
		},
	}
}

// run reconciles twice: once at t0, and again later.
func (in fuzzInput) run(t *testing.T) *instance {
	t.Helper()
	inst := in.instance(t)
	o := in.observation()
	inst.observe(t0(), o)
	inst.observe(t0().Add(seconds(in.later)), o)

	return inst
}

// fuzzInput is what FuzzCompute observes, packed into the primitive types a
// fuzz test takes.
type fuzzInput struct {
	// checks holds two bits per gating check, at the positions below: not
	// set, True, False or Unknown.
	checks uint8
	// operation names the operation in its low two bits, and its status in
	// the next two.
	operation uint8
	flags     uint16
	// available is what the provider reports in its low two bits, and the
	// Available condition written before in the next two.
	available uint8
	// prior holds the summary conditions written before, two bits each.
	prior uint8
	// started is how many seconds ago the generation was first reconciled,
	// since how many seconds ago the summary conditions written before have
	// had their status, and operationAge how long ago the operation's did.
	started, since, operationAge uint16
	// later is how many seconds pass before the second reconcile.
	later uint16
}

// The gating checks, by their position in fuzzInput.checks.
const (
	fuzzReferencesResolved uint8 = iota
	fuzzSynced
	fuzzPodsReady
	fuzzDataSourceReady
)

// fuzzInput.operation.
const (
	fuzzRestoring uint8 = 1
	fuzzUpgrading uint8 = 2
	fuzzResizing  uint8 = 3

	// The status is True unless these bits say False or Unknown.
	fuzzOperationEnded uint8 = 1 << 2
)

// fuzzInput.flags.
const (
	fuzzDeleting uint16 = 1 << iota
	fuzzRejected
	fuzzTerminalProblem
	fuzzConverged
	fuzzHadConverged
	fuzzReported
	fuzzRestoreInProgress
	fuzzWaiting
	fuzzRetriesExhausted
	fuzzMessage
	fuzzShortDeadlines
)

// The stored phase, in the two bits of fuzzInput.flags above the others.
const (
	fuzzPhaseShift              = 12
	fuzzPhasePending     uint16 = 1 << fuzzPhaseShift
	fuzzPhaseProvisioned uint16 = 2 << fuzzPhaseShift
	fuzzPhaseTerminating uint16 = 3 << fuzzPhaseShift
)

// fuzzInput.available.
const (
	fuzzReportsAvailable   uint8 = 1
	fuzzReportsUnavailable uint8 = 2
	fuzzReportsUnknown     uint8 = 3

	fuzzPriorAvailableShift       = 2
	fuzzWasAvailable        uint8 = 1 << fuzzPriorAvailableShift
	fuzzWasLoadingData      uint8 = 2 << fuzzPriorAvailableShift
	fuzzWasDeleting         uint8 = 3 << fuzzPriorAvailableShift
)

// fuzzInput.prior: one of each group.
const (
	fuzzWasReady    uint8 = 1
	fuzzWasNotReady uint8 = 2

	fuzzWasNotStalled        uint8 = 1 << 2
	fuzzWasStalled           uint8 = 2 << 2
	fuzzWasStalledByDeadline uint8 = 3 << 2

	fuzzHadNoProblem uint8 = 1 << 4
	fuzzHadAProblem  uint8 = 2 << 4

	fuzzWasNotReconciling uint8 = 1 << 6
	fuzzWasReconciling    uint8 = 2 << 6
)

// fuzzCheck sets a gating check in fuzzInput.checks.
func fuzzCheck(position uint8, status metav1.ConditionStatus) uint8 {
	var code uint8
	switch status {
	case metav1.ConditionTrue:
		code = 1
	case metav1.ConditionFalse:
		code = 2
	case metav1.ConditionUnknown:
		code = 3
	}

	return code << (2 * position)
}

// instance returns the Instance as the first reconcile finds it.
func (in fuzzInput) instance(t *testing.T) *instance {
	t.Helper()
	now := t0()
	inst := &instance{
		t:          t,
		generation: 1,
		start:      now.Add(-seconds(in.started)),
		converged:  in.flags&fuzzHadConverged != 0,
		phase: []v1alpha1.InstancePhase{
			"", v1alpha1.InstancePhasePending, v1alpha1.InstancePhaseProvisioned, v1alpha1.InstancePhaseTerminating,
		}[in.flags>>fuzzPhaseShift&3],
	}

	statuses := []metav1.ConditionStatus{"", yes, no, unknown}
	since := now.Add(-seconds(in.since))
	for _, prior := range []struct {
		condType string
		status   metav1.ConditionStatus
		reason   string
	}{
		{v1alpha1.ConditionReady, []metav1.ConditionStatus{"", yes, no, no}[in.prior&3], v1alpha1.ReasonCreating},
		{v1alpha1.ConditionStalled, []metav1.ConditionStatus{"", no, yes, yes}[in.prior>>2&3], v1alpha1.ReasonCrashLoopBackOff},
		{v1alpha1.ConditionNeedsAttention, []metav1.ConditionStatus{"", no, yes, yes}[in.prior>>4&3], v1alpha1.ReasonCrashLoopBackOff},
		{v1alpha1.ConditionReconciling, []metav1.ConditionStatus{"", no, yes, yes}[in.prior>>6&3], v1alpha1.ReasonCreating},
		{v1alpha1.ConditionAvailable, statuses[in.available>>fuzzPriorAvailableShift&3], v1alpha1.ReasonAvailable},
	} {
		if prior.status != "" {
			inst.set(since, prior.condType, prior.status, prior.reason, "")
		}
	}
	// What only the reason tells apart.
	if in.prior&fuzzWasStalledByDeadline == fuzzWasStalledByDeadline {
		inst.set(since, v1alpha1.ConditionStalled, yes, v1alpha1.ReasonProgressDeadlineExceeded, "")
	}
	switch in.available & (3 << fuzzPriorAvailableShift) {
	case fuzzWasLoadingData:
		inst.set(since, v1alpha1.ConditionAvailable, no, v1alpha1.ReasonLoadingData, "")
	case fuzzWasDeleting:
		inst.set(since, v1alpha1.ConditionAvailable, unknown, v1alpha1.ReasonDeleting, "")
	}

	for i, check := range []string{
		v1alpha1.ConditionReferencesResolved, v1alpha1.ConditionSynced,
		v1alpha1.ConditionPodsReady, v1alpha1.ConditionDataSourceReady,
	} {
		if status := statuses[in.checks>>(2*i)&3]; status != "" {
			inst.set(now, check, status, fuzzCheckReason(check, status), check+" is "+string(status))
		}
	}
	if operation := in.operation & 3; operation != 0 {
		condType := []string{"", v1alpha1.ConditionRestoring, v1alpha1.ConditionUpgrading, v1alpha1.ConditionStorageResizing}[operation]
		status := []metav1.ConditionStatus{yes, no, unknown, yes}[in.operation>>2&3]
		inst.set(now.Add(-seconds(in.operationAge)), condType, status, condType, condType+" is "+string(status))
	}
	if in.flags&fuzzRetriesExhausted != 0 {
		inst.set(now, v1alpha1.ConditionMaintenancePending, yes, v1alpha1.ReasonRetriesExhausted, "the restart failed 3 times")
	}

	return inst
}

// observation returns what the provider and the runtime report on each
// reconcile. instance.observe fills in the rest.
func (in fuzzInput) observation() Observation {
	o := Observation{
		Deleting:          in.flags&fuzzDeleting != 0,
		Converged:         in.flags&fuzzConverged != 0,
		Reported:          in.flags&fuzzReported != 0,
		RestoreInProgress: in.flags&fuzzRestoreInProgress != 0,
	}
	if in.flags&fuzzRejected != 0 {
		o.Rejected = &Cause{Reason: v1alpha1.ReasonValidationFailed, Message: "replicas must be odd"}
	}
	if in.flags&fuzzTerminalProblem != 0 {
		o.TerminalProblem = &Cause{Reason: "TopologyChangeUnsupported", Message: "a replica set cannot become sharded"}
	}
	if in.flags&fuzzWaiting != 0 {
		o.Waiting = &Cause{Reason: v1alpha1.ReasonWaitingForBackups, Message: "Waiting for 2 backups to be deleted"}
	}
	if in.flags&fuzzMessage != 0 {
		o.Message = "Waiting for pods"
	}
	if in.flags&fuzzShortDeadlines != 0 {
		o.ProblemDeadline, o.ProgressDeadline = time.Minute, 3*time.Minute
	}
	switch in.available & 3 {
	case fuzzReportsAvailable:
		o.Available = Availability{Status: yes}
	case fuzzReportsUnavailable:
		o.Available = Availability{Status: no, Reason: "NoPrimary", Message: "No replica set member is primary"}
	case fuzzReportsUnknown:
		o.Available = Availability{Status: unknown}
	}

	return o
}

// fuzzCheckReason is the reason of a gating check with the status.
func fuzzCheckReason(check string, status metav1.ConditionStatus) string {
	return check + string(status)
}

func seconds(n uint16) time.Duration {
	return time.Duration(n) * time.Second
}
