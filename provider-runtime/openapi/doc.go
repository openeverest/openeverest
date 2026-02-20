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
