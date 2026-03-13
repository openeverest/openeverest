// everest
// Copyright (C) 2023 Percona LLC
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

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/yaml"
)

func TestExtractSchemas(t *testing.T) {
	tmpDir := t.TempDir()

	crdContent := `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: testresources.test.percona.com
spec:
  group: test.percona.com
  names:
    kind: TestResource
    listKind: TestResourceList
    plural: testresources
    singular: testresource
  versions:
  - name: v1alpha1
    schema:
      openAPIV3Schema:
        description: TestResource is a test CRD
        type: object
        properties:
          apiVersion:
            type: string
          kind:
            type: string
          spec:
            type: object
            properties:
              field1:
                type: string
              field2:
                type: integer
`

	crdFile := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(crdFile, []byte(crdContent), 0o644)
	assert.NoError(t, err, "Failed to write test CRD file")

	schemas := make(map[string]*openapi3.SchemaRef)
	err = extractSchemas(crdFile, schemas)
	assert.NoError(t, err, "extractSchemas failed")

	assert.Contains(t, schemas, "TestResource")
	assert.Contains(t, schemas, "TestResourceList")

	resource := schemas["TestResource"]
	assert.NotNil(t, resource)
	assert.NotNil(t, resource.Value)

	assert.Equal(t, "object", resource.Value.Type.Slice()[0], "Expected type=object")

	assert.NotNil(t, resource.Value.Properties)
	assert.Contains(t, resource.Value.Properties, "spec")
}

func TestRun(t *testing.T) {
	tmpDir := t.TempDir()
	crdDir := filepath.Join(tmpDir, "crds")
	err := os.MkdirAll(crdDir, 0o755)
	assert.NoError(t, err, "Failed to create CRD directory")

	crdContent := `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: samples.test.percona.com
spec:
  group: test.percona.com
  names:
    kind: Sample
    listKind: SampleList
    plural: samples
    singular: sample
  versions:
  - name: v1
    schema:
      openAPIV3Schema:
        type: object
        properties:
          spec:
            type: object
            properties:
              name:
                type: string
`

	crdFile := filepath.Join(crdDir, "sample.yaml")
	err = os.WriteFile(crdFile, []byte(crdContent), 0o644)
	assert.NoError(t, err, "Failed to write test CRD")

	outputFile := filepath.Join(tmpDir, "output.yml")
	err = run(crdDir, outputFile, nil)
	assert.NoError(t, err, "run() failed")

	_, err = os.Stat(outputFile)
	assert.NoError(t, err, "Output file was not created")

	data, err := os.ReadFile(outputFile)
	assert.NoError(t, err, "Failed to read output file")

	var spec openapi3.T
	err = yaml.Unmarshal(data, &spec)
	assert.NoError(t, err, "Failed to unmarshal output")

	assert.Equal(t, "3.0.2", spec.OpenAPI)

	assert.NotNil(t, spec.Components)
	assert.NotNil(t, spec.Components.Schemas)

	assert.Contains(t, spec.Components.Schemas, "Sample")
	assert.Contains(t, spec.Components.Schemas, "SampleList")
}

func TestExtractGoTypeSchemas(t *testing.T) {
	testFile := filepath.Join("testdata", "types.go")

	schemas := make(map[string]*openapi3.SchemaRef)
	err := extractGoTypeSchemas(testFile, schemas)
	assert.NoError(t, err, "extractGoTypeSchemas failed")

	// Should have extracted User and Config schemas
	assert.Contains(t, schemas, "User", "User schema should be extracted")
	assert.Contains(t, schemas, "Config", "Config schema should be extracted")

	// Validate User schema
	userSchema := schemas["User"]
	assert.NotNil(t, userSchema)
	assert.NotNil(t, userSchema.Value)
	assert.Equal(t, "object", userSchema.Value.Type.Slice()[0], "User should be an object type")
	assert.Equal(t, "User represents a user in the system.", userSchema.Value.Description)

	// Check User properties
	assert.NotNil(t, userSchema.Value.Properties)
	assert.Contains(t, userSchema.Value.Properties, "username")
	assert.Contains(t, userSchema.Value.Properties, "email")
	assert.Contains(t, userSchema.Value.Properties, "age")
	assert.Contains(t, userSchema.Value.Properties, "isAdmin")

	usernameField := userSchema.Value.Properties["username"]
	assert.Equal(t, "string", usernameField.Value.Type.Slice()[0])
	assert.Equal(t, "Username is the unique identifier for the user.", usernameField.Value.Description)

	ageField := userSchema.Value.Properties["age"]
	assert.Equal(t, "integer", ageField.Value.Type.Slice()[0])
	assert.Equal(t, "Age is the user's age in years.", ageField.Value.Description)

	isAdminField := userSchema.Value.Properties["isAdmin"]
	assert.Equal(t, "boolean", isAdminField.Value.Type.Slice()[0])
	assert.Equal(t, "IsAdmin indicates if the user has admin privileges.", isAdminField.Value.Description)

	// Validate Config schema
	configSchema := schemas["Config"]
	assert.NotNil(t, configSchema)
	assert.NotNil(t, configSchema.Value)
	assert.Equal(t, "object", configSchema.Value.Type.Slice()[0], "Config should be an object type")
	assert.Equal(t, "Config is the configuration object.", configSchema.Value.Description)

	// Check Config properties
	assert.NotNil(t, configSchema.Value.Properties)
	assert.Contains(t, configSchema.Value.Properties, "timeout")
	assert.Contains(t, configSchema.Value.Properties, "retries")
	assert.Contains(t, configSchema.Value.Properties, "metadata")

	// Tags field with json:"-" on a map should become additionalProperties
	assert.NotNil(t, configSchema.Value.AdditionalProperties.Schema.Value)
	assert.Equal(t, "string", configSchema.Value.AdditionalProperties.Schema.Value.Type.Slice()[0])

	timeoutField := configSchema.Value.Properties["timeout"]
	assert.Equal(t, "integer", timeoutField.Value.Type.Slice()[0])
	assert.Equal(t, "Timeout specifies the timeout duration in seconds.", timeoutField.Value.Description)

	retriesField := configSchema.Value.Properties["retries"]
	assert.Equal(t, "integer", retriesField.Value.Type.Slice()[0])
	assert.Equal(t, "Retries is the number of retry attempts.", retriesField.Value.Description)

	metadataField := configSchema.Value.Properties["metadata"]
	assert.Equal(t, "string", metadataField.Value.Type.Slice()[0])
	assert.Equal(t, "Metadata stores additional configuration data.", metadataField.Value.Description)
}
