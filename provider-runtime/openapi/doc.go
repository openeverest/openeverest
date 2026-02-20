
// Package openapi contains OpenAPI schema definitions and utilities for the provider.
// It provides pre-generated OpenAPI schemas for provider custom spec types using kube-openapi.
//
// This package uses the Kubernetes kube-openapi tooling for schema generation:
// - Types are annotated with kubebuilder markers for validation
// - Schemas are generated at build time using openapi-gen
// - The SchemaRegistry serves pre-generated schemas at runtime
//
// Usage:
//
//	import "github.com/openeverest/openeverest/v2/provider-runtime/openapi"
//
//	// Get pre-generated definitions
//	defs := openapi.GetOpenAPIDefinitions(ref)
//
//	// Use SchemaRegistry with pre-generated schemas
//	registry := openapi.NewSchemaRegistry(defs)
package openapi
