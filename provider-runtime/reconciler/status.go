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

package reconciler

import (
	"cmp"
	"context"
	"errors"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
	"github.com/openeverest/openeverest/v2/provider-runtime/summary"
)

// syncGrace is how long the provider's calls may keep failing before that is
// reported as a problem. A conflict, or a webhook that is briefly
// unreachable, fails a call or two and then clears.
const syncGrace = time.Minute

// conditionReason is the format the API server accepts for the reason of a
// condition.
var conditionReason = regexp.MustCompile(`^[A-Za-z]([A-Za-z0-9_,:]*[A-Za-z0-9_])?$`)

// reasonOr returns reason when a condition can carry it, and fallback
// otherwise. A status with a malformed reason is rejected as a whole, so a
// reason a provider made up must not reach it unchecked.
func reasonOr(reason, fallback string) string {
	if conditionReason.MatchString(reason) {
		return reason
	}

	return fallback
}

// tracker remembers, for each Instance, what its status cannot hold: when
// the current generation was first reconciled, whether it has converged
// since, what the provider last reported and since when its calls have been
// failing. The deadlines that turn a wait into a stall count from the first
// of these, and they start again on a new generation.
//
// It lives in memory. A restart forgets it, so the deadlines and the failure
// streak start again, which only delays a stall. It also assumes that a
// single replica of the provider reconciles the Instances: two would each
// keep their own clocks and disagree. Run one replica, or add leader
// election.
type tracker struct {
	mu        sync.Mutex
	instances map[types.NamespacedName]*tracked
}

// tracked is what the tracker remembers about one generation of an Instance.
type tracked struct {
	uid        types.UID
	generation int64
	// start is when this generation was first reconciled.
	start time.Time
	// converged is true once this generation has been Ready. A provider
	// call that fails afterwards changes nothing until it has kept failing
	// for syncGrace, and an operation that starts afterwards was not asked
	// for by the spec.
	converged bool
	// lastConverged, message and terminal are what the provider last said:
	// whether the engine runs what the Instance asks for, what it waits for,
	// and the problem only a person can resolve. They stand in on a pass
	// that does not get to ask.
	lastConverged bool
	message       string
	terminal      *summary.Cause
	// failingSince is when the provider's calls started to fail, and zero
	// when the last one succeeded.
	failingSince time.Time
}

// track returns what is remembered about the Instance's current generation,
// starting afresh when the generation is new to this process. A generation
// stored as Ready had converged before the process started.
func (t *tracker) track(in *v1alpha1.Instance, now time.Time) *tracked {
	t.mu.Lock()
	defer t.mu.Unlock()

	key := client.ObjectKeyFromObject(in)
	if st, ok := t.instances[key]; ok && st.uid == in.UID && st.generation == in.Generation {
		return st
	}
	ready := meta.FindStatusCondition(in.Status.Conditions, v1alpha1.ConditionReady)
	converged := ready != nil && ready.Status == metav1.ConditionTrue && ready.ObservedGeneration == in.Generation
	st := &tracked{uid: in.UID, generation: in.Generation, start: now, converged: converged, lastConverged: converged}
	if t.instances == nil {
		t.instances = make(map[types.NamespacedName]*tracked)
	}
	t.instances[key] = st

	return st
}

func (t *tracker) forget(key types.NamespacedName) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.instances, key)
}

// pass is one reconcile of one Instance: its status as stored when the pass
// began, and what has been found out since. Every way out of a reconcile
// goes through finish, which turns the pass into the status to write.
type pass struct {
	in     *v1alpha1.Instance
	stored v1alpha1.InstanceStatus
	now    time.Time
	state  *tracked

	deleting bool
	// touched holds the conditions set in this pass.
	touched map[string]bool
	// rejected is set when the spec was rejected.
	rejected *summary.Cause
	// waiting names what a deletion waits for.
	waiting *summary.Cause
	// message is what the provider says it waits for.
	message string
	// report is the provider's status, when it was asked in this pass.
	report *controller.Status
	// providerRan is true when the provider was called and did not fail.
	providerRan bool
	// backupUsable is true when the runtime's own checks of the backup
	// configuration passed.
	backupUsable bool
	podsObserved bool
	// version is the version bundle the provider was handed.
	version string
	// restoring is true while a Restore of the Instance is in progress.
	restoring bool
	// recheck is how soon the Instance must be looked at again because
	// something changes with time alone. Zero means it need not be.
	recheck time.Duration
}

