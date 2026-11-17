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

// Package classgen renders the Go types of a technology contract into the
// parts of a class manifest a contract owns: the OpenAPI v3 schema of its
// parameters and the CEL validations of the objects that reference the class.
//
// A contract declares its parameters as a Go type with kubebuilder markers.
// [Packages.Schema] renders that type with controller-tools, the parser
// controller-gen uses for CRDs, in-process, so the schema a class publishes is
// the one controller-gen would write for a field of that type. A strict
// decoder of the same type then reads exactly what the schema admits.
//
// Rules a JSON schema cannot express, typically across the parameters and the
// other fields of the referencing object, are XValidation markers on the root
// of a view type: a type with the JSON shape of the referencing object's spec.
// [Packages.View] renders it and [Validations] turns its root rules into
// [Validation]s, each with the reason derived from the field it judges.
//
// Besides the kubebuilder markers, classgen understands
//
//	+openeverest:validation:Closed
//
// on a struct type, which closes the object it renders to
// (additionalProperties: false): a key the type does not declare is invalid,
// as it is for a strict decoder. controller-gen ignores it.
//
// The command in ./cmd prints the schema of a type as YAML, so a contract in
// another repository generates its schemas with
//
//	go run github.com/openeverest/openeverest/v2/pkg/classgen/cmd@<version> \
//		--package . --type Parameters --output zz_generated.parameters.yaml
package classgen
