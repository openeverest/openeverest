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

// Package manifest provides tooling for generating Provider CR manifests from
// a YAML configuration file and Go type definitions.
//
// Provider developers use this package to generate their provider.yaml manifest.
// The workflow is:
//
//  1. Define Go types in types/types.go (for schemas and type-safe deserialization)
//  2. Write a provider-config.yaml with versions, topology structure, and UI schema
//  3. Add a //go:generate directive to invoke the generate-provider-manifest tool
//  4. Run `go generate ./...` to produce provider.yaml
//
// Example gen.go:
//
//	package root
//
//	//go:generate go tool generate-provider-manifest --types-package=./types
//
// provider-config.yaml contains version, topology, and UI schema configuration.
// The --types-package flag on the CLI specifies which Go packages contain the type
// definitions (can be repeated for multiple packages).
package manifest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"

	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/server"
)

// Config holds the configuration for manifest generation.
type Config struct {
	// ConfigFile is the path to the provider-config.yaml file.
	ConfigFile string

	// OutputFile is the path to write the generated provider.yaml.
	// If empty, output is written to stdout.
	OutputFile string

	// Types maps type names (as referenced in provider-config.yaml) to Go type instances.
	// These are used to generate OpenAPI schemas via reflection.
	// This is the programmatic API — prefer using the generate-manifest CLI tool
	// with RawSchemas instead (which uses static analysis via go/packages).
	// Example: map[string]interface{}{"MongodCustomSpec": types.MongodCustomSpec{}}
	Types map[string]interface{}

	// RawSchemas maps type names to pre-generated OpenAPI JSON schemas.
	// Used by the generate-manifest CLI tool (static analysis via go/packages).
	// When set, takes precedence over Types for schema resolution.
	RawSchemas map[string]json.RawMessage
}

// providerConfig is the internal representation of provider-config.yaml.
type providerConfig struct {
	Name               string                            `json:"name"`
	Namespace          string                            `json:"namespace,omitempty"`
	ComponentTypes     map[string]v1alpha1.ComponentType `json:"componentTypes"`
	Components         map[string]componentConfig        `json:"components"`
	Topologies         map[string]topologyConfig         `json:"topologies"`
	GlobalConfigSchema string                            `json:"globalConfigSchema,omitempty"`
	UISchema           map[string]interface{}            `json:"uiSchema,omitempty"`
}

type componentConfig struct {
	Type             string `json:"type"`
	CustomSpecSchema string `json:"customSpecSchema,omitempty"` // Go type name reference, e.g. "MongodCustomSpec"
}

type topologyConfig struct {
	ConfigSchema string                             `json:"configSchema,omitempty"` // Go type name reference
	Components   map[string]topologyComponentConfig `json:"components"`
}

type topologyComponentConfig struct {
	Optional bool                   `json:"optional,omitempty"`
	Defaults map[string]interface{} `json:"defaults,omitempty"`
}

// Generate reads a provider-config.yaml, resolves Go type references to OpenAPI schemas,
// and writes the complete Provider CR manifest.
func Generate(cfg Config) error {
	// Read and parse the config file
	data, err := os.ReadFile(cfg.ConfigFile)
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %w", cfg.ConfigFile, err)
	}

	var pc providerConfig
	if err := yaml.Unmarshal(data, &pc); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	// Build the ProviderSpec
	spec, err := buildProviderSpec(&pc, cfg.Types, cfg.RawSchemas)
	if err != nil {
		return fmt.Errorf("failed to build provider spec: %w", err)
	}

	// Generate the YAML manifest
	provider := &v1alpha1.Provider{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "core.openeverest.io/v1alpha1",
			Kind:       "Provider",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      pc.Name,
			Namespace: pc.Namespace,
		},
		Spec: *spec,
	}

	output, err := yaml.Marshal(provider)
	if err != nil {
		return fmt.Errorf("failed to marshal provider to YAML: %w", err)
	}

	var buf bytes.Buffer
	buf.WriteString("# Provider CR generated from provider-config.yaml and Go types\n")
	buf.WriteString("# Do not edit manually - regenerate using: go generate ./...\n")
	buf.WriteString("---\n")
	buf.Write(output)

	// Write output
	if cfg.OutputFile == "" {
		_, err = os.Stdout.Write(buf.Bytes())
		return err
	}

	if err := os.WriteFile(cfg.OutputFile, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("failed to write output file %s: %w", cfg.OutputFile, err)
	}
	fmt.Fprintf(os.Stderr, "Generated: %s\n", cfg.OutputFile)
	return nil
}

