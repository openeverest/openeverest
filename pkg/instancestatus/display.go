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

// Package instancestatus turns an Instance's status into what a person sees:
// a status word with a tone, whether the database is available, and how many
// of each component's pods are ready.
//
// Everything written in Go that shows an Instance, or waits for one, reads
// its status through this package, so that they all agree. The UI has the
// same function in TypeScript, and both are tested against the cases in
// testdata/display.json.
package instancestatus

import (
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

// Tone says how to style the status word. It answers "wait or act?" and
// nothing else: whether the database is up is Availability.
type Tone string

const (
	// ToneOK means the Instance has converged.
	ToneOK Tone = "ok"
	// ToneProgress means something is happening that finishes by itself: wait.
	ToneProgress Tone = "progress"
	// ToneAttention means a problem is observed that may still clear.
	ToneAttention Tone = "attention"
	// ToneError means the Instance will not converge without a person: act.
	ToneError Tone = "error"
	// ToneStopped means the Instance is stopped because its spec asks for it.
	ToneStopped Tone = "stopped"
	// ToneDeleting means deletion was requested.
	ToneDeleting Tone = "deleting"
	// ToneUnknown means the provider has not reported on the current spec
	// for so long that nobody can tell what state the Instance is in.
	ToneUnknown Tone = "unknown"
)

// Availability says whether clients can connect to the database.
type Availability string

const (
	// AvailableYes means the endpoint serves requests.
	AvailableYes Availability = "Yes"
	// AvailableNo means it does not.
	AvailableNo Availability = "No"
	// AvailableUnknown means nobody can tell, or nobody has looked yet.
	AvailableUnknown Availability = "Unknown"
)

// conditionSuspended is True while compute is at zero because the spec asks
// for it.
const conditionSuspended = "Suspended"

// sentenceSeparator joins the parts of a details sentence.
const sentenceSeparator = " · "

// Input is the part of an Instance its status is read from. The Kubernetes
// type and the API client's type both carry these fields, so either is
// adapted without copying the object.
type Input struct {
	// Meta is the Instance's metadata. The generation and the deletion
	// timestamp are read. Nil reads as empty metadata.
	Meta *metav1.ObjectMeta
	// ObservedGeneration is status.observedGeneration.
	ObservedGeneration int64
	// Phase is status.phase.
	Phase string
	// Conditions is status.conditions.
	Conditions []metav1.Condition
}

// FromInstance adapts the Kubernetes type.
func FromInstance(inst *corev1alpha1.Instance) Input {
	return Input{
		Meta:               &inst.ObjectMeta,
		ObservedGeneration: inst.Status.ObservedGeneration,
		Phase:              string(inst.Status.Phase),
		Conditions:         inst.Status.Conditions,
	}
}

// Display is what a person sees for an Instance.
type Display struct {
	// Tone styles the status word.
	Tone Tone
	// Reason is the status word as the API spells it: "Upgrading",
	// "CrashLoopBackOff".
	Reason string
	// Label is the status word as a person reads it: "Upgrading",
	// "Crash loop".
	Label string
	// Available says whether clients can connect.
	Available Availability
	// Message is one sentence about the status word. While Stale is set it
	// is the Unattended condition's message, or empty, because the
	// provider's message describes an older spec.
	Message string
	// Since is when the Instance got into this status, or nil if that is
	// not known.
	Since *time.Time
	// Sentence joins availability and the status word for a details view:
	// "Up · Upgrading".
	Sentence string
	// Stale is set while the status describes an older spec than the current
	// one. The status word is then a generic one, and the conditions are
	// what was last observed.
	Stale bool
}

// Describe reads an Instance's status as of now.
//
// The tone comes from the statuses of the summary conditions and never from
// a reason, so a reason this code has never seen still gets the right tone,
// and its label is derived from its spelling.
//
// Nothing is inferred from how long the status has not changed: when nobody
// attends to an Instance, core says so with the Unattended condition. The
// current time is reserved for provider liveness, which will decide whether
// a status anyone still vouches for; callers pass it already.
func Describe(in Input, _ time.Time) Display {
	tone := in.tone()
	reason := in.reason(tone)
	d := Display{
		Tone:      tone,
		Reason:    reason,
		Label:     Label(reason),
		Available: in.availability(),
		Since:     in.since(tone, reason),
		Stale:     in.stale(),
	}
	if unattended := in.unattended(); unattended != nil {
		d.Message = unattended.Message
	} else if ready := in.condition(corev1alpha1.ConditionReady); ready != nil && !d.Stale {
		d.Message = ready.Message
	}
	d.Sentence = in.sentence(d)
	return d
}

// tone picks the tone. The first case that matches wins.
func (in Input) tone() Tone {
	unattended := in.unattended()
	switch {
	case in.deleting():
		return ToneDeleting
	case unattended != nil && unattended.Status == metav1.ConditionTrue:
		return ToneAttention
	case unattended != nil:
		return ToneUnknown
	case in.isTrue(conditionSuspended):
		return ToneStopped
	case in.stale():
		// The remaining conditions describe an older spec. A stall on it
		// says nothing about the current one: a fix may just have been
		// pushed.
		return ToneProgress
	case in.isTrue(corev1alpha1.ConditionStalled):
		return ToneError
	case in.isTrue(corev1alpha1.ConditionNeedsAttention):
		return ToneAttention
	case in.isTrue(corev1alpha1.ConditionReady):
		return ToneOK
	default:
		return ToneProgress
	}
}

// reason picks the status word: the reason of the Unattended condition
// while it is honoured, otherwise the reason of the Ready condition, unless
// that condition is missing or describes an older spec.
func (in Input) reason(tone Tone) string {
	ready := in.condition(corev1alpha1.ConditionReady)
	unattended := in.unattended()
	switch {
	case unattended != nil && unattended.Reason != "":
		return unattended.Reason
	case ready != nil && ready.Reason != "" && !in.stale():
		return ready.Reason
	case tone == ToneDeleting:
		return corev1alpha1.ReasonDeleting
	case in.neverUp():
		return corev1alpha1.ReasonCreating
	default:
		return corev1alpha1.ReasonApplyingChanges
	}
}

// availability reads the Available condition. It is shown even when the
// status is stale: whether clients can connect is observed, and does not
// depend on which spec was applied.
func (in Input) availability() Availability {
	available := in.condition(corev1alpha1.ConditionAvailable)
	switch {
	case available == nil:
		return AvailableUnknown
	case available.Status == metav1.ConditionTrue:
		return AvailableYes
	case available.Status == metav1.ConditionFalse:
		return AvailableNo
	default:
		return AvailableUnknown
	}
}

// since is when the Instance got into the status it shows.
func (in Input) since(tone Tone, reason string) *time.Time {
	if tone == ToneDeleting {
		return timeOf(in.meta().DeletionTimestamp)
	}
	if in.stale() && in.unattended() == nil {
		return nil
	}
	// A long operation is a condition named like the status word it
	// supplies, and the operation's own start is the time a person wants:
	// "Restoring since", whatever else moved in the meantime.
	if operation := in.condition(reason); operation != nil && operation.Status == metav1.ConditionTrue {
		return timeOf(&operation.LastTransitionTime)
	}
	if decidedBy := in.condition(in.decidingCondition(tone)); decidedBy != nil {
		return timeOf(&decidedBy.LastTransitionTime)
	}
	return nil
}

// decidingCondition names the condition whose status decided the tone, for
// the tones a condition decides.
func (in Input) decidingCondition(tone Tone) string {
	if tone != ToneDeleting && in.unattended() != nil {
		return corev1alpha1.ConditionUnattended
	}
	switch tone {
	case ToneError:
		return corev1alpha1.ConditionStalled
	case ToneAttention:
		return corev1alpha1.ConditionNeedsAttention
	case ToneProgress:
		return corev1alpha1.ConditionReconciling
	case ToneOK:
		return corev1alpha1.ConditionReady
	case ToneStopped:
		return conditionSuspended
	case ToneDeleting, ToneUnknown:
		return ""
	}
	return ""
}

// sentence joins availability and the status word.
func (in Input) sentence(d Display) string {
	if d.Tone == ToneDeleting {
		if d.Reason == corev1alpha1.ReasonDeleting {
			return d.Label
		}
		// Cleanup is waiting for something, and the status word says what.
		return Label(corev1alpha1.ReasonDeleting) + sentenceSeparator + d.Label
	}
	switch {
	case in.neverUp(), d.Tone == ToneStopped:
		// A database that has never been up is not "down", and neither is
		// one that was stopped on purpose.
		return d.Label
	case d.Available == AvailableYes:
		return "Up" + sentenceSeparator + d.Label
	case d.Available == AvailableNo:
		return "Down" + sentenceSeparator + d.Label
	default:
		return d.Label
	}
}

// Outcome is where a wait for an Instance to become ready stands.
type Outcome string

const (
	// Pending means the Instance may still become ready: keep waiting.
	Pending Outcome = "pending"
	// Succeeded means the Instance is ready.
	Succeeded Outcome = "succeeded"
	// Failed means the Instance will not become ready by waiting.
	Failed Outcome = "failed"
)

// WaitOutcome says whether waiting for the Instance to become ready is over.
// It succeeds once Ready is True and fails once Stalled is True, in both
// cases only for the current spec: a status that describes an older one
// decides nothing. An Instance that is being deleted never becomes ready.
func WaitOutcome(in Input) Outcome {
	switch {
	case in.deleting():
		return Failed
	case in.stale():
		return Pending
	case in.isTrue(corev1alpha1.ConditionStalled):
		return Failed
	case in.isTrue(corev1alpha1.ConditionReady):
		return Succeeded
	default:
		return Pending
	}
}

func (in Input) meta() *metav1.ObjectMeta {
	if in.Meta == nil {
		return &metav1.ObjectMeta{}
	}
	return in.Meta
}

func (in Input) condition(conditionType string) *metav1.Condition {
	return meta.FindStatusCondition(in.Conditions, conditionType)
}

func (in Input) isTrue(conditionType string) bool {
	return meta.IsStatusConditionTrue(in.Conditions, conditionType)
}

func (in Input) deleting() bool {
	return in.meta().DeletionTimestamp != nil
}

// neverUp reports whether the database has not been available yet. The
// phase is the only record of that, and it only changes the wording.
func (in Input) neverUp() bool {
	return in.Phase == "" || in.Phase == string(corev1alpha1.InstancePhasePending)
}

// stale reports whether the status describes an older spec than the current
// one. An Instance no provider has reconciled yet is stale too: its observed
// generation is the API default, which is below any generation.
func (in Input) stale() bool {
	return in.ObservedGeneration < in.meta().Generation
}

// unattended returns the Unattended condition core writes while nobody
// attends to the Instance's current spec, if it is to be honoured: only
// while the status is stale, since a provider that has caught up has
// answered it, and only True or Unknown, the statuses core writes.
func (in Input) unattended() *metav1.Condition {
	if !in.stale() {
		return nil
	}
	c := in.condition(corev1alpha1.ConditionUnattended)
	if c == nil || (c.Status != metav1.ConditionTrue && c.Status != metav1.ConditionUnknown) {
		return nil
	}
	return c
}

// timeOf copies a timestamp out of the object, or returns nil if it is unset.
func timeOf(t *metav1.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	at := t.Time
	return &at
}
