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

package validation

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/yaml"

	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
)

const testProviderSpec = `
componentTypes:
  mongod:
    versions:
    - {version: "7.0.1", image: "mongod:7.0.1"}
    - {version: "6.0.1", image: "mongod:6.0.1", deprecated: true, removedInVersion: "0.9"}
components:
  engine:
    type: mongod
    parametersSchema:
      openAPIV3Schema:
        type: object
        properties:
          configuration: {type: string}
          cacheSizeGB: {type: integer, minimum: 1}
  proxy:
    type: mongod
  configServer:
    type: mongod
versions:
- name: "7.0"
  components: {engine: "7.0.1"}
topologies:
  replicaSet:
    components:
      engine:
        supportedFields:
          openAPIV3Schema:
            required: [storage]
            properties:
              version: {}
              storage: {}
              replicas:
                minimum: 1
                x-kubernetes-validations:
                - {rule: "self % 2 == 1", message: "must be odd"}
              schedulingPolicy:
                properties:
                  affinity: {}
  sharded:
    parametersSchema:
      openAPIV3Schema:
        type: object
        properties:
          numShards: {type: integer, minimum: 1}
    components:
      engine: {}
      proxy: {}
      configServer: {}
`

func testProvider(t *testing.T) *v1alpha1.Provider {
	t.Helper()
	provider := &v1alpha1.Provider{}
	require.NoError(t, yaml.Unmarshal([]byte(testProviderSpec), &provider.Spec))
	return provider
}

func raw(json string) *runtime.RawExtension {
	return &runtime.RawExtension{Raw: []byte(json)}
}

func storage() *v1alpha1.Storage {
	return &v1alpha1.Storage{Size: resource.MustParse("1Gi")}
}

func replicaSet(mutate func(*v1alpha1.Instance)) *v1alpha1.Instance {
	in := &v1alpha1.Instance{Spec: v1alpha1.InstanceSpec{
		Topology: &v1alpha1.TopologySpec{Type: "replicaSet"},
		Components: map[string]v1alpha1.ComponentSpec{
			"engine": {Replicas: new(int32(3)), Storage: storage()},
		},
	}}
	if mutate != nil {
		mutate(in)
	}
	return in
}

// recordingValidator returns fixed errors and records whether it was called.
type recordingValidator struct {
	errs   field.ErrorList
	called bool
}

func (v *recordingValidator) Validate(controller.ValidationInput) controller.ValidationResult {
	v.called = true
	return controller.ValidationResult{Errors: v.errs, Warnings: []string{"from the provider"}}
}

func errorStrings(errs field.ErrorList) []string {
	if len(errs) == 0 {
		return nil
	}
	out := make([]string, 0, len(errs))
	for _, err := range errs {
		out = append(out, err.Error())
	}
	return out
}

