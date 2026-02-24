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

// generate-provider-manifest is a CLI tool for generating Provider CR YAML manifests
// from a provider-config.yaml and Go type definitions.
//
// It uses static analysis (go/packages) to inspect Go types and generate
// OpenAPI schemas without requiring reflection or a custom cmd/ file in
// each provider repository.
//
// Usage:
//
//	go tool generate-provider-manifest --types-package=./types [--types-package=./moretypes] [--config=provider-config.yaml] [--output=provider.yaml]
//
// Or via go:generate in any Go file:
//
//	//go:generate go tool generate-provider-manifest --types-package=./types
//
// The --types-package flag can be repeated to load types from multiple packages.
// Types are searched in declaration order across all specified packages.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"sigs.k8s.io/yaml"

	"github.com/openeverest/openeverest/v2/provider-runtime/manifest"
)

// stringsFlag implements flag.Value for a repeatable string flag.
type stringsFlag []string

func (f *stringsFlag) String() string { return strings.Join(*f, ", ") }
func (f *stringsFlag) Set(v string) error {
	*f = append(*f, v)
	return nil
}

func main() {
	var typesPkgs stringsFlag
	flag.Var(&typesPkgs, "types-package", "Go package pattern containing type definitions (repeatable, e.g. ./types)")
	configFile := flag.String("config", "provider-config.yaml", "path to provider-config.yaml")
	outputFile := flag.String("output", "provider.yaml", "output path for generated provider.yaml")
	flag.Parse()

	if len(typesPkgs) == 0 {
		fmt.Fprintln(os.Stderr, "Error: at least one --types-package flag is required (e.g. --types-package=./types)")
		flag.Usage()
		os.Exit(1)
	}

	if err := run(*configFile, *outputFile, typesPkgs); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(configFile, outputFile string, typesPkgs []string) error {
	// Read and parse the config file to discover which type names are referenced.
	data, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %w", configFile, err)
	}

	var rawConfig configForTypeDiscovery
	if err := yaml.Unmarshal(data, &rawConfig); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	// Collect all type name references from the config.
	typeNames := collectTypeNames(&rawConfig)

	// Load the Go packages and generate schemas for all referenced types.
	rawSchemas, err := generateSchemasFromPackages(typesPkgs, typeNames)
	if err != nil {
		return err
	}

	// Delegate to the manifest library.
	return manifest.Generate(manifest.Config{
		ConfigFile: configFile,
		OutputFile: outputFile,
		RawSchemas: rawSchemas,
	})
}

// configForTypeDiscovery is a minimal parse of provider-config.yaml to extract
// all type name references without needing full CRD types.
type configForTypeDiscovery struct {
	Components         map[string]componentSchemaRef `json:"components"`
	Topologies         map[string]topologySchemaRef  `json:"topologies"`
	GlobalConfigSchema string                        `json:"globalConfigSchema"`
}

type componentSchemaRef struct {
	CustomSpecSchema string `json:"customSpecSchema"`
}

type topologySchemaRef struct {
	ConfigSchema string `json:"configSchema"`
}

// collectTypeNames extracts all unique type name references from the parsed config.
func collectTypeNames(cfg *configForTypeDiscovery) []string {
	seen := make(map[string]bool)

	for _, comp := range cfg.Components {
		if comp.CustomSpecSchema != "" {
			seen[comp.CustomSpecSchema] = true
		}
	}
	for _, topo := range cfg.Topologies {
		if topo.ConfigSchema != "" {
			seen[topo.ConfigSchema] = true
		}
	}
	if cfg.GlobalConfigSchema != "" {
		seen[cfg.GlobalConfigSchema] = true
	}

	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	return names
}

// generateSchemasFromPackages loads one or more Go packages and generates OpenAPI
// schemas for the specified type names using static type analysis.
// Types are searched across packages in declaration order; the first match wins.
func generateSchemasFromPackages(pkgPatterns, typeNames []string) (map[string]json.RawMessage, error) {
	pkgs, err := loadPackages(pkgPatterns)
	if err != nil {
		return nil, fmt.Errorf("failed to load types packages: %w", err)
	}

	schemas := make(map[string]json.RawMessage, len(typeNames))
	for _, name := range typeNames {
		schema, err := typeSchemaFromPackages(pkgs, name)
		if err != nil {
			return nil, fmt.Errorf("failed to generate schema for type %q: %w", name, err)
		}
		schemas[name] = schema
	}
	return schemas, nil
}
