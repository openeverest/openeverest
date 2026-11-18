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

// Package summary derives an Instance's phase and summary conditions from
// what was observed about it.
//
// It is the one place that decides whether an Instance is Ready, whether to
// wait (Reconciling) or act (Stalled), whether clients can connect
// (Available) and which status word it shows, so that every provider
// describes the same situation the same way. It reads no clock and no
// cluster: everything it needs is in the Observation.
package summary

import (
	"cmp"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

const (
	// DefaultProblemDeadline is how long a problem may be observed before
	// the Instance is reported as stalled. Most problems clear by
	// themselves, such as a Secret that arrives late or a node being added,
	// and tools that read Stalled give up at once.
	DefaultProblemDeadline = 10 * time.Minute

	// DefaultProgressDeadline is how long an Instance may converge with no
	// named operation running before it is reported as stalled.
	DefaultProgressDeadline = 30 * time.Minute
)

const (
	messageReady       = "The instance is ready"
	messageCreating    = "The database is being created"
	messageApplying    = "Changes are being applied"
	messageDeleting    = "The instance is being deleted"
	messageNotCreated  = "The database has not been available yet"
	messageNotReported = "The provider does not report availability while converging"
	messageLoadingData = "Data is being loaded from the data source"
)

// GatingChecks returns the checks that keep an Instance from being Ready
// unless they are True. When several are False the first one names the
// problem, so they are ordered from cause to consequence: a missing referent
// keeps the provider from running, a failing provider leaves the rest
// unobserved, and a pod with no node cannot be ready.
func GatingChecks() []string {
	return []string{
		v1alpha1.ConditionReferencesResolved,
		v1alpha1.ConditionSynced,
		v1alpha1.ConditionPodsScheduled,
		v1alpha1.ConditionPodsReady,
		v1alpha1.ConditionEngineHealthy,
		v1alpha1.ConditionDataSourceReady,
		v1alpha1.ConditionBackupConfigured,
	}
}

// Operations returns the conditions that report a long operation in
// progress.
func Operations() []string {
	return []string{
		v1alpha1.ConditionRestoring,
		v1alpha1.ConditionUpgrading,
		v1alpha1.ConditionStorageResizing,
	}
}

// Cause is why an Instance is in the state it is in: a reason, which becomes
// the status word, and one sentence a person can act on.
type Cause struct {
	Reason  string
	Message string
}

// orMessage returns the cause with the fallback as its message if it has
// none.
func (c Cause) orMessage(fallback string) Cause {
	c.Message = cmp.Or(c.Message, fallback)

	return c
}

// Availability is what the provider says about whether clients can connect.
// The zero value means it did not say.
type Availability struct {
	Status  metav1.ConditionStatus
	Reason  string
	Message string
}

// Observation is everything the summary is computed from.
type Observation struct {
	// Now is the time of the observation.
	Now time.Time
	// Generation is the Instance's metadata.generation.
	Generation int64
	// Deleting is true once the Instance's deletion was requested.
	Deleting bool

	// Phase is the phase as stored. An unknown or empty value is rebuilt
	// from what is observed.
	Phase v1alpha1.InstancePhase
	// Conditions holds the checks and operations as observed now, and the
	// summary conditions as last written. The summary conditions are read
	// for the time they last changed, which is what the deadlines count
	// from.
	Conditions []metav1.Condition

	// GenerationStart is when this generation was first reconciled. Both
	// deadlines count from it at the earliest, so a fix committed to a
	// stalled Instance reads as in progress.
	GenerationStart time.Time
	// HadConverged is true when this generation has already been Ready. An
	// operation that starts afterwards was not asked for by the spec.
	HadConverged bool

	// Reported is true when the provider was asked for its status and
	// answered. Converged, Message and Available are what it said. When it
	// was not asked, Converged is what it last said and Available is left
	// as it was.
	Reported bool
	// Converged is true when the engine runs what the Instance asks for.
	Converged bool
	// Message says what is being waited for, if anything.
	Message string
	// Available says whether clients can connect.
	Available Availability

	// Rejected is set when the spec was rejected. Only a person can fix
	// that.
	Rejected *Cause
	// TerminalProblem is a problem the provider says only a person can
	// resolve.
	TerminalProblem *Cause
	// Waiting names what the Instance waits for when it is more specific
	// than the generic status word, such as the backups a deletion waits
	// for.
	Waiting *Cause
	// RestoreInProgress is true while a Restore of this Instance is running.
	// Someone asked for it, so the operation that carries it out keeps the
	// Instance from being Ready.
	RestoreInProgress bool

	// ProblemDeadline and ProgressDeadline override the defaults when
	// positive.
	ProblemDeadline  time.Duration
	ProgressDeadline time.Duration
}

// Result is the phase and the summary conditions to write.
type Result struct {
	Phase v1alpha1.InstancePhase
	// Conditions holds Available, Ready, Reconciling, Stalled and
	// NeedsAttention. Each keeps its last transition time unless its status
	// changed.
	Conditions []metav1.Condition
	// Recheck is how long until a deadline passes and the result changes
	// with nothing else changing. Zero means no deadline is pending.
	Recheck time.Duration
}

// Ready reports whether the Instance has converged.
func (r Result) Ready() bool {
	return meta.IsStatusConditionTrue(r.Conditions, v1alpha1.ConditionReady)
}

type outcome int

const (
	outcomeReady outcome = iota
	outcomeReconciling
	outcomeStalled
)

// verdict is the answer to "wait, or act?" and the cause behind it.
type verdict struct {
	outcome outcome
	// cause is Ready's reason and message.
	cause Cause
	// stall is Stalled's reason and message when the outcome is stalled.
	stall Cause
	// problem is true while a gating check is False.
	problem bool
	recheck time.Duration
}

// facts is what the conditions and the provider's report boil down to.
type facts struct {
	terminal  *Cause
	problem   *Cause
	unknown   *Cause
	operation *Cause
	generic   Cause
	converged bool
}

// Compute returns the phase and the summary conditions for what was
// observed.
//
// While Ready is not True exactly one of Reconciling and Stalled is True, and
// when Ready is True the other three are False. Tools that read Reconciling
// and Stalled take the first of the two they find, so they must never both
// be True.
func Compute(o Observation) Result {
	if o.ProblemDeadline <= 0 {
		o.ProblemDeadline = DefaultProblemDeadline
	}
	if o.ProgressDeadline <= 0 {
		o.ProgressDeadline = DefaultProgressDeadline
	}

	available := o.availability()
	phase := o.phase(available.Status)
	v := o.verdict(phase)

	quiet := Cause{Reason: v1alpha1.ReasonNoProblems}
	reconciling, stalled, attention := quiet, quiet, quiet
	readyStatus := metav1.ConditionFalse
	switch v.outcome {
	case outcomeReady:
		readyStatus = metav1.ConditionTrue
	case outcomeReconciling:
		reconciling = v.cause
		if v.problem {
			attention = v.cause
		}
	case outcomeStalled:
		reconciling = Cause{Reason: v.stall.Reason}
		stalled, attention = v.stall, v.cause
	}

	return Result{
		Phase: phase,
		Conditions: []metav1.Condition{
			o.condition(v1alpha1.ConditionAvailable, available.Status, Cause{Reason: available.Reason, Message: available.Message}),
			o.condition(v1alpha1.ConditionReady, readyStatus, v.cause),
			o.condition(v1alpha1.ConditionReconciling, conditionStatus(v.outcome == outcomeReconciling), reconciling),
			o.condition(v1alpha1.ConditionStalled, conditionStatus(v.outcome == outcomeStalled), stalled),
			o.condition(v1alpha1.ConditionNeedsAttention, conditionStatus(v.outcome == outcomeStalled || v.problem), attention),
		},
		Recheck: v.recheck,
	}
}

// verdict decides between ready, wait and act. A cause only a person can
// resolve stalls at once. A problem outranks any operation, and stalls once
// it has outlasted its deadline. An operation keeps the Instance from being
// Ready by itself only when someone asked for it. A check that is Unknown
// keeps it from being Ready and leaves the status word alone.
func (o *Observation) verdict(phase v1alpha1.InstancePhase) verdict {
	f := o.facts(phase)
	switch {
	case f.terminal != nil:
		return verdict{outcome: outcomeStalled, cause: *f.terminal, stall: *f.terminal}
	case f.problem != nil:
		due := latest(o.problemSince(), o.GenerationStart).Add(o.ProblemDeadline)
		if !o.Now.Before(due) {
			return verdict{outcome: outcomeStalled, cause: *f.problem, stall: *f.problem, problem: true}
		}

		return o.progress(&f, verdict{outcome: outcomeReconciling, cause: *f.problem, problem: true, recheck: due.Sub(o.Now)})
	case f.operation != nil && (!o.HadConverged || o.RestoreInProgress):
		return verdict{outcome: outcomeReconciling, cause: f.operation.orMessage(f.generic.Message)}
	case f.unknown != nil || !f.converged:
		// A check that is Unknown is ordinary progress: pods come and go
		// while a change rolls. It must not supply the status word, or the
		// word would alternate with the generic one during a single
		// rollout.
		cause := firstCause(f.operation, &f.generic).orMessage(f.generic.Message)

		return o.progress(&f, verdict{outcome: outcomeReconciling, cause: cause})
	case f.operation != nil:
		// The engine or an autoscaler started it on a spec that had already
		// converged. The Instance stays Ready, so tools that watch it do not
		// flap, and the operation supplies the status word.
		return verdict{outcome: outcomeReady, cause: f.operation.orMessage(messageReady)}
	default:
		return verdict{outcome: outcomeReady, cause: Cause{Reason: v1alpha1.ReasonReady, Message: messageReady}}
	}
}

// progress applies the progress deadline to a verdict that says to wait. A
// named operation is exempt: a restore may take hours.
//
// The deadline counts from the latest of four times, none of which is when
// Reconciling last changed: stalling turns Reconciling False, so a stall
// measured from that time would undo itself on the next pass. Ready's time
// does not move during a stall.
func (o *Observation) progress(f *facts, v verdict) verdict {
	if f.operation != nil {
		return v
	}
	due := latest(o.readyLost(), o.stallEnded(), o.operationEnded(), o.GenerationStart).Add(o.ProgressDeadline)
	if o.Now.Before(due) {
		v.recheck = soonest(v.recheck, due.Sub(o.Now))

		return v
	}

	message := fmt.Sprintf("No progress for %s", o.ProgressDeadline)
	if v.cause.Message != "" {
		message += ": " + v.cause.Message
	}
	v.outcome = outcomeStalled
	v.stall = Cause{Reason: v1alpha1.ReasonProgressDeadlineExceeded, Message: message}
	v.recheck = 0
	// The status word stays the problem's when there is one.
	if !v.problem {
		v.cause = v.stall
	}

	return v
}

func (o *Observation) facts(phase v1alpha1.InstancePhase) facts {
	if o.Deleting {
		// The checks describe an engine that is being torn down. Only a
		// cleanup that keeps failing is a problem now.
		return facts{
			problem: o.check([]string{v1alpha1.ConditionSynced}, metav1.ConditionFalse),
			generic: o.generic(phase, nil),
		}
	}
	unknown := o.check(GatingChecks(), metav1.ConditionUnknown)

	return facts{
		terminal:  firstCause(o.Rejected, o.retriesExhausted(), o.TerminalProblem),
		problem:   o.check(GatingChecks(), metav1.ConditionFalse),
		unknown:   unknown,
		operation: o.runningOperation(),
		generic:   o.generic(phase, unknown),
		converged: o.Converged,
	}
}

// generic is the cause when nothing more specific is known. Its reason says
// where the Instance is in its life, so while the Instance converges it
// changes at most once, when the database is first available. Its message is
// what the provider says it waits for, or else what a check that is still in
// progress says, or else a stock sentence.
func (o *Observation) generic(phase v1alpha1.InstancePhase, unknown *Cause) Cause {
	if o.Waiting != nil {
		return *o.Waiting
	}
	var c Cause
	switch phase {
	case v1alpha1.InstancePhaseTerminating:
		c = Cause{Reason: v1alpha1.ReasonDeleting, Message: messageDeleting}
	case v1alpha1.InstancePhaseProvisioned:
		c = Cause{Reason: v1alpha1.ReasonApplyingChanges, Message: messageApplying}
	case v1alpha1.InstancePhasePending:
		c = Cause{Reason: v1alpha1.ReasonCreating, Message: messageCreating}
	}
	inProgress := ""
	if unknown != nil {
		inProgress = unknown.Message
	}
	c.Message = cmp.Or(o.Message, inProgress, c.Message)

	return c
}

// check returns the reason and message of the first of the checks that has
// the status.
func (o *Observation) check(checks []string, status metav1.ConditionStatus) *Cause {
	for _, t := range checks {
		if c := o.find(t); c != nil && c.Status == status {
			return &Cause{Reason: c.Reason, Message: c.Message}
		}
	}

	return nil
}

// runningOperation returns the operation that started last among those in
// progress. Its condition type is the status word.
func (o *Observation) runningOperation() *Cause {
	var running *metav1.Condition
	for _, t := range Operations() {
		c := o.find(t)
		if c == nil || c.Status != metav1.ConditionTrue {
			continue
		}
		if running == nil || c.LastTransitionTime.After(running.LastTransitionTime.Time) {
			running = c
		}
	}
	if running == nil {
		return nil
	}

	return &Cause{Reason: running.Type, Message: running.Message}
}

// retriesExhausted returns the cause when an approved maintenance action kept
// failing and is no longer retried: only a new approval re-arms it.
func (o *Observation) retriesExhausted() *Cause {
	c := o.find(v1alpha1.ConditionMaintenancePending)
	if c == nil || c.Status != metav1.ConditionTrue || c.Reason != v1alpha1.ReasonRetriesExhausted {
		return nil
	}

	return &Cause{Reason: c.Reason, Message: c.Message}
}

// problemSince is when the problem began: NeedsAttention stays True for as
// long as one is observed, also through a stall.
func (o *Observation) problemSince() time.Time {
	if c := o.find(v1alpha1.ConditionNeedsAttention); c != nil && c.Status == metav1.ConditionTrue {
		return c.LastTransitionTime.Time
	}

	return o.Now
}

// readyLost is when Ready last stopped being True.
func (o *Observation) readyLost() time.Time {
	if c := o.find(v1alpha1.ConditionReady); c != nil && c.Status != metav1.ConditionTrue {
		return c.LastTransitionTime.Time
	}

	return o.Now
}

// stallEnded is when the Instance last stopped being stalled, so that an
// Instance recovering from a problem that had stalled it gets the whole
// progress deadline again. A stall set by the progress deadline itself has
// not ended: it holds until one of the other times moves.
func (o *Observation) stallEnded() time.Time {
	c := o.find(v1alpha1.ConditionStalled)
	switch {
	case c == nil:
		return time.Time{}
	case c.Status != metav1.ConditionTrue:
		return c.LastTransitionTime.Time
	case c.Reason == v1alpha1.ReasonProgressDeadlineExceeded:
		return time.Time{}
	default:
		return o.Now
	}
}

// operationEnded is when the most recent named operation finished. What
// follows a long restore gets the whole progress deadline.
func (o *Observation) operationEnded() time.Time {
	var ended time.Time
	for _, t := range Operations() {
		if c := o.find(t); c != nil && c.Status == metav1.ConditionFalse {
			ended = latest(ended, c.LastTransitionTime.Time)
		}
	}

	return ended
}

// availability returns the Available condition. Deletion and the loading of
// seed data override what the provider says: anything written while a new
// Instance is seeded would be overwritten. A provider that does not say is
// assumed available once it has converged, unavailable until the database
// has first been available, and unknown otherwise, so it never claims an
// outage it cannot see.
func (o *Observation) availability() Availability {
	seed := o.find(v1alpha1.ConditionDataSourceReady)
	prior := o.find(v1alpha1.ConditionAvailable)
	switch {
	case o.Deleting:
		return Availability{Status: metav1.ConditionUnknown, Reason: v1alpha1.ReasonDeleting, Message: messageDeleting}
	case seed != nil && seed.Status != metav1.ConditionTrue:
		a := Availability{Status: metav1.ConditionFalse, Reason: v1alpha1.ReasonLoadingData, Message: seed.Message}
		if a.Message == "" {
			a.Message = messageLoadingData
		}

		return a
	case o.Available.Status != "":
		return o.reportedAvailability()
	case !o.Reported && prior != nil && prior.Reason != v1alpha1.ReasonLoadingData && prior.Reason != v1alpha1.ReasonDeleting:
		// Nobody looked in this pass, so the last observation stands.
		return Availability{Status: prior.Status, Reason: prior.Reason, Message: prior.Message}
	case o.Converged:
		return Availability{Status: metav1.ConditionTrue, Reason: v1alpha1.ReasonAvailable}
	case o.Phase != v1alpha1.InstancePhaseProvisioned:
		return Availability{Status: metav1.ConditionFalse, Reason: v1alpha1.ReasonNotCreated, Message: messageNotCreated}
	default:
		return Availability{Status: metav1.ConditionUnknown, Reason: v1alpha1.ReasonNotReported, Message: messageNotReported}
	}
}

func (o *Observation) reportedAvailability() Availability {
	a := o.Available
	if a.Reason != "" {
		return a
	}
	switch a.Status {
	case metav1.ConditionTrue:
		a.Reason = v1alpha1.ReasonAvailable
	case metav1.ConditionFalse:
		a.Reason = v1alpha1.ReasonNotAvailable
	case metav1.ConditionUnknown:
		a.Reason = v1alpha1.ReasonNotReported
	}

	return a
}

// phase moves forward only: Pending until the database is first available,
// then Provisioned, and Terminating once deletion is requested. It is rebuilt
// the same way when the stored value is lost.
func (o *Observation) phase(available metav1.ConditionStatus) v1alpha1.InstancePhase {
	switch {
	case o.Deleting:
		return v1alpha1.InstancePhaseTerminating
	case o.Phase == v1alpha1.InstancePhaseProvisioned || available == metav1.ConditionTrue:
		return v1alpha1.InstancePhaseProvisioned
	default:
		return v1alpha1.InstancePhasePending
	}
}

// condition builds a summary condition. Its last transition time moves only
// when the status changes; the deadlines depend on that.
func (o *Observation) condition(condType string, status metav1.ConditionStatus, cause Cause) metav1.Condition {
	c := metav1.Condition{
		Type:               condType,
		Status:             status,
		ObservedGeneration: o.Generation,
		LastTransitionTime: metav1.NewTime(o.Now),
		Reason:             cause.Reason,
		Message:            cause.Message,
	}
	if prior := o.find(condType); prior != nil && prior.Status == status {
		c.LastTransitionTime = prior.LastTransitionTime
	}

	return c
}

func (o *Observation) find(condType string) *metav1.Condition {
	return meta.FindStatusCondition(o.Conditions, condType)
}

func conditionStatus(b bool) metav1.ConditionStatus {
	if b {
		return metav1.ConditionTrue
	}

	return metav1.ConditionFalse
}

func firstCause(causes ...*Cause) *Cause {
	for _, c := range causes {
		if c != nil {
			return c
		}
	}

	return nil
}

func latest(times ...time.Time) time.Time {
	var out time.Time
	for _, t := range times {
		if t.After(out) {
			out = t
		}
	}

	return out
}

// soonest returns the shorter of two waits, where zero means no wait.
func soonest(a, b time.Duration) time.Duration {
	if a <= 0 || (b > 0 && b < a) {
		return b
	}

	return a
}