func TestRun(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		instance *v1alpha1.Instance
		want     []string
	}{
		{
			name:     "valid",
			instance: replicaSet(nil),
		},
		{
			name: "component type is checked by the catalog alone",
			instance: replicaSet(func(in *v1alpha1.Instance) {
				engine := in.Spec.Components["engine"]
				engine.Type = "mongod"
				in.Spec.Components["engine"] = engine
			}),
		},
		{
			name: "unknown parameter",
			instance: replicaSet(func(in *v1alpha1.Instance) {
				engine := in.Spec.Components["engine"]
				engine.Parameters = raw(`{"configuration":"x","cacheSizeGb":2}`)
				in.Spec.Components["engine"] = engine
			}),
			want: []string{
				"spec.components[engine].parameters.cacheSizeGb: Unsupported value: the provider declares no such parameter",
			},
		},
		{
			name: "parameter out of bounds",
			instance: replicaSet(func(in *v1alpha1.Instance) {
				engine := in.Spec.Components["engine"]
				engine.Parameters = raw(`{"cacheSizeGB":0}`)
				in.Spec.Components["engine"] = engine
			}),
			want: []string{
				"spec.components[engine].parameters.cacheSizeGB: Invalid value: 0: " +
					"cacheSizeGB in body should be greater than or equal to 1",
			},
		},
		{
			name: "parameters where none are declared",
			instance: replicaSet(func(in *v1alpha1.Instance) {
				in.Spec.Parameters = raw(`{"tls":true}`)
			}),
			want: []string{"spec.parameters.tls: Unsupported value: the provider declares no such parameter"},
		},
		{
			name: "topology parameters",
			instance: &v1alpha1.Instance{Spec: v1alpha1.InstanceSpec{
				Topology: &v1alpha1.TopologySpec{Type: "sharded", Parameters: raw(`{"numShards":0}`)},
				Components: map[string]v1alpha1.ComponentSpec{
					"engine": {}, "proxy": {}, "configServer": {},
				},
			}},
			want: []string{
				"spec.topology.parameters.numShards: Invalid value: 0: " +
					"numShards in body should be greater than or equal to 1",
			},
		},
		{
			name: "core field the topology does not support",
			instance: replicaSet(func(in *v1alpha1.Instance) {
				engine := in.Spec.Components["engine"]
				engine.Image = "mongod:custom"
				engine.SchedulingPolicy = &common.SchedulingPolicy{NodeSelector: map[string]string{"disk": "ssd"}}
				in.Spec.Components["engine"] = engine
			}),
			want: []string{
				"spec.components[engine].image: Unsupported value: the provider does not support this field in the replicaSet topology",
				"spec.components[engine].schedulingPolicy.nodeSelector: Unsupported value: " +
					"the provider does not support this field in the replicaSet topology",
			},
		},
		{
			name: "declared bounds, required and CEL on core fields",
			instance: replicaSet(func(in *v1alpha1.Instance) {
				in.Spec.Components["engine"] = v1alpha1.ComponentSpec{Replicas: new(int32(2))}
			}),
			want: []string{
				"spec.components[engine].storage: Required value",
				"spec.components[engine].replicas: Invalid value: 2: must be odd",
			},
		},
		{
			name: "catalog",
			instance: &v1alpha1.Instance{Spec: v1alpha1.InstanceSpec{
				Version:  "9.0",
				Topology: &v1alpha1.TopologySpec{Type: "sharded"},
				Components: map[string]v1alpha1.ComponentSpec{
					"engine":  {Type: "mysqld", Version: "5.0.0"},
					"proxy":   {},
					"monitor": {},
				},
			}},
			want: []string{
				`spec.version: Unsupported value: "9.0": supported values: "7.0"`,
				`spec.components[engine].type: Unsupported value: "mysqld": supported values: "mongod"`,
				`spec.components[engine].version: Unsupported value: "5.0.0": supported values: "7.0.1", "6.0.1"`,
				`spec.components[monitor]: Unsupported value: "monitor": supported values: "configServer", "engine", "proxy"`,
				"spec.components[configServer]: Required value: the sharded topology requires this component",
			},
		},
		{
			name: "component outside the topology",
			instance: replicaSet(func(in *v1alpha1.Instance) {
				in.Spec.Components["proxy"] = v1alpha1.ComponentSpec{}
			}),
			want: []string{"spec.components[proxy]: Forbidden: not a component of the replicaSet topology, which has engine"},
		},
		{
			name: "unknown topology",
			instance: &v1alpha1.Instance{Spec: v1alpha1.InstanceSpec{
				Topology: &v1alpha1.TopologySpec{Type: "galera"},
			}},
			want: []string{`spec.topology.type: Unsupported value: "galera": supported values: "replicaSet", "sharded"`},
		},
		{
			name:     "no topology type skips the component set",
			instance: &v1alpha1.Instance{Spec: v1alpha1.InstanceSpec{Components: map[string]v1alpha1.ComponentSpec{"proxy": {}}}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			v := &recordingValidator{}
			res := Run(t.Context(), v, controller.ValidationInput{Instance: tc.instance, Provider: testProvider(t)})

			assert.Equal(t, tc.want, errorStrings(res.Errors))
			assert.Equal(t, len(tc.want) == 0, v.called, "Validate runs only when the declared checks pass")
		})
	}
}

func TestRun_ProviderErrorsAndWarnings(t *testing.T) {
	t.Parallel()
	odd := field.Invalid(field.NewPath("spec", "components").Key("engine").Child("replicas"), 1, "must be at least 3")
	v := &recordingValidator{errs: field.ErrorList{odd}}
	in := replicaSet(func(in *v1alpha1.Instance) {
		engine := in.Spec.Components["engine"]
		engine.Version = "6.0.1"
		in.Spec.Components["engine"] = engine
	})

	res := Run(t.Context(), v, controller.ValidationInput{Instance: in, Provider: testProvider(t)})

	require.True(t, v.called)
	assert.Equal(t, field.ErrorList{odd}, res.Errors)
	assert.Equal(t,
		[]string{
			"spec.components[engine].version: version 6.0.1 is deprecated and is removed in provider release 0.9",
			"from the provider",
		},
		res.Warnings,
	)
}

func TestSummary(t *testing.T) {
	t.Parallel()
	errs := make(field.ErrorList, 0, 12)
	for i := range 12 {
		errs = append(errs, field.Required(field.NewPath("spec", fmt.Sprintf("f%d", i)), ""))
	}

	assert.Equal(t, "spec.f0: Required value", Summary(errs[:1]))
	assert.Equal(t,
		"spec.f0: Required value; spec.f1: Required value; spec.f2: Required value; spec.f3: Required value; "+
			"spec.f4: Required value; spec.f5: Required value; spec.f6: Required value; spec.f7: Required value; "+
			"spec.f8: Required value; spec.f9: Required value; and 2 more",
		Summary(errs),
	)
}
