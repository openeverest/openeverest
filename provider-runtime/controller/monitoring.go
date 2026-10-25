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

package controller

import (
	"fmt"
	"slices"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	apicommon "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/monitoring"
)

// ResolvedBinding is one MonitoringBinding of the Instance with the objects
// it references resolved.
type ResolvedBinding struct {
	Binding *monitoringv1alpha1.MonitoringBinding
	// Destination is nil when the referenced MonitoringDestination does not
	// exist.
	Destination *monitoringv1alpha1.MonitoringDestination
	// Class is nil while the binding has no class or the class does not
	// exist.
	Class *monitoringv1alpha1.MonitoringClass
	// Entry is the matching Instance.spec.monitoring.destinations[] entry,
	// nil when the entry was removed and the binding is about to be deleted.
	Entry *v1alpha1.InstanceMonitoringDestination
	// Apply is true when the binding is ProviderManaged, its entry is still
	// listed and names the binding's destination, it is Accepted for its
	// current generation and its destination and class exist: the provider
	// MUST render it. When false for a ProviderManaged binding with an
	// Entry, the provider MUST keep whatever the live engine CR carries
	// (hold rule); a binding without an Entry is not rendered.
	Apply bool
}

// Mode returns the binding's execution mode, empty until core resolves it.
func (b ResolvedBinding) Mode() v1alpha1.MonitoringExecutionMode {
	return b.Binding.Spec.ExecutionMode
}

// Integration returns the ProviderManaged integration name of the class, or
// "".
func (b ResolvedBinding) Integration() string {
	if b.Class == nil || b.Class.Spec.ProviderManaged == nil {
		return ""
	}
	return b.Class.Spec.ProviderManaged.Integration
}

// bindingResult is a per-binding outcome staged during Sync.
type bindingResult struct {
	reason  monitoringv1alpha1.ConfiguredReason
	message string
}

// MonitoringBindings lists the Instance's bindings with destination, class
// and spec entry resolved.
func (c *Context) MonitoringBindings() ([]ResolvedBinding, error) {
	bindings, err := monitoring.ListBindingsForInstance(c.ctx, c.client, c.in)
	if err != nil {
		return nil, err
	}
	entries := make(map[string]*v1alpha1.InstanceMonitoringDestination)
	if c.in.Spec.Monitoring != nil {
		for i := range c.in.Spec.Monitoring.Destinations {
			entries[c.in.Spec.Monitoring.Destinations[i].Name] = &c.in.Spec.Monitoring.Destinations[i]
		}
	}

	out := make([]ResolvedBinding, 0, len(bindings))
	for i := range bindings {
		b := &bindings[i]
		rb := ResolvedBinding{
			Binding: b,
			Entry:   entries[b.Labels[monitoringv1alpha1.BindingEntryLabel]],
		}
		dst := &monitoringv1alpha1.MonitoringDestination{}
		if err := c.client.Get(c.ctx, client.ObjectKey{Namespace: b.Namespace, Name: b.Spec.DestinationRef.Name}, dst); err == nil {
			rb.Destination = dst
		} else if !apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("get MonitoringDestination %q: %w", b.Spec.DestinationRef.Name, err)
		}
		if b.Spec.ClassRef != nil {
			class := &monitoringv1alpha1.MonitoringClass{}
			if err := c.client.Get(c.ctx, client.ObjectKey{Name: b.Spec.ClassRef.Name}, class); err == nil {
				rb.Class = class
			} else if !apierrors.IsNotFound(err) {
				return nil, fmt.Errorf("get MonitoringClass %q: %w", b.Spec.ClassRef.Name, err)
			}
		}
		rb.Apply = b.Spec.ExecutionMode == v1alpha1.MonitoringExecutionModeProviderManaged && rb.Entry != nil &&
			kindDefaulted(b.Spec.DestinationRef) == kindDefaulted(rb.Entry.DestinationRef) &&
			monitoring.IsAccepted(b) && rb.Destination != nil && rb.Class != nil
		out = append(out, rb)
	}
	return out, nil
}

// kindDefaulted returns ref with the kind the CRD defaults to when unset.
func kindDefaulted(ref apicommon.MonitoringDestinationRef) apicommon.MonitoringDestinationRef {
	if ref.Kind == "" {
		ref.Kind = "MonitoringDestination"
	}
	return ref
}

// Requirement is the three-valued answer to "does a bound class need this
// source?". The zero value behaves as Unresolved, so a provider that gets an
// error from RequiredSources still holds whatever is live.
type Requirement string

const (
	// Required means a class bound to the Instance needs the source: start
	// it.
	Required Requirement = "Required"
	// NotRequired means every entry of spec.monitoring.destinations is
	// resolved and none of their classes needs the source: stop it.
	NotRequired Requirement = "NotRequired"
	// Unresolved means some entry has no binding with a resolved mode and
	// class yet (core still materialising it, class being reinstalled).
	// This is the hold rule: keep whatever is live, neither start nor stop.
	Unresolved Requirement = "Unresolved"
)

