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

// Provider upgrade preflight.
//
// This file implements the generic, provider-agnostic checks that decide
// whether upgrading a provider release would strand a running Instance:
//
//   - Catalog membership: is each Instance's effective engine version
//     still present — and not removed — in the target provider's catalog?
//   - Upgrade path: is the release jump itself within the target's
//     declared minUpgradableFrom floor?
//
// The checks run inside the provider chart's Helm pre-upgrade hook, before
// any of the release's resources are applied. Providers with rules a version
// floor cannot capture implement the optional UpgradeProvider interface.
//
// Design background: spec 009 (provider upgrades) in the openeverest/specs
// repository, which labels these two risks R1 and R2.

import (
	"fmt"
	"sort"

	goversion "github.com/hashicorp/go-version"

	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

// UpgradeSeverity classifies an UpgradeIssue.
type UpgradeSeverity string

const (
	// UpgradeError blocks the upgrade: applying it would leave at least one
	// Instance unmanageable.
	UpgradeError UpgradeSeverity = "Error"

	// UpgradeWarning does not block the upgrade but names a remediation the
	// user should perform (e.g. move off a deprecated engine version).
	UpgradeWarning UpgradeSeverity = "Warning"
)

// Reasons reported on UpgradeIssue.
const (
	// UpgradeReasonVersionRemoved indicates an engine version an Instance
	// runs is absent from the target catalog or removed in the target release.
	UpgradeReasonVersionRemoved = "VersionRemoved"

	// UpgradeReasonVersionDeprecated indicates an engine version an Instance
	// runs is deprecated and scheduled for removal.
	UpgradeReasonVersionDeprecated = "VersionDeprecated"

	// UpgradeReasonComponentUnsupported indicates a component an Instance
	// uses is no longer defined by the target provider.
	UpgradeReasonComponentUnsupported = "ComponentUnsupported"

	// UpgradeReasonPathBlocked indicates the installed release is below the
	// target's minUpgradableFrom floor.
	UpgradeReasonPathBlocked = "UpgradePathBlocked"

	// UpgradeReasonPathUnverified indicates the upgrade path could not be
	// validated (e.g. the installed provider does not declare its release
	// version).
	UpgradeReasonPathUnverified = "UpgradePathUnverified"
)

// UpgradeIssue is a single finding produced by the upgrade preflight.
type UpgradeIssue struct {
	Severity     UpgradeSeverity
	InstanceName string
	Namespace    string
	Component    string
	Reason       string
	// Message is human-readable and must not leak operator internals.
	Message string
}

// HasBlockingIssues reports whether any issue is an UpgradeError.
func HasBlockingIssues(issues []UpgradeIssue) bool {
	for _, issue := range issues {
		if issue.Severity == UpgradeError {
			return true
		}
	}
	return false
}

// RunUpgradePreflight runs the full generic preflight — the upgrade-path
// floor check and the catalog-membership check — and appends any
// provider-specific issues from the optional UpgradeProvider interface.
//
// current is the spec of the installed Provider CR (nil when none is
// installed); target is the spec carried by the new chart. c may be nil when
// the provider does not implement UpgradeProvider.
func RunUpgradePreflight(c *Context, provider ProviderInterface, current, target *v1alpha1.ProviderSpec, instances []v1alpha1.Instance) []UpgradeIssue {
	issues := CheckUpgradePath(current, target)
	issues = append(issues, PreflightUpgrade(target, instances)...)
	if up, ok := provider.(UpgradeProvider); ok {
		issues = append(issues, up.CheckUpgrade(c, target, instances)...)
	}
	return issues
}

// CheckUpgradePath validates the release jump against the target's
// minUpgradableFrom floor. It returns nothing when the target declares
// no floor, an UpgradeError when the installed release is below the floor,
// and an UpgradeWarning when the path cannot be verified.
func CheckUpgradePath(current, target *v1alpha1.ProviderSpec) []UpgradeIssue {
	if target == nil || target.Release == nil || target.Release.MinUpgradableFrom == "" {
		return nil
	}
	floorStr := target.Release.MinUpgradableFrom

	currentStr := ""
	if current != nil && current.Release != nil {
		currentStr = current.Release.Version
	}
	if currentStr == "" {
		return []UpgradeIssue{{
			Severity: UpgradeWarning,
			Reason:   UpgradeReasonPathUnverified,
			Message: fmt.Sprintf(
				"the installed provider does not declare release.version; cannot verify the upgrade path (the target release requires at least %s)",
				floorStr),
		}}
	}

	currentVer, err := goversion.NewVersion(currentStr)
	if err != nil {
		return []UpgradeIssue{pathUnparseableIssue("installed", currentStr)}
	}
	floorVer, err := goversion.NewVersion(floorStr)
	if err != nil {
		return []UpgradeIssue{pathUnparseableIssue("target minUpgradableFrom", floorStr)}
	}

	if currentVer.LessThan(floorVer) {
		targetStr := target.Release.Version
		return []UpgradeIssue{{
			Severity: UpgradeError,
			Reason:   UpgradeReasonPathBlocked,
			Message: fmt.Sprintf(
				"provider %s cannot be upgraded directly to %s: the target release requires at least %s; step through the intermediate provider release(s) first",
				currentStr, targetStr, floorStr),
		}}
	}
	return nil
}

func pathUnparseableIssue(what, value string) UpgradeIssue {
	return UpgradeIssue{
		Severity: UpgradeWarning,
		Reason:   UpgradeReasonPathUnverified,
		Message:  fmt.Sprintf("cannot verify the upgrade path: %s release version %q is not a valid version", what, value),
	}
}

// PreflightUpgrade runs the generic catalog-membership and deprecation check
// for every Instance against the target provider spec.
//
// For each Instance the effective engine version of every component is
// resolved exactly as the reconciler resolves it: the component's explicit
// spec version, else the frozen bundle (spec.version → status.version →
// target default). A version absent from the target catalog — or flagged as
// removed in the target release — is an UpgradeError; a deprecated-but-present
// version is an UpgradeWarning with a remediation runway.
//
// Instances already being deleted are skipped.
func PreflightUpgrade(target *v1alpha1.ProviderSpec, instances []v1alpha1.Instance) []UpgradeIssue {
	var issues []UpgradeIssue
	for i := range instances {
		in := &instances[i]
		if in.DeletionTimestamp != nil {
			continue
		}
		issues = append(issues, preflightInstance(target, in)...)
	}
	return issues
}

func preflightInstance(target *v1alpha1.ProviderSpec, in *v1alpha1.Instance) []UpgradeIssue {
	var issues []UpgradeIssue

	// Resolve the effective bundle the same way the reconciler does:
	// spec.version → status.version → target default.
	bundleName := in.Spec.Version
	if bundleName == "" {
		bundleName = in.Status.Version
	}
	if bundleName == "" {
		bundleName = GetDefaultVersionBundleName(target)
	}

	var bundle *v1alpha1.VersionBundle
	if bundleName != "" {
		var err error
		bundle, err = ResolveVersionBundle(target, bundleName)
		if err != nil {
			issues = append(issues, UpgradeIssue{
				Severity:     UpgradeError,
				InstanceName: in.Name,
				Namespace:    in.Namespace,
				Reason:       UpgradeReasonVersionRemoved,
				Message: fmt.Sprintf(
					"version %q used by this Instance is not present in the target provider; upgrade this database to a supported version before upgrading the provider",
					bundleName),
			})
		}
	}

	// Iterate components in a stable order so reports are deterministic.
	compNames := make([]string, 0, len(in.Spec.Components))
	for name := range in.Spec.Components {
		compNames = append(compNames, name)
	}
	sort.Strings(compNames)

	for _, compName := range compNames {
		effective := in.Spec.Components[compName].Version
		if effective == "" && bundle != nil {
			effective = bundle.Components[compName]
		}
		if effective == "" {
			// Falls back to the target's per-type default, which exists by
			// construction.
			continue
		}
		if issue := preflightComponent(target, in, compName, effective); issue != nil {
			issues = append(issues, *issue)
		}
	}
	return issues
}

func preflightComponent(target *v1alpha1.ProviderSpec, in *v1alpha1.Instance, compName, effective string) *UpgradeIssue {
	issue := &UpgradeIssue{
		InstanceName: in.Name,
		Namespace:    in.Namespace,
		Component:    compName,
	}

	comp, ok := target.Components[compName]
	if !ok {
		issue.Severity = UpgradeError
		issue.Reason = UpgradeReasonComponentUnsupported
		issue.Message = fmt.Sprintf("component %q is not defined by the target provider", compName)
		return issue
	}
	componentType, ok := target.ComponentTypes[comp.Type]
	if !ok {
		issue.Severity = UpgradeError
		issue.Reason = UpgradeReasonComponentUnsupported
		issue.Message = fmt.Sprintf("component type %q is not defined by the target provider", comp.Type)
		return issue
	}

	entry := findComponentVersion(componentType, effective)
	if entry == nil {
		issue.Severity = UpgradeError
		issue.Reason = UpgradeReasonVersionRemoved
		issue.Message = fmt.Sprintf(
			"%s %s is not present in the target provider catalog; upgrade this database to a supported version before upgrading the provider",
			comp.Type, effective)
		return issue
	}

	if entry.RemovedInVersion != "" && removedAtOrBeforeTarget(entry.RemovedInVersion, target) {
		issue.Severity = UpgradeError
		issue.Reason = UpgradeReasonVersionRemoved
		issue.Message = fmt.Sprintf(
			"%s %s is removed in provider %s; upgrade this database to a supported version before upgrading the provider",
			comp.Type, effective, entry.RemovedInVersion)
		return issue
	}

	if entry.Deprecated || entry.RemovedInVersion != "" {
		issue.Severity = UpgradeWarning
		issue.Reason = UpgradeReasonVersionDeprecated
		if entry.RemovedInVersion != "" {
			issue.Message = fmt.Sprintf(
				"%s %s is deprecated and is removed in provider %s; upgrade this database to a supported version before that provider upgrade",
				comp.Type, effective, entry.RemovedInVersion)
		} else {
			issue.Message = fmt.Sprintf(
				"%s %s is deprecated; plan an upgrade to a supported version",
				comp.Type, effective)
		}
		return issue
	}

	return nil
}

// findComponentVersion returns the catalog entry for version, or nil when the
// component type does not list it.
func findComponentVersion(componentType v1alpha1.ComponentType, version string) *v1alpha1.ComponentVersion {
	for i := range componentType.Versions {
		if componentType.Versions[i].Version == version {
			return &componentType.Versions[i]
		}
	}
	return nil
}

// removedAtOrBeforeTarget reports whether removedIn is at or below the target
// release version, i.e. the target release has already dropped the entry.
// Unparseable versions — a provider definition bug — degrade to false so the
// entry is reported as a deprecation warning instead of a hard block.
func removedAtOrBeforeTarget(removedIn string, target *v1alpha1.ProviderSpec) bool {
	if target.Release == nil || target.Release.Version == "" {
		return false
	}
	removedVer, err := goversion.NewVersion(removedIn)
	if err != nil {
		return false
	}
	targetVer, err := goversion.NewVersion(target.Release.Version)
	if err != nil {
		return false
	}
	return removedVer.LessThanOrEqual(targetVer)
}
