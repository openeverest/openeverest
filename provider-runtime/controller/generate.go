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

// Provider Manifest Generator
//
// This file provides utilities for generating Provider CR YAML manifests
// from Go code at build time.
//
// Provider developers define their metadata in Go and generate a YAML manifest
// at build time. The manifest is then included in the Helm chart.
//
// Workflow:
//   1. Define metadata in Go (e.g., in psmdb_metadata.go)
//   2. Create a generator tool (e.g., cmd/generate/main.go)
//   3. Run `go generate` or `make generate` to create provider.yaml
//   4. Include provider.yaml in Helm chart
//
// Advantages:
//   - YAML is visible and reviewable in version control
//   - No runtime dependencies - manifest exists before deployment
//   - Easy to inspect and debug
//   - Works with GitOps workflows (manifest is static)

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

// GenerateManifest generates a Provider CR YAML manifest and writes it to a file.
// This is a convenience function for use in go:generate directives or build scripts.
//
// Example usage in a generator tool:
//
//	func main() {
//	    spec := defineProviderSpec()
//	    if err := controller.GenerateManifest(spec, "my-provider", "", "charts/provider.yaml"); err != nil {
//	        log.Fatal(err)
//	    }
//	}
func GenerateManifest(spec *v1alpha1.ProviderSpec, name, namespace, outputPath string) error {
	// Validate spec first
	if err := ValidateProviderSpec(spec); err != nil {
		return fmt.Errorf("invalid provider spec: %w", err)
	}

	// Generate YAML
	yaml, err := ProviderSpecToYAML(spec, name, namespace)
	if err != nil {
		return fmt.Errorf("failed to generate YAML: %w", err)
	}

	// Ensure output directory exists
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write to file
	if err := os.WriteFile(outputPath, []byte(yaml), 0o644); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	return nil
}

// GenerateManifestToStdout generates a Provider CR YAML manifest and writes it to stdout.
// Useful for piping to other tools or quick inspection.
func GenerateManifestToStdout(spec *v1alpha1.ProviderSpec, name, namespace string) error {
	// Validate spec first
	if err := ValidateProviderSpec(spec); err != nil {
		return fmt.Errorf("invalid provider spec: %w", err)
	}

	// Generate YAML
	yaml, err := ProviderSpecToYAML(spec, name, namespace)
	if err != nil {
		return fmt.Errorf("failed to generate YAML: %w", err)
	}

	fmt.Print(yaml)
	return nil
}

// MustGenerateManifest is like GenerateManifest but panics on error.
// Useful for go:generate directives where error handling is awkward.
func MustGenerateManifest(spec *v1alpha1.ProviderSpec, name, namespace, outputPath string) {
	if err := GenerateManifest(spec, name, namespace, outputPath); err != nil {
		panic(err)
	}
}