func (r *ProviderReconciler) now() time.Time {
	if r.clock != nil {
		return r.clock()
	}

	return time.Now()
}

func (r *ProviderReconciler) begin(in *v1alpha1.Instance) *pass {
	now := r.now()

	return &pass{
		in:      in,
		stored:  *in.Status.DeepCopy(),
		now:     now,
		state:   r.tracker.track(in, now),
		touched: make(map[string]bool),
	}
}

func (p *pass) find(condType string) *metav1.Condition {
	return meta.FindStatusCondition(p.in.Status.Conditions, condType)
}

// check records a condition observed in this pass.
func (p *pass) check(condType string, status metav1.ConditionStatus, reason, message string) {
	setCondition(p.in, condType, status, reason, message, metav1.NewTime(p.now))
	p.touched[condType] = true
}

func (p *pass) lookAgainIn(d time.Duration) {
	p.recheck = soonest(p.recheck, d)
}

// reject records that the spec was rejected.
func (p *pass) reject(reason, message string) {
	p.rejected = &summary.Cause{Reason: reason, Message: message}
}

// wait records that the provider asked to be called again later.
func (p *pass) wait(err error) {
	if wait, ok := errors.AsType[*controller.WaitError](err); ok {
		p.message = wait.Reason
	}
	p.state.lastConverged = false
	p.state.message = p.message
	p.providerRan = true
	p.lookAgainIn(controller.GetWaitDuration(err))
}

// referentsExist records that the Provider and the user Secret exist.
// Whether their contents fit is checked later in the pass, so a verdict
// about contents stands until then.
func (p *pass) referentsExist() {
	c := p.find(v1alpha1.ConditionReferencesResolved)
	if c != nil && c.Status == metav1.ConditionFalse && c.Reason == v1alpha1.ReasonInvalidReference {
		return
	}
	p.check(v1alpha1.ConditionReferencesResolved, metav1.ConditionTrue, v1alpha1.ReasonResolved, "")
}

// flushDataSource records the seeding of a new Instance on DataSourceReady:
// True once seeded, Unknown while seeding is under way, and False when it
// failed or cannot go on.
func (p *pass) flushDataSource(syncCtx *controller.Context) {
	ds := syncCtx.GetDataSourceStatus()
	if ds == nil || ds.State == controller.DataSourceStateNone {
		return
	}
	status := metav1.ConditionUnknown
	switch {
	case ds.State == controller.DataSourceStateSucceeded:
		status = metav1.ConditionTrue
	case ds.State == controller.DataSourceStateFailed, ds.Blocked:
		status = metav1.ConditionFalse
	}
	p.check(v1alpha1.ConditionDataSourceReady, status, reasonOr(ds.Reason, string(ds.State)), ds.Message)
}

// reported records what the provider observes about the engine.
func (p *pass) reported(ctx context.Context, status controller.Status) {
	p.report = &status
	p.message = status.Message
	p.providerRan = true

	st := p.state
	st.lastConverged = status.Converged
	st.message = status.Message
	st.terminal = nil

	for _, op := range status.Operations {
		if !slices.Contains(summary.Operations(), op.Type) {
			log.FromContext(ctx).Info("Ignoring an operation that is not one of the known operation conditions", "type", op.Type)
		}
	}

	if len(status.Problems) == 0 {
		p.check(v1alpha1.ConditionEngineHealthy, metav1.ConditionTrue, v1alpha1.ReasonHealthy, "")

		return
	}
	// The problem only a person can resolve leads: it decides the status word.
	problems := slices.Clone(status.Problems)
	slices.SortStableFunc(problems, func(a, b controller.Problem) int {
		return cmp.Compare(boolRank(b.Terminal), boolRank(a.Terminal))
	})
	lead := problems[0]
	reason := reasonOr(lead.Reason, v1alpha1.ReasonEngineError)
	messages := make([]string, 0, len(problems))
	for _, problem := range problems {
		if problem.Message != "" {
			messages = append(messages, problem.Message)
		}
	}
	message := strings.Join(messages, "\n")
	p.check(v1alpha1.ConditionEngineHealthy, metav1.ConditionFalse, reason, message)
	if lead.Terminal {
		st.terminal = &summary.Cause{Reason: reason, Message: message}
	}
}

