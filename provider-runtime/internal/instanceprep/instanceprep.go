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

// Package instanceprep holds the transformations the runtime applies to an
// Instance before a provider sees it.
//
// It is internal on purpose. Providers are handed an already-prepared Instance,
// so they have no reason to call this, and keeping it unexported leaves the
// signature free to change as more preparation steps are added.
package instanceprep

import (
	"slices"

	v1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
)

// PrepareForSync returns the Instance a provider is actually handed: a copy of
// in with everything the runtime resolves on the provider's behalf already
// applied. The stored object is never mutated.
//
// Every transformation the runtime performs between admission and Sync belongs
// here, so that what a provider receives has a single definition.
func PrepareForSync(
	spec *v1alpha1.ProviderSpec,
	in *v1alpha1.Instance,
) (*v1alpha1.Instance, string, error) {
	bundleName := controller.EffectiveVersionBundleName(spec, in)

	prepared, err := applyVersionBundle(spec, in, bundleName)
	if err != nil {
		return nil, "", err
	}
	return withSchedulesPausedWhileSuspended(prepared), bundleName, nil
}

// withSchedulesPausedWhileSuspended disables the backup schedules of a
// suspended Instance, so the operator takes no scheduled backup of an engine
// that is not running. The schedules, storages and PITR are otherwise kept as
// they were, and the user's enabled flags apply again once it resumes.
func withSchedulesPausedWhileSuspended(in *v1alpha1.Instance) *v1alpha1.Instance {
	if !in.SuspendRequested() || in.Spec.Backup == nil ||
		!slices.ContainsFunc(in.Spec.Backup.Storages, hasEnabledSchedule) {
		return in
	}
	out := in.DeepCopy()
	for i := range out.Spec.Backup.Storages {
		for j := range out.Spec.Backup.Storages[i].Schedules {
			out.Spec.Backup.Storages[i].Schedules[j].Enabled = false
		}
	}
	return out
}

func hasEnabledSchedule(s v1alpha1.InstanceBackupStorage) bool {
	return slices.ContainsFunc(s.Schedules, func(sch v1alpha1.InstanceBackupSchedule) bool { return sch.Enabled })
}

// applyVersionBundle fills in each component's Version from the named bundle,
// for components the user did not pin explicitly. With no bundle in force the
// Instance is returned untouched, and Sync falls back to the per-type defaults
// in the componentTypes catalog.
func applyVersionBundle(
	spec *v1alpha1.ProviderSpec,
	in *v1alpha1.Instance,
	bundleName string,
) (*v1alpha1.Instance, error) {
	if bundleName == "" {
		return in, nil
	}

	bundle, err := controller.ResolveVersionBundle(spec, bundleName)
	if err != nil {
		return nil, err
	}

	resolved := in.DeepCopy()
	for componentName, bundleVersion := range bundle.Components {
		component, exists := resolved.Spec.Components[componentName]
		if !exists || component.Version != "" {
			continue
		}
		component.Version = bundleVersion
		resolved.Spec.Components[componentName] = component
	}
	return resolved, nil
}
