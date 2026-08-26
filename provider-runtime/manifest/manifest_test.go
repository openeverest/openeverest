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

package manifest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
)

// buildSpecFromYAML runs the same path Generate does, minus reading the file.
func buildSpecFromYAML(t *testing.T, config string) *providerConfig {
	t.Helper()

	var pc providerConfig
	require.NoError(t, yaml.Unmarshal([]byte(config), &pc))
	return &pc
}

func TestSupportedFieldsReachesTheProviderSpec(t *testing.T) {
	t.Parallel()

	pc := buildSpecFromYAML(t, `
name: test-provider
componentTypes: {}
components:
  engine:
    type: mongod
topologies:
  sharded:
    components:
      engine:
        supportedFields:
          required: [storage]
          properties:
            storage: {}
            replicas: {minimum: 1}
            schedulingPolicy:
              properties:
                nodeSelector: {}
                tolerations: {}
      backupAgent:
        optional: true
`)

	spec, err := buildProviderSpec(pc, nil, nil)
	require.NoError(t, err)

	engine := spec.Topologies["sharded"].Components["engine"]
	require.NotNil(t, engine.SupportedFields)
	schema := engine.SupportedFields.OpenAPIV3Schema
	require.NotNil(t, schema)

	assert.Equal(t, []string{"storage"}, schema.Required)
	assert.Contains(t, schema.Properties, "storage")
	assert.NotContains(t, schema.Properties, "resources", "an undeclared field must not appear")

	// A bound survives the round trip through the wire format.
	replicas := schema.Properties["replicas"]
	require.NotNil(t, replicas.Minimum)
	assert.InDelta(t, 1.0, *replicas.Minimum, 0)

	// Nesting survives, so a component can honour part of a grouped field.
	scheduling := schema.Properties["schedulingPolicy"]
	assert.Contains(t, scheduling.Properties, "nodeSelector")
	assert.Contains(t, scheduling.Properties, "tolerations")
	assert.NotContains(t, scheduling.Properties, "affinity")
}

func TestComponentWithoutSupportedFieldsIsUnconstrained(t *testing.T) {
	t.Parallel()

	pc := buildSpecFromYAML(t, `
name: test-provider
componentTypes: {}
components:
  engine:
    type: mongod
topologies:
  replicaSet:
    components:
      engine:
        optional: true
`)

	spec, err := buildProviderSpec(pc, nil, nil)
	require.NoError(t, err)

	engine := spec.Topologies["replicaSet"].Components["engine"]
	assert.True(t, engine.Optional)
	assert.Nil(t, engine.SupportedFields)
}