func boolRank(b bool) int {
	if b {
		return 1
	}

	return 0
}

// setOperations turns the long operations in progress into their conditions:
// True while one runs, and False with reason Succeeded once it no longer
// does, so that a client can still tell when it finished. On a pass that did
// not ask the provider, what it last reported stands.
func (p *pass) setOperations() {
	running := p.runningOperations()
	seed := p.find(v1alpha1.ConditionDataSourceReady)
	for _, condType := range summary.Operations() {
		if op, ok := running[condType]; ok {
			p.check(condType, metav1.ConditionTrue, reasonOr(op.Reason, condType), op.Message)

			continue
		}
		current := p.find(condType)
		if current == nil || current.Status != metav1.ConditionTrue {
			continue
		}
		// The seeding operation is the runtime's own, so it is known to have
		// ended even on a pass that did not ask the provider.
		own := condType == v1alpha1.ConditionRestoring && current.Reason == v1alpha1.ReasonSeeding
		if p.report == nil && !own {
			continue
		}
		reason := v1alpha1.ReasonSucceeded
		if own && seed != nil && seed.Status == metav1.ConditionFalse {
			reason = v1alpha1.ReasonDataSourceFailed
		}
		p.check(condType, metav1.ConditionFalse, reason, "")
	}
}

// runningOperations returns the operations in progress by condition type:
// those the provider reported in this pass, and the seeding of a new
// Instance, which is a restore whether or not the provider says so.
func (p *pass) runningOperations() map[string]controller.Operation {
	running := make(map[string]controller.Operation)
	if p.report != nil {
		for _, op := range p.report.Operations {
			if _, ok := running[op.Type]; !ok {
				running[op.Type] = op
			}
		}
	}

	seed := p.find(v1alpha1.ConditionDataSourceReady)
	if seed == nil || seed.Status != metav1.ConditionUnknown {
		return running
	}
	if _, ok := running[v1alpha1.ConditionRestoring]; ok {
		return running
	}
	// A restore the provider reported earlier stands on a pass that did not
	// ask it.
	current := p.find(v1alpha1.ConditionRestoring)
	byProvider := current != nil && current.Status == metav1.ConditionTrue && current.Reason != v1alpha1.ReasonSeeding
	if p.report == nil && byProvider {
		return running
	}
	running[v1alpha1.ConditionRestoring] = controller.Operation{
		Type:    v1alpha1.ConditionRestoring,
		Reason:  v1alpha1.ReasonSeeding,
		Message: seed.Message,
	}

	return running
}

// recordOutcome maintains the Synced check from how the pass ended. It turns
// False once the provider's calls have failed for syncGrace without a
// success in between, and True on the first success. Before that a failure
// on a generation that had already converged changes nothing: one failed
// call must not make a healthy database read as not ready. On a generation
// that is still converging the check turns Unknown, in progress, which
// leaves the status word as it is and shows the error as the sentence when
// the provider gave none.
func (p *pass) recordOutcome(err error) {
	st := p.state
	if err == nil {
		if p.providerRan {
			st.failingSince = time.Time{}
			p.check(v1alpha1.ConditionSynced, metav1.ConditionTrue, v1alpha1.ReasonSynced, "")
		}

		return
	}

	if st.failingSince.IsZero() {
		st.failingSince = p.now
	}
	failing := p.now.Sub(st.failingSince)
	if failing >= syncGrace {
		p.check(v1alpha1.ConditionSynced, metav1.ConditionFalse, v1alpha1.ReasonReconcileError, err.Error())

		return
	}
	p.lookAgainIn(syncGrace - failing)
	if !st.converged {
		p.check(v1alpha1.ConditionSynced, metav1.ConditionUnknown, v1alpha1.ReasonReconcileError, err.Error())
	}
}

