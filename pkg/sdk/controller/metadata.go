package controller

// Provider Metadata Types
//
// This file defines the types for declaring provider metadata in Go code.
// Provider developers use these types to define their component types, versions,
// and topologies programmatically.
//
// The metadata is generated into a YAML manifest via CLI tooling during the
// build process, then included in the Helm chart.
//
// See docs/PROVIDER_CR_GENERATION.md for the complete workflow.

import (
	"bytes"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	"github.com/openeverest/openeverest/v2/pkg/apis/v1alpha1"
)

// =============================================================================
// PROVIDER METADATA TYPES
// =============================================================================

// ProviderMetadata defines the metadata for a provider.
// This includes component types, components, and topologies that the provider supports.
// Provider developers define this in Go code, and the SDK handles converting it
// to a Provider CR via CLI generation.
type ProviderMetadata struct {
	// ComponentTypes defines the available component types with their versions.
	// Example: "mongod" component type with versions "6.0.19-16", "7.0.18-11", etc.
	ComponentTypes map[string]ComponentTypeMeta `json:"componentTypes,omitempty"`

	// Components defines the logical components that use the component types.
	// Example: "engine", "configServer", "proxy" all use the "mongod" type.
	Components map[string]ComponentMeta `json:"components,omitempty"`

	// Topologies defines the supported deployment topologies.
	// Example: "standard" (single replica set), "sharded" (sharded cluster).
	Topologies map[string]TopologyMeta `json:"topologies,omitempty"`
}

// ComponentTypeMeta defines a component type with its available versions.
type ComponentTypeMeta struct {
	// Versions lists all available versions for this component type.
	Versions []ComponentVersionMeta `json:"versions,omitempty"`
}

// ComponentVersionMeta defines a specific version of a component type.
type ComponentVersionMeta struct {
	// Version is the semantic version string (e.g., "8.0.8-3").
	Version string `json:"version,omitempty"`

	// Image is the container image for this version.
	Image string `json:"image,omitempty"`

	// Default indicates if this is the default version for the component type.
	Default bool `json:"default,omitempty"`
}

// ComponentMeta defines a logical component that uses a component type.
type ComponentMeta struct {
	// Type references a component type defined in ComponentTypes.
	Type string `json:"type,omitempty"`
}

// TopologyMeta defines a deployment topology.
type TopologyMeta struct {
	// Components defines which components are part of this topology.
	Components map[string]TopologyComponentMeta `json:"components,omitempty"`
}

// TopologyComponentMeta defines a component within a topology.
type TopologyComponentMeta struct {
	// Optional indicates if this component is optional in the topology.
	Optional bool `json:"optional,omitempty"`

	// Defaults provides default values for this component in this topology.
	Defaults map[string]interface{} `json:"defaults,omitempty"`
}

// =============================================================================
// CONVERSION TO PROVIDER CR
// =============================================================================

// ToProviderCR converts ProviderMetadata to a Provider custom resource.
// This is used by the CLI tool to generate YAML manifests.
func (m *ProviderMetadata) ToProviderCR(name, namespace string) *v1alpha1.Provider {
	provider := &v1alpha1.Provider{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "everest.percona.com/v1alpha1",
			Kind:       "Provider",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1alpha1.ProviderSpec{
			ComponentTypes: make(map[string]v1alpha1.ComponentType),
			Components:     make(map[string]v1alpha1.Component),
			Topologies:     make(map[string]v1alpha1.Topology),
		},
	}

	// Convert component types
	for typeName, typeMeta := range m.ComponentTypes {
		versions := make([]v1alpha1.ComponentVersion, 0, len(typeMeta.Versions))
		for _, v := range typeMeta.Versions {
			versions = append(versions, v1alpha1.ComponentVersion{
				Version: v.Version,
				Image:   v.Image,
				Default: v.Default,
			})
		}
		provider.Spec.ComponentTypes[typeName] = v1alpha1.ComponentType{
			Versions: versions,
		}
	}

	// Convert components
	for compName, compMeta := range m.Components {
		provider.Spec.Components[compName] = v1alpha1.Component{
			Type: compMeta.Type,
		}
	}

	// Convert topologies
	for topoName, topoMeta := range m.Topologies {
		components := make(map[string]v1alpha1.TopologyComponent)
		for compName, compMeta := range topoMeta.Components {
			components[compName] = v1alpha1.TopologyComponent{
				Optional: compMeta.Optional,
				// Defaults: compMeta.Defaults,
			}
		}
		provider.Spec.Topologies[topoName] = v1alpha1.Topology{
			Components: components,
		}
	}

	return provider
}

