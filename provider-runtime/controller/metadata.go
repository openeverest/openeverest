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

// =============================================================================
// VALIDATION HELPERS
// =============================================================================

// ValidateProviderSpec checks that a ProviderSpec is internally consistent.
// It verifies that:
//   - All component types referenced by components exist
//   - All components referenced by topologies exist
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
