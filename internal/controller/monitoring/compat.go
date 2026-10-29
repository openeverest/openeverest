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
	"fmt"
	"maps"
	"slices"

	"github.com/blang/semver/v4"

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
)

// verdict says whether one execution mode can fulfil a class and, if not, why.
type verdict struct {
	ok      bool
	reason  string
	message string
}

// fulfilment is how a MonitoringClass can be fulfilled for a Provider, from
// static data only. Binding acceptance and MonitoringClass.status.providers
// are both derived from it, so they cannot disagree.
type fulfilment struct {
	class            *monitoringv1alpha1.MonitoringClass
	providerManaged  verdict
	extensionManaged verdict
	// extensionDisruption is the worst disruption over the eligible
	// component types.
	extensionDisruption monitoringv1alpha1.Disruption
}

// assess evaluates class for provider. authMode is the destination's
// authentication mode, or "" to accept any of the class's modes. Only
// componentTypes are considered for ExtensionManaged.
func assess(provider *corev1alpha1.Provider, class *monitoringv1alpha1.MonitoringClass, authMode string, componentTypes []string) fulfilment {
	f := fulfilment{class: class, providerManaged: providerManaged(provider, class, authMode)}
	f.extensionManaged, f.extensionDisruption = extensionManaged(provider, class, componentTypes)
	return f
}

// preferred picks ProviderManaged over ExtensionManaged. When neither fits,
// the reason is that of the only block the class declares, or Incompatible
// naming both.
func (f fulfilment) preferred() (corev1alpha1.MonitoringExecutionMode, string, string) {
	switch {
	case f.providerManaged.ok:
		return corev1alpha1.MonitoringExecutionModeProviderManaged, "", ""
	case f.extensionManaged.ok:
		return corev1alpha1.MonitoringExecutionModeExtensionManaged, "", ""
	case f.class.Spec.ExtensionManaged == nil:
		return "", f.providerManaged.reason, f.providerManaged.message
	case f.class.Spec.ProviderManaged == nil:
		return "", f.extensionManaged.reason, f.extensionManaged.message
	}
	reason := monitoringv1alpha1.ReasonIncompatible
	if f.providerManaged.reason == monitoringv1alpha1.ReasonIntegrationInUse {
		reason = f.providerManaged.reason
	}
	return "", reason, fmt.Sprintf("providerManaged: %s; extensionManaged: %s", f.providerManaged.message, f.extensionManaged.message)
}

// providerCompatibility is the MonitoringClass.status.providers entry for
// one Provider: every component type the Provider declares is considered.
func providerCompatibility(provider *corev1alpha1.Provider, class *monitoringv1alpha1.MonitoringClass) monitoringv1alpha1.ProviderCompatibility {
	var componentTypes []string
	if provider.Spec.Monitoring != nil {
		componentTypes = slices.Sorted(maps.Keys(provider.Spec.Monitoring.ComponentTypes))
	}
	f := assess(provider, class, "", componentTypes)
	mode, reason, message := f.preferred()
	pc := monitoringv1alpha1.ProviderCompatibility{Name: provider.Name, Mode: mode, Reason: reason, Message: message}
	switch mode {
	case corev1alpha1.MonitoringExecutionModeProviderManaged:
		// A native integration is an agent the operator injects into the pods.
		pc.Disruption = monitoringv1alpha1.DisruptionRollingRestart
	case corev1alpha1.MonitoringExecutionModeExtensionManaged:
		pc.Disruption = f.extensionDisruption
	}
	return pc
}

// instanceComponentTypes returns the distinct component types of the
// Instance's components, sorted.
func instanceComponentTypes(in *corev1alpha1.Instance, provider *corev1alpha1.Provider) []string {
	componentTypes := make(map[string]struct{})
	for name, comp := range in.Spec.Components {
		t := comp.Type
		if t == "" {
			t = provider.Spec.Components[name].Type
		}
		if t != "" {
			componentTypes[t] = struct{}{}
		}
	}
	return slices.Sorted(maps.Keys(componentTypes))
}

