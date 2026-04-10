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

// Provider Spec Helpers
//
// This file provides helper functions for working with the Provider CR spec
// (v1alpha1.ProviderSpec). These helpers make it easy to look up version and
// image information from the Provider spec during reconciliation.
//
// The Provider spec is fetched dynamically from the controller-runtime cache
// via Context.ProviderSpec(), ensuring it is always up-to-date without
// hitting the Kubernetes API server.
//
// EXAMPLE USAGE:
//
//	func SyncDatabase(c *controller.Context) error {
//	    spec, err := c.ProviderSpec()
//	    if err != nil {
//	        return err
//	    }
//
//	    // Get the default image for a component type
//	    image := controller.GetDefaultImage(spec, "mongod")
//
//	    // Or go from component name to default image
//	    image = controller.GetDefaultImageForComponent(spec, "engine")
//
//	    // Apply it to your custom resource
//	    myDB.Spec.Image = image
//	    return c.Apply(myDB)
//	}

import (
	"bytes"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

// =============================================================================
// PROVIDER SPEC QUERY HELPERS
// =============================================================================

// GetDefaultVersion returns the default ComponentVersion for a given component type.
// Returns nil if the component type doesn't exist or has no default version.
func GetDefaultVersion(spec *v1alpha1.ProviderSpec, componentType string) *v1alpha1.ComponentVersion {
	ct, ok := spec.ComponentTypes[componentType]
	if !ok {
		return nil
	}

	for i, v := range ct.Versions {
		if v.Default {
			return &ct.Versions[i]
		}
	}

	return nil
}

// GetDefaultImage returns the default image for a given component type.
// Returns empty string if the component type doesn't exist or has no default version.
func GetDefaultImage(spec *v1alpha1.ProviderSpec, componentType string) string {
	version := GetDefaultVersion(spec, componentType)
	if version == nil {
		return ""
	}
	return version.Image
}

// GetComponentType returns the component type name for a given component name.
// For example, if "engine" component uses "mongod" type, GetComponentType(spec, "engine") returns "mongod".
// Returns empty string if the component doesn't exist.
func GetComponentType(spec *v1alpha1.ProviderSpec, componentName string) string {
	comp, ok := spec.Components[componentName]
	if !ok {
		return ""
	}
	return comp.Type
}

// GetDefaultImageForComponent returns the default image for a given component name.
// This is a convenience function that combines GetComponentType and GetDefaultImage.
// For example, if "engine" uses "mongod" type, and "mongod" has a default version,
// this returns the image for that default version.
// Returns empty string if the component doesn't exist or has no default version.
func GetDefaultImageForComponent(spec *v1alpha1.ProviderSpec, componentName string) string {
	componentType := GetComponentType(spec, componentName)
	if componentType == "" {
		return ""
	}
	return GetDefaultImage(spec, componentType)
}

// GetImageForVersion returns the container image for a specific component name
// and explicit version string. Typical use is inside Sync() after the
// provider-runtime has resolved the version bundle into ComponentSpec.Version.
// Returns empty string if the component, its type, or the requested version
// cannot be found.
func GetImageForVersion(spec *v1alpha1.ProviderSpec, componentName, version string) string {
	componentType := GetComponentType(spec, componentName)
	if componentType == "" {
		return ""
	}
	ct, ok := spec.ComponentTypes[componentType]
	if !ok {
		return ""
	}
	for _, v := range ct.Versions {
		if v.Version == version {
			return v.Image
		}
	}
	return ""
}

// =============================================================================
// VERSION BUNDLE HELPERS
// =============================================================================

// ResolveVersionBundle looks up the named version bundle in the ProviderSpec.
// Returns an error if the bundle name is not found.
func ResolveVersionBundle(spec *v1alpha1.ProviderSpec, version string) (*v1alpha1.VersionBundle, error) {
	for i := range spec.Versions {
		if spec.Versions[i].Name == version {
			return &spec.Versions[i], nil
		}
	}
	return nil, fmt.Errorf("version bundle %q not found in provider spec", version)
}

// GetDefaultVersionBundle returns the bundle marked as Default: true.
// Returns nil if no bundles are defined or none is marked as default.
func GetDefaultVersionBundle(spec *v1alpha1.ProviderSpec) *v1alpha1.VersionBundle {
	for i := range spec.Versions {
		if spec.Versions[i].Default {
			return &spec.Versions[i]
		}
	}
	return nil
}

// GetDefaultVersionBundleName returns the name of the bundle marked as
// Default: true. Returns empty string if no default bundle is defined.
func GetDefaultVersionBundleName(spec *v1alpha1.ProviderSpec) string {
	for _, b := range spec.Versions {
		if b.Default {
			return b.Name
		}
	}
	return ""
}

// =============================================================================
// VALIDATION HELPERS
// =============================================================================

// ValidateProviderSpec checks that a ProviderSpec is internally consistent.
// It verifies that:
//   - All component types referenced by components exist
//   - All components referenced by topologies exist
//   - All component versions referenced in version bundles exist in the catalog
func ValidateProviderSpec(spec *v1alpha1.ProviderSpec) error {
	// Check that component types referenced by components exist
	for compName, comp := range spec.Components {
		if _, ok := spec.ComponentTypes[comp.Type]; !ok {
			return fmt.Errorf("component %q references unknown component type %q", compName, comp.Type)
		}
	}

	// Check that components referenced by topologies exist
	for topoName, topo := range spec.Topologies {
		for compName := range topo.Components {
			if _, ok := spec.Components[compName]; !ok {
				return fmt.Errorf("topology %q references unknown component %q", topoName, compName)
			}
		}
	}

	// Check that version bundles only reference known components and valid versions
	for _, bundle := range spec.Versions {
		for compName, ver := range bundle.Components {
			comp, ok := spec.Components[compName]
			if !ok {
				return fmt.Errorf("version bundle %q: component %q is not defined", bundle.Name, compName)
			}
			ct, ok := spec.ComponentTypes[comp.Type]
			if !ok {
				return fmt.Errorf("version bundle %q: component %q has unknown type %q", bundle.Name, compName, comp.Type)
			}
			found := false
			for _, v := range ct.Versions {
				if v.Version == ver {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("version bundle %q: component %q version %q not found in componentTypes[%q]", bundle.Name, compName, ver, comp.Type)
			}
		}
	}

	return nil
}

// =============================================================================
// YAML GENERATION
// =============================================================================

// ProviderSpecToYAML converts a ProviderSpec to a YAML manifest string for a Provider CR.
// This is used by CLI tooling to generate the Provider CR for Helm packaging.
func ProviderSpecToYAML(spec *v1alpha1.ProviderSpec, name, namespace string) (string, error) {
	provider := &v1alpha1.Provider{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "core.openeverest.io/v1alpha1",
			Kind:       "Provider",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: *spec,
	}

	data, err := yaml.Marshal(provider)
	if err != nil {
		return "", fmt.Errorf("failed to marshal provider to YAML: %w", err)
	}

	// Add a header comment
	var buf bytes.Buffer
	buf.WriteString("# Provider CR generated from Go code\n")
	buf.WriteString("# Do not edit manually - regenerate using: provider-sdk generate-manifest\n")
	buf.WriteString("---\n")
	buf.Write(data)

	return buf.String(), nil
}
