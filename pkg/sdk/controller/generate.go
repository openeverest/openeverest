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
//
// See docs/PROVIDER_CR_GENERATION.md for complete documentation.

import (
	"fmt"
	"os"
	"path/filepath"
)

// GenerateManifest generates a Provider CR YAML manifest and writes it to a file.
// This is a convenience function for use in go:generate directives or build scripts.
//
// Example usage in a generator tool:
//
//	func main() {
//	    metadata := defineMetadata()
//	    if err := sdk.GenerateManifest(metadata, "my-provider", "", "charts/provider.yaml"); err != nil {
//	        log.Fatal(err)
//	    }
//	}
func GenerateManifest(metadata *ProviderMetadata, name, namespace, outputPath string) error {
	// Validate metadata first
	if err := metadata.Validate(); err != nil {
		return fmt.Errorf("invalid metadata: %w", err)
	}

	// Generate YAML
	yaml, err := metadata.ToYAML(name, namespace)
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
func GenerateManifestToStdout(metadata *ProviderMetadata, name, namespace string) error {
	// Validate metadata first
	if err := metadata.Validate(); err != nil {
		return fmt.Errorf("invalid metadata: %w", err)
	}

	// Generate YAML
	yaml, err := metadata.ToYAML(name, namespace)
	if err != nil {
		return fmt.Errorf("failed to generate YAML: %w", err)
	}

	fmt.Print(yaml)
	return nil
}

// MustGenerateManifest is like GenerateManifest but panics on error.
// Useful for go:generate directives where error handling is awkward.
func MustGenerateManifest(metadata *ProviderMetadata, name, namespace, outputPath string) {
	if err := GenerateManifest(metadata, name, namespace, outputPath); err != nil {
		panic(err)
	}
}
