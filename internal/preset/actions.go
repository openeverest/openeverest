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

package preset

import (
	"context"
	"fmt"

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

// DefaultResolver looks up the default resource name for a reference in a
// namespace. Cluster-scoped kinds (StorageClass) ignore the namespace argument.
type DefaultResolver interface {
	ResolveDefault(ctx context.Context, namespace string, kind ResourceKind, component string) (string, error)
}

// EnsureNamespaceRefsEmpty returns an error if any namespace-scoped reference in the spec
// holds a value.
func EnsureNamespaceRefsEmpty(spec *corev1alpha1.InstanceSpec) error {
	return WalkSpec(spec, func(ref FieldRef) error {
		if ref.Scope() != ScopeNamespace || ref.IsEmpty() {
			return nil
		}

		return fmt.Errorf("component %q: %s must be empty in preset", ref.Component(), ref.Path())
	})
}

// ClearNamespaceRefs empties every namespace-scoped reference in the spec. It is used when
// turning an Instance into an InstancePreset.
func ClearNamespaceRefs(spec *corev1alpha1.InstanceSpec) error {
	return WalkSpec(spec, func(ref FieldRef) error {
		if ref.Scope() == ScopeNamespace {
			ref.Set("")
		}

		return nil
	})
}

// ResolveNamespaceRefs fills every empty reference in the spec with the default resolved
// for the namespace. References that already hold a value are left unchanged.
func ResolveNamespaceRefs(ctx context.Context, spec *corev1alpha1.InstanceSpec, namespace string, resolver DefaultResolver) error {
	return WalkSpec(spec, func(ref FieldRef) error {
		if !ref.IsEmpty() {
			return nil
		}

		name, err := resolver.ResolveDefault(ctx, namespace, ref.Kind(), ref.Component())
		if err != nil {
			return err
		}

		ref.Set(name)

		return nil
	})
}