// providerManaged: the Provider declares the class's integration, renders
// authMode (or, when it is "", one of the class's modes) and the class's
// agentVersion is within the range the Provider renders.
func providerManaged(provider *corev1alpha1.Provider, class *monitoringv1alpha1.MonitoringClass, authMode string) verdict {
	pm := class.Spec.ProviderManaged
	if pm == nil {
		return verdict{reason: monitoringv1alpha1.ReasonIncompatible, message: fmt.Sprintf("MonitoringClass %q declares no providerManaged block", class.Name)}
	}
	if provider.Spec.Monitoring == nil {
		return verdict{reason: monitoringv1alpha1.ReasonIntegrationUnsupported, message: fmt.Sprintf("Provider %q declares no monitoring contract", provider.Name)}
	}
	integration, ok := provider.Spec.Monitoring.Integrations[pm.Integration]
	if !ok {
		return verdict{reason: monitoringv1alpha1.ReasonIntegrationUnsupported, message: fmt.Sprintf("Provider %q does not declare integration %q", provider.Name, pm.Integration)}
	}
	if authMode != "" && !slices.Contains(integration.AuthModes, authMode) {
		return verdict{reason: monitoringv1alpha1.ReasonAuthModeUnsupported, message: fmt.Sprintf("Provider %q does not render authentication mode %q for integration %q", provider.Name, authMode, pm.Integration)}
	}
	if authMode == "" && !slices.ContainsFunc(class.Spec.AuthModeNames(), func(m string) bool { return slices.Contains(integration.AuthModes, m) }) {
		return verdict{reason: monitoringv1alpha1.ReasonAuthModeUnsupported, message: fmt.Sprintf("Provider %q renders none of the authentication modes of MonitoringClass %q for integration %q", provider.Name, class.Name, pm.Integration)}
	}
	if integration.AgentVersions == "" {
		return verdict{ok: true}
	}
	inRange, err := semver.ParseRange(integration.AgentVersions)
	if err != nil {
		return verdict{reason: monitoringv1alpha1.ReasonIncompatible, message: fmt.Sprintf("Provider %q agentVersions %q: %v", provider.Name, integration.AgentVersions, err)}
	}
	v, err := semver.ParseTolerant(pm.AgentVersion)
	if err != nil {
		return verdict{reason: monitoringv1alpha1.ReasonIncompatible, message: fmt.Sprintf("MonitoringClass %q agentVersion %q: %v", class.Name, pm.AgentVersion, err)}
	}
	if !inRange(v) {
		return verdict{reason: monitoringv1alpha1.ReasonIncompatible, message: fmt.Sprintf("MonitoringClass %q agentVersion %s is outside the range %q Provider %q supports", class.Name, pm.AgentVersion, integration.AgentVersions, provider.Name)}
	}
	return verdict{ok: true}
}

// extensionManaged: at least one of componentTypes is eligible for the
// class's extensionManaged block. The disruption is the worst over the
// eligible types.
func extensionManaged(provider *corev1alpha1.Provider, class *monitoringv1alpha1.MonitoringClass, componentTypes []string) (verdict, monitoringv1alpha1.Disruption) {
	em := class.Spec.ExtensionManaged
	if em == nil {
		return verdict{reason: monitoringv1alpha1.ReasonIncompatible, message: fmt.Sprintf("MonitoringClass %q declares no extensionManaged block", class.Name)}, ""
	}
	if provider.Spec.Monitoring == nil {
		return verdict{reason: monitoringv1alpha1.ReasonIncompatible, message: fmt.Sprintf("Provider %q declares no monitoring contract", provider.Name)}, ""
	}
	eligible := false
	worst := monitoringv1alpha1.DisruptionNone
	for _, t := range componentTypes {
		ct, ok := provider.Spec.Monitoring.ComponentTypes[t]
		if !ok || !componentEligible(ct, em.Requires) {
			continue
		}
		eligible = true
		if disruption(ct, em.Requires) == monitoringv1alpha1.DisruptionRollingRestart {
			worst = monitoringv1alpha1.DisruptionRollingRestart
		}
	}
	if !eligible {
		return verdict{reason: monitoringv1alpha1.ReasonIncompatible, message: fmt.Sprintf("no component satisfies the requirements of MonitoringClass %q", class.Name)}, ""
	}
	return verdict{ok: true}, worst
}

// componentEligible: the component exposes metrics if the class requires
// them.
func componentEligible(ct corev1alpha1.MonitoringComponentType, req monitoringv1alpha1.SourceRequirements) bool {
	return !req.Metrics || (ct.Metrics != "" && ct.Metrics != corev1alpha1.MetricsSourceNone)
}

// disruption is what binding the class does to a component's pods: an
// exporter is started as a sidecar, which rolls them.
func disruption(ct corev1alpha1.MonitoringComponentType, req monitoringv1alpha1.SourceRequirements) monitoringv1alpha1.Disruption {
	if req.Metrics && ct.Metrics == corev1alpha1.MetricsSourceExporter {
		return monitoringv1alpha1.DisruptionRollingRestart
	}
	return monitoringv1alpha1.DisruptionNone
}