// withdrawBackupVerdict removes a BackupConfigured verdict the runtime's own
// checks issued once they pass and nothing else set the condition in this
// pass: the cause is gone, and whether the engine accepts the configuration
// is not known until Sync has applied it.
func (p *pass) withdrawBackupVerdict() {
	if !p.backupUsable || p.touched[v1alpha1.ConditionBackupConfigured] {
		return
	}
	c := p.find(v1alpha1.ConditionBackupConfigured)
	if c == nil || c.Status != metav1.ConditionFalse {
		return
	}
	issued := []string{
		v1alpha1.ReasonBackupClassNotFound,
		v1alpha1.ReasonBackupClassIncompatible,
		v1alpha1.ReasonBackupStorageNotFound,
		v1alpha1.ReasonBackupStorageCredentialsNotFound,
		controller.LimitsExceededReason,
		controller.PITRConfigInvalidReason,
	}
	if slices.Contains(issued, c.Reason) {
		meta.RemoveStatusCondition(&p.in.Status.Conditions, v1alpha1.ConditionBackupConfigured)
	}
}

// finish ends a pass: it completes what the pass observed, writes the status
// if it changed, and makes sure the Instance is reconciled again no later
// than the next time its status changes with time alone, such as a deadline
// that passes or a crash loop that has been quiet for long enough.
//
// A reconcile that returns an error is retried with a backoff that can grow
// well past such a time, so the Instance is also queued for that time
// directly.
func (r *ProviderReconciler) finish(ctx context.Context, p *pass, err error) (reconcile.Result, error) {
	p.recordOutcome(err)
	if !p.deleting {
		r.observeStoredPods(ctx, p)
		p.setOperations()
		p.withdrawBackupVerdict()
		p.restoring = r.restoreInProgress(ctx, p.in)
	}

	if writeErr := r.writeStatus(ctx, p); writeErr != nil {
		log.FromContext(ctx).Error(writeErr, "Failed to update status")
		if err == nil {
			err = writeErr
		}
	}
	if err != nil {
		r.requeueAfter(client.ObjectKeyFromObject(p.in), p.recheck)

		return reconcile.Result{}, err
	}

	word := ""
	if ready := p.find(v1alpha1.ConditionReady); ready != nil {
		word = ready.Reason
	}
	log.FromContext(ctx).Info("Reconciliation complete", "phase", p.in.Status.Phase, "status", word, "requeueAfter", p.recheck)

	return reconcile.Result{RequeueAfter: p.recheck}, nil
}

// requeueAfter queues the Instance to be reconciled again after the delay,
// whatever the reconcile that asks for it returns. The queue keeps the
// earlier of this and any retry already scheduled.
func (r *ProviderReconciler) requeueAfter(key client.ObjectKey, after time.Duration) {
	if again := r.requeue.Load(); again != nil && after > 0 {
		(*again)(reconcile.Request{NamespacedName: key}, after)
	}
}

// observeStoredPods looks at the pods of the components the status already
// lists on a pass that did not get to ask the provider which components
// there are. A crash loop starts and ends whether or not the provider can be
// called.
func (r *ProviderReconciler) observeStoredPods(ctx context.Context, p *pass) {
	if p.podsObserved || len(p.stored.Components) == 0 {
		return
	}
	components := make([]string, 0, len(p.stored.Components))
	for _, c := range p.stored.Components {
		components = append(components, c.Name)
	}
	p.lookAgainIn(r.setPodStatus(ctx, p.in, components, p.now))
	p.podsObserved = true
}

