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

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

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
	// listed, it is Accepted for its current generation and its destination
	// and class exist: the provider MUST render it. When false for a
	// ProviderManaged binding with an Entry, the provider MUST keep whatever
	// the live engine CR carries (hold rule); a binding without an Entry is
	// not rendered.
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
	entries := map[string]*v1alpha1.InstanceMonitoringDestination{}
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
			monitoring.IsAccepted(b) && rb.Destination != nil && rb.Class != nil
		out = append(out, rb)
	}
	return out, nil
}

// Requirement is the three-valued answer to "does a bound class need this
// source?".
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
// NotRequired and current, what is live now, for Unresolved. A provider's
// exporter gate is one line: req.Metrics.Else(hasExporter(liveCR)).
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

// RequiredSources is the union of the extensionManaged.requires blocks of
// the classes bound to an Instance.
type RequiredSources struct {
	// Metrics is Required when an ExtensionManaged binding's class requires
	// metrics. Acceptance is deliberately not looked at: a binding that is
	// not Accepted still records that the user asked for metrics, and the
	// exporter must not flap with it.
	Metrics Requirement
}

// RequiredSources tells the provider which sources the classes bound to the
// Instance need, so it runs exporters only while something consumes them.
func (c *Context) RequiredSources() (RequiredSources, error) {
	bindings, err := c.MonitoringBindings()
	if err != nil {
		return RequiredSources{}, err
	}
	resolved := map[string]bool{}
	metrics := false
	for _, rb := range bindings {
		switch rb.Mode() {
		case v1alpha1.MonitoringExecutionModeProviderManaged:
			resolved[rb.Binding.Labels[monitoringv1alpha1.BindingEntryLabel]] = true
		case v1alpha1.MonitoringExecutionModeExtensionManaged:
			if rb.Class == nil {
				continue
			}
			resolved[rb.Binding.Labels[monitoringv1alpha1.BindingEntryLabel]] = true
			if em := rb.Class.Spec.ExtensionManaged; em != nil && em.Requires.Metrics {
				metrics = true
			}
		}
	}
	if metrics {
		return RequiredSources{Metrics: Required}, nil
	}
	if c.in.Spec.Monitoring != nil {
		for _, e := range c.in.Spec.Monitoring.Destinations {
			if !resolved[e.Name] {
				return RequiredSources{Metrics: Unresolved}, nil
			}
		}
	}
	return RequiredSources{Metrics: NotRequired}, nil
}

// SetMonitoringBindingResult stages the outcome of one ProviderManaged
// binding, keyed by binding name. After Sync the runtime writes it on the
// binding's Configured condition with the status the reason implies; it never
// fails Sync. Bindings without a staged result are reported Configured.
func (c *Context) SetMonitoringBindingResult(bindingName string, reason monitoringv1alpha1.ConfiguredReason, message string) {
	if c.monitoringResults == nil {
		c.monitoringResults = map[string]bindingResult{}
	}
	c.monitoringResults[bindingName] = bindingResult{reason: reason, message: message}
}

// MonitoringBindingResult returns the result staged for a binding, if any
// (used internally by the reconciler).
func (c *Context) MonitoringBindingResult(bindingName string) (monitoringv1alpha1.ConfiguredReason, string, bool) {
	r, ok := c.monitoringResults[bindingName]
	return r.reason, r.message, ok
}