// Else returns whether the source should run: true for Required, false for
// NotRequired and current, what is live now, for Unresolved and the zero
// value. A provider's exporter gate is one line:
// req.Metrics.Else(hasExporter(liveCR)).
func (r Requirement) Else(current bool) bool {
	switch r {
	case Required:
		return true
	case NotRequired:
		return false
	case Unresolved:
		return current
	}
	return current
}

// RequiredSources is what the classes bound to an Instance need from it.
// Bindings whose entry was removed from spec.monitoring.destinations are
// ignored: only removing the entry removes what it asked for.
type RequiredSources struct {
	// Metrics is Required when an ExtensionManaged binding's class requires
	// metrics. Acceptance is deliberately not looked at: a binding that is
	// not Accepted still records that the user asked for metrics, and the
	// exporter must not flap with it. It is Unresolved while an entry has
	// no binding, or one without a mode or, for ExtensionManaged, a class.
	Metrics Requirement
	// Integrations has one key per integration the Provider declares in
	// spec.monitoring.integrations or the provider returns from
	// MonitoringIntegrations. An integration is Required while one of its
	// ProviderManaged bindings has Apply set; Unresolved while one of its
	// bindings is held (Apply false), or an entry has no binding, or one
	// without a mode or, for ProviderManaged, a class; NotRequired
	// otherwise. The gate is one line per integration,
	// req.Integrations["pmm"].Else(livePMMEnabled): hold every live
	// integration while it is Unresolved. A missing key reads as the zero
	// Requirement, which holds as well.
	Integrations map[string]Requirement
}

// RequiredSources tells the provider which sources and integrations the
// classes bound to the Instance need, so it runs exporters and agents only
// while something consumes them.
func (c *Context) RequiredSources() (RequiredSources, error) {
	bindings, err := c.MonitoringBindings()
	if err != nil {
		return RequiredSources{}, err
	}
	spec, err := c.ProviderSpec()
	if err != nil {
		return RequiredSources{}, err
	}
	integrations := make(map[string]Requirement)
	if spec.Monitoring != nil {
		for name := range spec.Monitoring.Integrations {
			integrations[name] = NotRequired
		}
	}
	for _, name := range c.monitoringIntegrations {
		integrations[name] = NotRequired
	}

	metricsResolved := make(map[string]bool)
	integrationsResolved := make(map[string]bool)
	metrics := false
	for _, rb := range bindings {
		if rb.Entry == nil {
			continue
		}
		switch rb.Mode() {
		case v1alpha1.MonitoringExecutionModeProviderManaged:
			metricsResolved[rb.Entry.Name] = true
			if rb.Class == nil {
				continue
			}
			integrationsResolved[rb.Entry.Name] = true
			name := rb.Integration()
			current, declared := integrations[name]
			if !declared {
				continue
			}
			switch {
			case rb.Apply:
				integrations[name] = Required
			case current != Required:
				integrations[name] = Unresolved
			}
		case v1alpha1.MonitoringExecutionModeExtensionManaged:
			integrationsResolved[rb.Entry.Name] = true
			if rb.Class == nil {
				continue
			}
			metricsResolved[rb.Entry.Name] = true
			if em := rb.Class.Spec.ExtensionManaged; em != nil && em.Requires.Metrics {
				metrics = true
			}
		}
	}

	var entries []v1alpha1.InstanceMonitoringDestination
	if c.in.Spec.Monitoring != nil {
		entries = c.in.Spec.Monitoring.Destinations
	}
	unresolved := func(resolved map[string]bool) bool {
		return slices.ContainsFunc(entries, func(e v1alpha1.InstanceMonitoringDestination) bool { return !resolved[e.Name] })
	}
	req := RequiredSources{Metrics: NotRequired, Integrations: integrations}
	switch {
	case metrics:
		req.Metrics = Required
	case unresolved(metricsResolved):
		req.Metrics = Unresolved
	}
	if unresolved(integrationsResolved) {
		for name, r := range integrations {
			if r != Required {
				integrations[name] = Unresolved
			}
		}
	}
	return req, nil
}

// SetMonitoringIntegrations records the integrations the provider renders,
// from MonitoringIntegrationsProvider (used internally by the reconciler
// before Sync).
func (c *Context) SetMonitoringIntegrations(names ...string) {
	c.monitoringIntegrations = names
}

// SetMonitoringBindingResult stages the outcome of one ProviderManaged
// binding, keyed by binding name. After Sync the runtime writes it on the
// binding's Configured condition with the status the reason implies; it never
// fails Sync. Bindings without a staged result are reported Configured.
func (c *Context) SetMonitoringBindingResult(bindingName string, reason monitoringv1alpha1.ConfiguredReason, message string) {
	if c.monitoringResults == nil {
		c.monitoringResults = make(map[string]bindingResult)
	}
	c.monitoringResults[bindingName] = bindingResult{reason: reason, message: message}
}

// MonitoringBindingResult returns the result staged for a binding, if any
// (used internally by the reconciler).
func (c *Context) MonitoringBindingResult(bindingName string) (monitoringv1alpha1.ConfiguredReason, string, bool) {
	r, ok := c.monitoringResults[bindingName]
	return r.reason, r.message, ok
}
