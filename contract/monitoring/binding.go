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

// Package monitoring holds the rules on the monitoring kinds that core, the
// provider runtime and monitoring extensions share: how a MonitoringBinding
// is named and labelled, which field manager writes which part of the status
// core owns, when a binding or a class counts as Accepted, and how a
// binding's Ready condition follows from its other conditions and its
// destination. Core writes that condition on the binding and the provider
// runtime reports the same verdict on the Instance's monitoring entry, so
// both derive it here.
package monitoring

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
)

// Field managers of the status fields core writes. Every writer applies
// under its own field manager so no writer erases another's fields.
const (
	// MaterializerFieldOwner owns MonitoringBinding.spec and the Accepted
	// condition on bindings and destinations.
	MaterializerFieldOwner = "openeverest.io/binding-materializer"
	// ReadinessFieldOwner owns the Ready condition on bindings.
	ReadinessFieldOwner = "openeverest.io/binding-readiness"
	// CompatibilityFieldOwner owns MonitoringClass.status.providers.
	CompatibilityFieldOwner = "openeverest.io/monitoring-compatibility"
)

const (
	bindingNameMaxLength = 63
	bindingNameHashChars = 8
)

// BindingName returns the deterministic MonitoringBinding name for an
// Instance entry: <instance>-<entry>-<first 8 hex chars of
// sha256("<instance>/<entry>")>. The prefix is truncated so the name fits in
// 63 characters.
func BindingName(instance, entry string) string {
	sum := sha256.Sum256([]byte(instance + "/" + entry))
	suffix := hex.EncodeToString(sum[:])[:bindingNameHashChars]
	base := instance + "-" + entry
	if maxBase := bindingNameMaxLength - 1 - bindingNameHashChars; len(base) > maxBase {
		base = strings.TrimRight(base[:maxBase], "-.")
	}
	return base + "-" + suffix
}

// BindingLabels returns the labels core stamps on a binding. The class label
// is omitted while the class is not resolved yet.
func BindingLabels(instance, entry, destination, class string) map[string]string {
	l := map[string]string{
		monitoringv1alpha1.BindingInstanceLabel:    instance,
		monitoringv1alpha1.BindingEntryLabel:       entry,
		monitoringv1alpha1.BindingDestinationLabel: destination,
	}
	if class != "" {
		l[monitoringv1alpha1.BindingClassLabel] = class
	}
	return l
}

// IsAccepted reports whether the binding's Accepted condition is True for
// the binding's current generation.
func IsAccepted(b *monitoringv1alpha1.MonitoringBinding) bool {
	cond := meta.FindStatusCondition(b.Status.Conditions, monitoringv1alpha1.MonitoringBindingConditionAccepted)
	return cond != nil && cond.Status == metav1.ConditionTrue && cond.ObservedGeneration == b.Generation
}

// ClassIsAccepted reports whether a class has been claimed by its controller
// at its current generation. An Accepted condition observed at another
// generation is stale and counts as not accepted.
func ClassIsAccepted(class *monitoringv1alpha1.MonitoringClass) bool {
	cond := meta.FindStatusCondition(class.Status.Conditions, monitoringv1alpha1.MonitoringClassConditionAccepted)
	return cond != nil && cond.Status == metav1.ConditionTrue && cond.ObservedGeneration == class.Generation
}