// MustGenerate calls Generate and exits with an error message on failure.
func MustGenerate(cfg Config) {
	if err := Generate(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// buildProviderSpec constructs a v1alpha1.ProviderSpec from the parsed config and type registry.
func buildProviderSpec(pc *providerConfig, types map[string]interface{}, rawSchemas map[string]json.RawMessage) (*v1alpha1.ProviderSpec, error) {
	spec := &v1alpha1.ProviderSpec{
		ComponentTypes: pc.ComponentTypes,
		Components:     make(map[string]v1alpha1.Component),
		Topologies:     make(map[string]v1alpha1.Topology),
	}

	// Build components with optional schemas
	for name, cc := range pc.Components {
		comp := v1alpha1.Component{Type: cc.Type}
		if cc.CustomSpecSchema != "" {
			schema, err := generateSchemaRaw(cc.CustomSpecSchema, types, rawSchemas)
			if err != nil {
				return nil, fmt.Errorf("component %q customSpecSchema: %w", name, err)
			}
			comp.CustomSpecSchema = schema
		}
		spec.Components[name] = comp
	}

	// Build topologies with optional schemas
	for name, tc := range pc.Topologies {
		topo := v1alpha1.Topology{
			Components: make(map[string]v1alpha1.TopologyComponent),
		}
		if tc.ConfigSchema != "" {
			schema, err := generateSchemaRaw(tc.ConfigSchema, types, rawSchemas)
			if err != nil {
				return nil, fmt.Errorf("topology %q configSchema: %w", name, err)
			}
			topo.ConfigSchema = schema
		}
		for compName, compCfg := range tc.Components {
			topoComp := v1alpha1.TopologyComponent{
				Optional: compCfg.Optional,
			}
			if len(compCfg.Defaults) > 0 {
				defaultsJSON, err := json.Marshal(compCfg.Defaults)
				if err != nil {
					return nil, fmt.Errorf("topology %q component %q defaults: %w", name, compName, err)
				}
				topoComp.Defaults = &runtime.RawExtension{Raw: defaultsJSON}
			}
			topo.Components[compName] = topoComp
		}
		spec.Topologies[name] = topo
	}

	// Build global schema
	if pc.GlobalConfigSchema != "" {
		schema, err := generateSchemaRaw(pc.GlobalConfigSchema, types, rawSchemas)
		if err != nil {
			return nil, fmt.Errorf("globalConfigSchema: %w", err)
		}
		spec.GlobalConfigSchema = schema
	}

	// Build UI schema
	if pc.UISchema != nil {
		uiJSON, err := json.Marshal(pc.UISchema)
		if err != nil {
			return nil, fmt.Errorf("UI schema: %w", err)
		}
		spec.UISchema = &runtime.RawExtension{Raw: uiJSON}
	}

	return spec, nil
}

// generateSchemaRaw resolves a type name to an OpenAPI schema as RawExtension.
// It checks RawSchemas first (pre-generated by CLI tool), then falls back to
// reflection-based generation via Types.
func generateSchemaRaw(typeName string, types map[string]interface{}, rawSchemas map[string]json.RawMessage) (*runtime.RawExtension, error) {
	// Try pre-generated raw schemas first (from CLI tool / go/packages).
	if rawSchemas != nil {
		if raw, ok := rawSchemas[typeName]; ok {
			return &runtime.RawExtension{Raw: raw}, nil
		}
	}

	// Fall back to reflection-based generation.
	if types != nil {
		typ, ok := types[typeName]
		if !ok {
			return nil, fmt.Errorf("type %q not found in type registry or raw schemas", typeName)
		}

		schemaRef, err := server.GenerateSchema(typ)
		if err != nil {
			return nil, fmt.Errorf("failed to generate OpenAPI schema for type %q: %w", typeName, err)
		}

		schemaJSON, err := schemaRef.MarshalJSON()
		if err != nil {
			return nil, fmt.Errorf("failed to marshal schema for type %q: %w", typeName, err)
		}

		return &runtime.RawExtension{Raw: schemaJSON}, nil
	}

	return nil, fmt.Errorf("type %q not found: no raw schemas or type registry provided", typeName)
}