// ToYAML converts ProviderMetadata to a YAML manifest string.
// This is used by the CLI tool to generate the Provider CR for Helm packaging.
func (m *ProviderMetadata) ToYAML(name, namespace string) (string, error) {
	provider := m.ToProviderCR(name, namespace)

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

// =============================================================================
// VALIDATION HELPERS
// =============================================================================

// Validate checks that the ProviderMetadata is internally consistent.
// It verifies that:
// - All component types referenced by components exist
// - All components referenced by topologies exist
func (m *ProviderMetadata) Validate() error {
	// Check that component types referenced by components exist
	for compName, comp := range m.Components {
		if _, ok := m.ComponentTypes[comp.Type]; !ok {
			return fmt.Errorf("component %q references unknown component type %q", compName, comp.Type)
		}
	}

	// Check that components referenced by topologies exist
	for topoName, topo := range m.Topologies {
		for compName := range topo.Components {
			if _, ok := m.Components[compName]; !ok {
				return fmt.Errorf("topology %q references unknown component %q", topoName, compName)
			}
		}
	}

	return nil
}

// =============================================================================
// BUILDER HELPERS (Fluent API for defining metadata)
// =============================================================================

// NewProviderMetadata creates a new empty ProviderMetadata.
func NewProviderMetadata() *ProviderMetadata {
	return &ProviderMetadata{
		ComponentTypes: make(map[string]ComponentTypeMeta),
		Components:     make(map[string]ComponentMeta),
		Topologies:     make(map[string]TopologyMeta),
	}
}

// AddComponentType adds a component type with its versions.
func (m *ProviderMetadata) AddComponentType(name string, versions ...ComponentVersionMeta) *ProviderMetadata {
	m.ComponentTypes[name] = ComponentTypeMeta{Versions: versions}
	return m
}

// AddComponent adds a component that uses a component type.
func (m *ProviderMetadata) AddComponent(name, typeName string) *ProviderMetadata {
	m.Components[name] = ComponentMeta{Type: typeName}
	return m
}

// AddTopology adds a topology with its component configuration.
func (m *ProviderMetadata) AddTopology(name string, components map[string]TopologyComponentMeta) *ProviderMetadata {
	m.Topologies[name] = TopologyMeta{Components: components}
	return m
}

// Version creates a ComponentVersionMeta (helper for fluent API).
func Version(version, image string) ComponentVersionMeta {
	return ComponentVersionMeta{Version: version, Image: image}
}

// DefaultVersion creates a default ComponentVersionMeta (helper for fluent API).
func DefaultVersion(version, image string) ComponentVersionMeta {
	return ComponentVersionMeta{Version: version, Image: image, Default: true}
}

// TopologyComponent creates a TopologyComponentMeta (helper for fluent API).
func TopologyComponent(optional bool, defaults map[string]interface{}) TopologyComponentMeta {
	return TopologyComponentMeta{Optional: optional, Defaults: defaults}
}

// RequiredComponent creates a required TopologyComponentMeta with defaults.
func RequiredComponent(defaults map[string]interface{}) TopologyComponentMeta {
	return TopologyComponentMeta{Optional: false, Defaults: defaults}
}

// OptionalComponent creates an optional TopologyComponentMeta.
func OptionalComponent() TopologyComponentMeta {
	return TopologyComponentMeta{Optional: true}
}

// =============================================================================
// METADATA QUERY HELPERS
// =============================================================================
//
// These helpers make it easy to look up version and image information from
// provider metadata during reconciliation.
//
// EXAMPLE USAGE:
//
//	func SyncDatabase(c *Context) error {
//	    metadata := MyProviderMetadata()
//
//	    // Get the engine component from the instance spec
//	    engine := c.Instance().Spec.Components["engine"]
//
//	    // Look up the default image for the component's type
//	    image := metadata.GetDefaultImage(engine.Type)
//
//	    // Or use the convenience method to go directly from component name
//	    image = metadata.GetDefaultImageForComponent("engine")
//
//	    // Apply it to your custom resource
//	    myDB.Spec.Image = image
//	    return c.Apply(myDB)
//	}
//
// TYPICAL PATTERN:
//
//	// User specifies a component with a type (e.g., "mongod")
//	component := c.Instance().Spec.Components["engine"]
//
//	// Determine which image to use
//	var image string
//	if component.Image != "" {
//	    // User explicitly specified an image override
//	    image = component.Image
//	} else {
//	    // Use the default image for this component type
//	    image = metadata.GetDefaultImage(component.Type)
//	}
//

// GetDefaultVersion returns the default ComponentVersionMeta for a given component type.
// Returns nil if the component type doesn't exist or has no default version.
func (m *ProviderMetadata) GetDefaultVersion(componentType string) *ComponentVersionMeta {
	typeMeta, ok := m.ComponentTypes[componentType]
	if !ok {
		return nil
	}

	for _, v := range typeMeta.Versions {
		if v.Default {
			return &v
		}
	}

	return nil
}

// GetDefaultImage returns the default image for a given component type.
// Returns empty string if the component type doesn't exist or has no default version.
func (m *ProviderMetadata) GetDefaultImage(componentType string) string {
	version := m.GetDefaultVersion(componentType)
	if version == nil {
		return ""
	}
	return version.Image
}

// GetComponentType returns the component type name for a given component name.
// For example, if "engine" component uses "mongod" type, GetComponentType("engine") returns "mongod".
// Returns empty string if the component doesn't exist.
func (m *ProviderMetadata) GetComponentType(componentName string) string {
	comp, ok := m.Components[componentName]
	if !ok {
		return ""
	}
	return comp.Type
}

// GetDefaultImageForComponent returns the default image for a given component name.
// This is a convenience method that combines GetComponentType and GetDefaultImage.
// For example, if "engine" uses "mongod" type, and "mongod" has a default version,
// this returns the image for that default version.
// Returns empty string if the component doesn't exist or has no default version.
func (m *ProviderMetadata) GetDefaultImageForComponent(componentName string) string {
	componentType := m.GetComponentType(componentName)
	if componentType == "" {
		return ""
	}
	return m.GetDefaultImage(componentType)
}

// =============================================================================
// CONVERSION FROM SCHEMA PROVIDER
// =============================================================================

// TopologiesFromSchemaProvider converts SchemaProvider topology definitions to metadata topologies.
// This allows you to define topologies once in your SchemaProvider implementation
// and derive the metadata structure from it.
//
// Example:
//
//	func (p *PSMDBProvider) GetMetadata() *ProviderMetadata {
//	    metadata := &ProviderMetadata{
//	        ComponentTypes: ...,
//	        Components: ...,
//	    }
//	    // Derive topologies from SchemaProvider
//	    metadata.Topologies = TopologiesFromSchemaProvider(p.Topologies())
//	    return metadata
//	}
func TopologiesFromSchemaProvider(topologies map[string]TopologyDefinition) map[string]TopologyMeta {
	result := make(map[string]TopologyMeta)
	for topoName, topoDef := range topologies {
		components := make(map[string]TopologyComponentMeta)
		for compName, compDef := range topoDef.Components {
			components[compName] = TopologyComponentMeta{
				Optional: compDef.Optional,
				Defaults: compDef.Defaults,
			}
		}
		result[topoName] = TopologyMeta{
			Components: components,
		}
	}
	return result
}
