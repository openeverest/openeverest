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

// Package gate holds a new build of an extension at startup until the core
// installed in the cluster supports the contract version it links.
//
// Every extension workload built with an OpenEverest SDK runs it before it
// starts controllers, takes part in leader election or serves requests. The
// checks at install time and at admission stop an incompatible extension on
// most apply paths, but not on all of them: manifests rendered ahead of
// time, an admission webhook that does not answer, an image overridden
// after install. The gate runs in the build itself, so it covers every path
// the same way.
//
// Only a new rollout is held: a build whose contract version differs from
// the version observed on its root object, or whose root object has none
// yet. A build with the observed version was let in when it first started,
// so it starts without a check: a pod restart, a node drain or a core rolled
// back underneath it never takes down an extension that already runs. A
// held build stays alive but unready, so under a RollingUpdate the pods of
// the previous build keep serving, and it starts by itself once core is
// upgraded.
//
// Decide is the decision, a pure function of what the gate reads. Runner
// reads, decides and waits, and records the version that starts, through
// functions the workload provides, so the package needs no Kubernetes
// client.
package gate

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openeverest/openeverest/v2/contract"
)

// Reasons of a Decision other than those of contract.Result.
const (
	// ReasonAlreadyRunning means the build links the contract version
	// observed on its root object: it was let in when it first started, so
	// it is not checked again.
	ReasonAlreadyRunning = "AlreadyRunning"
	// ReasonReadFailed means the gate could not read the root object or the
	// CRD the core publishes its window on, so it cannot decide yet.
	ReasonReadFailed = "ReadFailed"
)

// Core is what a core publishes on the CRD of an extension's root kind.
type Core struct {
	// Window is the window of contract versions the core serves.
	Window contract.Window
	// HasWindow is false when the CRD publishes no window, because the core
	// predates published windows, or when the CRD does not exist, because
	// no core is installed.
	HasWindow bool
	// Version is the core release, the contract.CoreVersionKey label.
	Version string
	// AcknowledgedExtensions is the list of acknowledgements the core chart
	// publishes, the value of contract.AcknowledgedExtensionsAnnotation.
	AcknowledgedExtensions string
}

// CoreFromCRD returns what a core publishes in the labels and annotations
// of the CRD of an extension's root kind.
func CoreFromCRD(labels, annotations map[string]string) Core {
	w, ok := contract.WindowFromAnnotations(annotations)
	return Core{
		Window:                 w,
		HasWindow:              ok,
		Version:                labels[contract.CoreVersionKey],
		AcknowledgedExtensions: annotations[contract.AcknowledgedExtensionsAnnotation],
	}
}

// Root is what the gate reads on the extension's root object: its
// Provider, MonitoringClass or Plugin. The zero Root stands for a root
// object that does not exist yet.
type Root struct {
	// Observed is the contract version on the root object's status,
	// written by the build that last started.
	Observed string
	// Acknowledged is the core release on which the extension chart
	// acknowledges an incompatibility, the value of
	// contract.AcknowledgedAnnotation.
	Acknowledged string
}

// Decision is whether a build may start, and why.
type Decision struct {
	// Start reports whether the build may start.
	Start bool
	// Reason is ReasonAlreadyRunning or ReasonReadFailed when the build was
	// not checked, and the reason of Result when it was.
	Reason string
	// Result is the result of checking the build against the window, in
	// its acknowledged form when an acknowledgement lets the build start.
	// It is the zero Result when the build was not checked.
	Result contract.Result
	// Acknowledged reports whether the build starts only because the
	// incompatibility was acknowledged.
	Acknowledged bool
	// WaitingFor is the OpenEverest release a held build waits for: the
	// release it was built for, when the core is too old for it and that
	// release is not a development value. It is empty otherwise.
	WaitingFor string
	// Message describes the decision to a user. It names core releases
	// only: a development value is described as an unreleased OpenEverest.
	Message string
}

