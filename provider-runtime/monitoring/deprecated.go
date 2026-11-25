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

package monitoring

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
	contractmonitoring "github.com/openeverest/openeverest/v2/contract/monitoring"
)

// The identifiers below moved to contract/monitoring, which core and the
// provider runtime both import, so that core no longer imports the provider
// runtime. They stay here for one release so that extensions can move at
// their own pace.

const (
	// MaterializerFieldOwner is contractmonitoring.MaterializerFieldOwner.
	//
	// Deprecated: use contract/monitoring.MaterializerFieldOwner; removed in
	// the next release.
	MaterializerFieldOwner = contractmonitoring.MaterializerFieldOwner
	// ReadinessFieldOwner is contractmonitoring.ReadinessFieldOwner.
	//
	// Deprecated: use contract/monitoring.ReadinessFieldOwner; removed in the
	// next release.
	ReadinessFieldOwner = contractmonitoring.ReadinessFieldOwner
	// CompatibilityFieldOwner is contractmonitoring.CompatibilityFieldOwner.
	//
	// Deprecated: use contract/monitoring.CompatibilityFieldOwner; removed in
	// the next release.
	CompatibilityFieldOwner = contractmonitoring.CompatibilityFieldOwner
)

// BindingName returns the deterministic MonitoringBinding name for an
// Instance entry.
//
// Deprecated: use contract/monitoring.BindingName; removed in the next release.
func BindingName(instance, entry string) string {
	return contractmonitoring.BindingName(instance, entry)
}

// BindingLabels returns the labels core stamps on a binding.
//
// Deprecated: use contract/monitoring.BindingLabels; removed in the next
// release.
func BindingLabels(instance, entry, destination, class string) map[string]string {
	return contractmonitoring.BindingLabels(instance, entry, destination, class)
}

// IsAccepted reports whether the binding's Accepted condition is True for
// the binding's current generation.
//
// Deprecated: use contract/monitoring.IsAccepted; removed in the next release.
func IsAccepted(b *monitoringv1alpha1.MonitoringBinding) bool {
	return contractmonitoring.IsAccepted(b)
}

// ClassIsAccepted reports whether a class has been claimed by its controller
// at its current generation.
//
// Deprecated: use contract/monitoring.ClassIsAccepted; removed in the next
// release.
func ClassIsAccepted(class *monitoringv1alpha1.MonitoringClass) bool {
	return contractmonitoring.ClassIsAccepted(class)
}

// BindingReady derives the Ready condition of binding b from its Accepted
// and Configured conditions and its destination dst.
//
// Deprecated: use contract/monitoring.BindingReady; removed in the next
// release.
func BindingReady(
	b *monitoringv1alpha1.MonitoringBinding,
	accepted, configured metav1.Condition,
	dst *monitoringv1alpha1.MonitoringDestination,
) metav1.Condition {
	return contractmonitoring.BindingReady(b, accepted, configured, dst)
}

// DestinationReadiness returns the status and reason of the destination's
// Ready condition at its current generation.
//
// Deprecated: use contract/monitoring.DestinationReadiness; removed in the next
// release.
func DestinationReadiness(dst *monitoringv1alpha1.MonitoringDestination) (metav1.ConditionStatus, string) {
	return contractmonitoring.DestinationReadiness(dst)
}
