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

package validation

import (
	"fmt"
	"maps"
	"slices"

	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

const (
	suspendField = "suspend"
	// teardownField only governs what deleting the Instance does, so it may
	// change while the Instance is suspended, e.g. in the update that
	// precedes a delete.
	teardownField = "deletionPolicy"
)

// checkSuspend rejects a suspend request the Provider does not declare
// support for and, on an update, any spec change other than spec.suspend and
// the teardown field while the Instance is suspended or in the update that
// suspends or resumes it. A provider renders the suspended Instance as it
// was, so a change made then could only take effect, unverified, on resume.
func checkSuspend(in, old *v1alpha1.Instance, spec *v1alpha1.ProviderSpec) field.ErrorList {
	specPath := field.NewPath("spec")
	if in.SuspendRequested() && spec.Suspend == nil {
		detail := fmt.Sprintf("provider %s does not support suspending instances", in.Spec.ProviderRef.Name)
		return field.ErrorList{field.Forbidden(specPath.Child(suspendField, "suspended"), detail)}
	}
	if old == nil || (!old.SuspendRequested() && !in.SuspendRequested()) {
		return nil
	}

	detail := "cannot change while the instance is suspended; resume it first"
	switch {
	case !old.SuspendRequested():
		detail = "cannot change in the update that suspends the instance"
	case !in.SuspendRequested():
		detail = "cannot change in the update that resumes the instance"
	}

	changed, err := changedSpecFields(&old.Spec, &in.Spec)
	if err != nil {
		if equality.Semantic.DeepEqual(withoutSuspend(&old.Spec), withoutSuspend(&in.Spec)) {
			return nil
		}
		return field.ErrorList{field.Forbidden(specPath, detail)}
	}
	var errs field.ErrorList
	for _, name := range changed {
		errs = append(errs, field.Forbidden(specPath.Child(name), detail))
	}
	return errs
}

// changedSpecFields returns the sorted top-level spec fields, other than
// spec.suspend and the teardown field, whose values differ between old and
// updated.
func changedSpecFields(old, updated *v1alpha1.InstanceSpec) ([]string, error) {
	before, err := runtime.DefaultUnstructuredConverter.ToUnstructured(old)
	if err != nil {
		return nil, err
	}
	after, err := runtime.DefaultUnstructuredConverter.ToUnstructured(updated)
	if err != nil {
		return nil, err
	}

	names := make(map[string]struct{}, len(before)+len(after))
	for name := range maps.Keys(before) {
		names[name] = struct{}{}
	}
	for name := range maps.Keys(after) {
		names[name] = struct{}{}
	}
	delete(names, suspendField)
	delete(names, teardownField)

	var changed []string
	for _, name := range slices.Sorted(maps.Keys(names)) {
		if !equality.Semantic.DeepEqual(before[name], after[name]) {
			changed = append(changed, name)
		}
	}
	return changed, nil
}

func withoutSuspend(spec *v1alpha1.InstanceSpec) *v1alpha1.InstanceSpec {
	out := spec.DeepCopy()
	out.Suspend = nil
	out.DeletionPolicy = ""
	return out
}