// Decide reports whether a build may start. name is the name of its root
// object, by which the core's acknowledgements name the extension, linked
// the contract version the build links, root what its root object holds,
// and core what the core publishes on the CRD of the root kind.
//
// A build with the version observed on its root object starts. Any other
// build starts when core's window supports it, or when an acknowledgement
// holds, by contract.AcknowledgementHolds, on the root object or in core's
// list. A core that publishes no window, because it predates published
// windows or is not installed, supports no build: it cannot interpret what
// a build for a newer contract writes, and installing a newer core is what
// lets the build start. A window or a version that is not valid holds the
// build too, unless acknowledged, as there is nothing reliable to compare.
func Decide(name, linked string, root Root, core Core) Decision {
	if root.Observed != "" && linked == root.Observed {
		return Decision{
			Start:   true,
			Reason:  ReasonAlreadyRunning,
			Message: "this build" + builtFor(linked) + " is the one that last started, so it starts again without a compatibility check",
		}
	}

	result := contract.Result{Status: metav1.ConditionFalse, Reason: contract.ReasonCoreTooOld}
	if core.HasWindow {
		result = core.Window.Check(linked)
	}
	newBuild := "a new build" + builtFor(linked)
	if result.Status == metav1.ConditionTrue {
		return Decision{
			Start:   true,
			Reason:  result.Reason,
			Result:  result,
			Message: newBuild + " starts: it " + contract.Message(result, linked, core.Window),
		}
	}

	if acknowledged(name, root, core) {
		result = contract.Acknowledge(result)
		msg := contract.Message(result, linked, core.Window)
		if result.Reason != contract.ReasonCoreTooOldAcknowledged && result.Reason != contract.ReasonExtensionTooOldAcknowledged {
			msg += "; allowed anyway because the incompatibility was acknowledged"
		}
		return Decision{
			Start:        true,
			Reason:       result.Reason,
			Result:       result,
			Acknowledged: true,
			Message:      newBuild + " starts: it " + msg,
		}
	}

	d := Decision{Reason: result.Reason, Result: result}
	if result.Reason == contract.ReasonCoreTooOld && contract.Valid(linked) && !contract.IsDevelopment(linked) {
		d.WaitingFor = linked
		d.Message = fmt.Sprintf("%s is waiting until OpenEverest %s is installed", newBuild, linked)
		if !core.HasWindow {
			d.Message += fmt.Sprintf("; the OpenEverest in this cluster does not publish the versions it supports, which OpenEverest %s does", linked)
		}
		return d
	}
	d.Message = newBuild + " is waiting until the installed OpenEverest supports it: it " + contract.Message(result, linked, core.Window)
	return d
}

// acknowledged reports whether an acknowledgement of the extension named
// name holds on core: the one on its root object, or an entry for it in
// core's list. A list that does not parse acknowledges nothing, as
// contract.ParseAcknowledgedExtensions requires.
func acknowledged(name string, root Root, core Core) bool {
	holds := func(ack string) bool {
		return contract.AcknowledgementHolds(ack, core.Window.Current, core.Version)
	}
	if root.Acknowledged != "" && holds(root.Acknowledged) {
		return true
	}
	acks, err := contract.ParseAcknowledgedExtensions(core.AcknowledgedExtensions)
	if err != nil {
		return false
	}
	ack, ok := acks[name]
	return ok && holds(ack)
}

// builtFor names, in parentheses after a space, the OpenEverest a build
// linking linked was made for. A development value, which nobody can
// install, is described as an unreleased OpenEverest, and a value that is
// not valid is left out, as the message about it quotes it already.
func builtFor(linked string) string {
	switch {
	case !contract.Valid(linked):
		return ""
	case contract.IsDevelopment(linked):
		return " (for an unreleased OpenEverest)"
	default:
		return " (for OpenEverest " + linked + ")"
	}
}
