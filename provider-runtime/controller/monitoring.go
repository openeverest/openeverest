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

// =============================================================================
// MONITORING (Optional interfaces for the extensible monitoring contract)
// =============================================================================

// MonitoringSourcesProvider is an optional interface a provider implements to publish
// the per-Instance monitoring sources (identity, credentials, scrape
// endpoints) that ExtensionManaged MonitoringClasses consume.
//
// When implemented the runtime:
//   - watches MonitoringBindings of the provider's Instances and re-enqueues
//     the Instance when one changes,
//   - writes Instance.status.monitoring.sources after every successful Sync (the
//     runtime is the only writer of Instance status),
//   - writes Instance.status.monitoring and the MonitoringConfigured condition.
type MonitoringSourcesProvider interface {
	// MonitoringSources returns the endpoints that are currently serving.
	// Returning nil clears Instance.status.monitoring.sources.
	MonitoringSources(c *Context) (*v1alpha1.MonitoringSources, error)
}

// MonitoringIntegrationsProvider is an optional interface a provider implements to fulfil
// ProviderManaged MonitoringClasses through its engine operator's native
// integration (e.g. Percona spec.pmm).
//
// The provider renders the integration inside Sync from
// Context.MonitoringBindings(), on the same Apply as the rest of the engine
// CR. The runtime owns the bindings' Configured condition: it derives
// Configured from Sync success and from the results staged with
// Context.SetMonitoringBindingResult, and it never fails Sync over a binding.
type MonitoringIntegrationsProvider interface {
	// MonitoringIntegrations returns the integration names the provider can
	// render, matching Provider.spec.monitoring.integrations.
	MonitoringIntegrations() []string
}

// ResolvedBinding is one MonitoringBinding of the Instance with the objects
// it references resolved.
type ResolvedBinding struct {
	Binding *monitoringv1alpha1.MonitoringBinding
	// Destination is nil when the referenced MonitoringDestination does not
	// exist.
	Destination *monitoringv1alpha1.MonitoringDestination
	// Class is nil when the binding has not resolved a class yet.
	Class *monitoringv1alpha1.MonitoringClass
	// Entry is the matching Instance.spec.monitoring.destinations[] entry, nil
	// if the entry has been removed and the binding is about to be deleted.
	Entry *v1alpha1.InstanceMonitoringDestination
	// Apply is true when the binding is ProviderManaged and Accepted: the
	// provider MUST render it. When false for a ProviderManaged binding the
	// provider MUST keep whatever the live engine CR carries (hold rule).
	Apply bool
}

// Mode returns the binding's execution mode.
func (b ResolvedBinding) Mode() v1alpha1.MonitoringExecutionMode {
	return b.Binding.Spec.ExecutionMode
}

// Integration returns the ProviderManaged integration name, or "".
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
	bindings, err := monitoring.ListBindingsForInstance(c.ctx, c.client, c.in.Namespace, c.in.Name)
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
		rb.Apply = b.Spec.ExecutionMode == v1alpha1.MonitoringExecutionModeProviderManaged &&
			monitoring.IsAccepted(b) && rb.Destination != nil && rb.Class != nil
		out = append(out, rb)
	}
	return out, nil
}

// SetMonitoringBindingResult stages the outcome of one ProviderManaged
// binding (keyed by binding name). The runtime reports it on the binding's
// Configured condition after Sync, with the status the reason implies; it
// never aborts Sync.
func (c *Context) SetMonitoringBindingResult(bindingName string, reason monitoringv1alpha1.ConfiguredReason, message string) {
	if c.monitoringResults == nil {
		c.monitoringResults = map[string]bindingResult{}
	}
	c.monitoringResults[bindingName] = bindingResult{reason: reason, message: message}
}

// monitoringResult returns the staged result for a binding, if any.
func (c *Context) monitoringResult(bindingName string) (reason monitoringv1alpha1.ConfiguredReason, message string, ok bool) {
	r, ok := c.monitoringResults[bindingName]
	return r.reason, r.message, ok
}

// MonitoringBindingResult returns the staged result for a binding, if any.
// Used by the reconciler after Sync.
func (c *Context) MonitoringBindingResult(bindingName string) (reason monitoringv1alpha1.ConfiguredReason, message string, ok bool) {
	return c.monitoringResult(bindingName)
}
