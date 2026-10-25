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

// Package validationtest runs a provider's validation the way the runtime
// does, against the Provider spec the provider's chart publishes, so a rule is
// tested with a table of Instance fixtures and no cluster.
//
//	func TestValidate(t *testing.T) {
//	    v := validationtest.New(t, provider.New(), "../../charts/my-provider/generated/provider-spec.yaml")
//	    res := v.Run(validationtest.Load(t, "testdata/sharded-one-config-server.yaml"), nil)
//	    require.Len(t, res.Errors, 1)
//	    assert.Equal(t, "spec.components[configServer].replicas", res.Errors[0].Field)
//	}
package validationtest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"

	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
	"github.com/openeverest/openeverest/v2/provider-runtime/validation"
)

// Validator runs the runtime's validation for one provider.
type Validator struct {
	tb       testing.TB
	provider validation.Validator
	spec     *v1alpha1.Provider
}

// New returns a Validator for p against the Provider spec at path, the
// generated provider-spec.yaml the provider's chart ships.
func New(tb testing.TB, p validation.Validator, path string) *Validator {
	tb.Helper()
	data, err := os.ReadFile(filepath.Clean(path))
	require.NoError(tb, err, "reading the provider spec")
	spec := &v1alpha1.Provider{}
	require.NoError(tb, yaml.Unmarshal(data, spec), "decoding the provider spec")
	if named, ok := p.(interface{ Name() string }); ok {
		spec.Name = named.Name()
	}
	return &Validator{tb: tb, provider: p, spec: spec}
}

// Run validates in as the runtime does before Sync. Pass the previous Instance
// as old to exercise rules about an update, or nil for a create.
func (v *Validator) Run(in, old *v1alpha1.Instance) controller.ValidationResult {
	v.tb.Helper()
	return validation.Run(v.tb.Context(), v.provider, controller.ValidationInput{
		Instance: in,
		Old:      old,
		Provider: v.spec.DeepCopy(),
	})
}

// Load reads an Instance from a YAML file. Fields the Instance does not have
// fail the test, as they would fail kubectl apply.
func Load(tb testing.TB, path string) *v1alpha1.Instance {
	tb.Helper()
	data, err := os.ReadFile(filepath.Clean(path))
	require.NoError(tb, err, "reading the Instance")
	in := &v1alpha1.Instance{}
	require.NoError(tb, yaml.UnmarshalStrict(data, in), "decoding the Instance")
	return in
}