// restoreInProgress reports whether a Restore of the Instance is running.
// Someone asked for it, so the operation that carries it out keeps the
// Instance from being Ready even though its spec did not change.
func (r *ProviderReconciler) restoreInProgress(ctx context.Context, in *v1alpha1.Instance) bool {
	if !r.isBackupProvider() {
		return false
	}
	restores := &backupv1alpha1.RestoreList{}
	if err := r.List(ctx, restores,
		client.InNamespace(in.Namespace),
		client.MatchingFields{controller.IndexRestoreInstanceName: in.Name},
	); err != nil {
		log.FromContext(ctx).Error(err, "Listing the restores of the instance failed")

		return false
	}
	for i := range restores.Items {
		restore := &restores.Items[i]
		if !restore.GetDeletionTimestamp().IsZero() {
			continue
		}
		if state := restore.Status.State; state != backupv1alpha1.RestoreStateSucceeded && state != backupv1alpha1.RestoreStateFailed {
			return true
		}
	}

	return false
}

// summarize derives the phase and the summary conditions from what the pass
// observed.
func (r *ProviderReconciler) summarize(p *pass) {
	in, st := p.in, p.state
	o := summary.Observation{
		Now:               p.now,
		Generation:        in.Generation,
		Deleting:          p.deleting,
		Phase:             in.Status.Phase,
		Conditions:        in.Status.Conditions,
		GenerationStart:   st.start,
		HadConverged:      st.converged,
		Converged:         st.lastConverged,
		Message:           cmp.Or(p.message, st.message),
		Rejected:          p.rejected,
		TerminalProblem:   st.terminal,
		Waiting:           p.waiting,
		RestoreInProgress: p.restoring,
		ProblemDeadline:   r.problemDeadline,
		ProgressDeadline:  r.progressDeadline,
	}
	if p.report != nil {
		o.Reported = true
		o.Available = summary.Availability{
			Status:  metav1.ConditionStatus(p.report.Available),
			Reason:  reasonOr(p.report.AvailableReason, ""),
			Message: p.report.AvailableMessage,
		}
	}

	res := summary.Compute(o)
	in.Status.Phase = res.Phase
	for _, c := range res.Conditions {
		if i := slices.IndexFunc(in.Status.Conditions, func(have metav1.Condition) bool { return have.Type == c.Type }); i >= 0 {
			in.Status.Conditions[i] = c
		} else {
			in.Status.Conditions = append(in.Status.Conditions, c)
		}
	}
	st.converged = st.converged || res.Ready()
	p.lookAgainIn(res.Recheck)
}

// writeStatus derives the summary and writes the status if it differs from
// what is stored. provider-runtime is the only writer of an Instance's
// status.
func (r *ProviderReconciler) writeStatus(ctx context.Context, p *pass) error {
	r.summarize(p)
	in := p.in
	in.Status.ObservedGeneration = in.Generation
	settleConditions(p.stored.Conditions, in.Status.Conditions)
	if equality.Semantic.DeepEqual(&p.stored, &in.Status) {
		return nil
	}
	if err := r.Client.Status().Update(ctx, in); err != nil {
		return err
	}
	p.stored = *in.Status.DeepCopy()

	return nil
}

// settleConditions makes the conditions comparable with the stored ones. A
// condition whose status is what was stored keeps the stored transition
// time, however often it was set during the pass, and the conditions are put
// in a fixed order: the summary first, then checks, operations and notices.
func settleConditions(stored, conditions []metav1.Condition) {
	for i := range conditions {
		if was := meta.FindStatusCondition(stored, conditions[i].Type); was != nil && was.Status == conditions[i].Status {
			conditions[i].LastTransitionTime = was.LastTransitionTime
		}
	}

	order := slices.Concat(
		[]string{
			v1alpha1.ConditionAvailable,
			v1alpha1.ConditionReady,
			v1alpha1.ConditionReconciling,
			v1alpha1.ConditionStalled,
			v1alpha1.ConditionNeedsAttention,
		},
		summary.GatingChecks(),
		[]string{
			v1alpha1.ConditionConnectionDetailsReady,
			v1alpha1.ConditionMonitoringConfigured,
			v1alpha1.ConditionMonitoringReady,
		},
		summary.Operations(),
	)
	rank := func(c metav1.Condition) int {
		if i := slices.Index(order, c.Type); i >= 0 {
			return i
		}

		return len(order)
	}
	slices.SortStableFunc(conditions, func(a, b metav1.Condition) int {
		return cmp.Compare(rank(a), rank(b))
	})
}
