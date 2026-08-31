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

// Maintenance gating.
//
// Providers declare each unit of disruptive work through
// Context.RequestMaintenance during Sync, classified by its observable
// database impact. The runtime authorizes anything within the Instance's
// standing tolerance (spec.maintenance.autoApproveUpTo) or explicitly
// approved by token (spec.maintenance.approved); everything else is held and
// flushed to status.pendingMaintenance by the reconciler, so a provider
// upgrade never causes surprise downtime.
//
// Design background: spec 009 (provider upgrades) in the openeverest/specs
// repository.

import (
	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

// MaintenanceSeverity re-exports the Instance API severity scale for provider
// ergonomics.
type MaintenanceSeverity = v1alpha1.MaintenanceSeverity

// Severity levels, ordered by observable impact.
const (
	MaintenanceNonDisruptive  = v1alpha1.MaintenanceNonDisruptive
	MaintenanceRollingRestart = v1alpha1.MaintenanceRollingRestart
	MaintenanceDowntime       = v1alpha1.MaintenanceDowntime
)

// severityRank orders severities for tolerance comparison. Unknown values
// rank above Downtime so a misclassified action is held, never auto-applied.
func severityRank(s MaintenanceSeverity) int {
	switch s {
	case MaintenanceNonDisruptive:
		return 0
	case MaintenanceRollingRestart:
		return 1
	case MaintenanceDowntime:
		return 2
	default:
		return 3
	}
}

// RequestMaintenance declares a unit of work with the given observable impact
// and reports whether the provider may perform it this reconcile.
//
// token identifies the action occurrence: it must be unique per occurrence
// and stable while the action is pending (e.g. "upgrade-to-0.3",
// "rotate-certs-2026H2"). It is both the action's identity on
// status.pendingMaintenance and the value the user echoes into
// spec.maintenance.approved. The framework treats it as an opaque string.
//
// When the result is false the provider MUST NOT perform the action this
// reconcile; the runtime records it on status.pendingMaintenance instead.
// The pending list is rebuilt every reconcile from the calls the provider
// makes, so an action the provider stops requesting disappears on its own.
func (c *Context) RequestMaintenance(token, description string, severity MaintenanceSeverity) bool {
	if c.maintenanceApproved(token, severity) {
		return true
	}
	c.pendingMaintenance = append(c.pendingMaintenance, v1alpha1.PendingMaintenanceAction{
		Description:   description,
		Severity:      severity,
		ApprovalToken: token,
	})
	return false
}

// maintenanceApproved reports whether an action is within the Instance's
// standing tolerance or explicitly approved by token.
func (c *Context) maintenanceApproved(token string, severity MaintenanceSeverity) bool {
	tolerance := MaintenanceNonDisruptive
	approved := ""
	if m := c.in.Spec.Maintenance; m != nil {
		if m.AutoApproveUpTo != "" {
			tolerance = m.AutoApproveUpTo
		}
		approved = m.Approved
	}
	if severityRank(severity) <= severityRank(tolerance) {
		return true
	}
	return token != "" && token == approved
}

// GetPendingMaintenance returns the actions RequestMaintenance held this
// reconcile pass, in request order. Used by the reconciler to flush
// status.pendingMaintenance.
func (c *Context) GetPendingMaintenance() []v1alpha1.PendingMaintenanceAction {
	return c.pendingMaintenance
}
