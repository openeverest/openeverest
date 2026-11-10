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

package s3

import (
	_ "embed"
	"errors"
	"fmt"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/yaml"
)

// manifest is the s3 BackupStorageClass, generated from the types of this
// package by make gen.
//
//go:embed class.yaml
var manifest []byte

// Manifest returns the YAML manifest of the s3 BackupStorageClass.
func Manifest() []byte {
	return append([]byte(nil), manifest...)
}

// ParametersSchema returns the parametersSchema of the s3 class, the OpenAPI
// v3 schema of Parameters. Executors that support a subset of it publish the
// narrowed schema on their BackupClass.
func ParametersSchema() *apiextensionsv1.JSONSchemaProps {
	var class struct {
		Spec struct {
			ParametersSchema struct {
				OpenAPIV3Schema *apiextensionsv1.JSONSchemaProps `json:"openAPIV3Schema"` //nolint:tagliatelle
			} `json:"parametersSchema"`
		} `json:"spec"`
	}
	err := yaml.Unmarshal(manifest, &class)
	if err == nil && class.Spec.ParametersSchema.OpenAPIV3Schema == nil {
		err = errors.New("no spec.parametersSchema.openAPIV3Schema")
	}
	if err != nil {
		panic(fmt.Sprintf("the embedded s3 class manifest is invalid, run make gen: %v", err))
	}
	return class.Spec.ParametersSchema.OpenAPIV3Schema
}
