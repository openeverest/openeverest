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

package validationtest_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller/validationtest"
)

// shardedProvider requires three config servers when the engine has more than
// one replica.
type shardedProvider struct{}

func (shardedProvider) Name() string { return "sharded-db" }

func (shardedProvider) Validate(in controller.ValidationInput) controller.ValidationResult {
	if in.Provider.Name != "sharded-db" || in.Instance.GetTopologyType() != "sharded" {
		return controller.ValidationResult{}
	}
	engine := *in.Instance.Spec.Components["engine"].Replicas
	configServer := *in.Instance.Spec.Components["configServer"].Replicas
	if engine > 1 && configServer < 3 {
		return controller.ValidationResult{Errors: field.ErrorList{field.Invalid(
			field.NewPath("spec", "components").Key("configServer").Child("replicas"), configServer,
			"must be at least 3 when spec.components[engine].replicas is greater than 1",
		)}}
	}
	return controller.ValidationResult{}
}

func TestRun(t *testing.T) {
	t.Parallel()
	v := validationtest.New(t, shardedProvider{}, "testdata/provider-spec.yaml")

	tests := []struct {
		fixture string
		want    []string
	}{
		{fixture: "testdata/sharded.yaml"},
		{
			fixture: "testdata/sharded-one-config-server.yaml",
			want:    []string{"spec.components[configServer].replicas"},
		},
		{
			fixture: "testdata/sharded-unknown-parameter.yaml",
			want:    []string{"spec.topology.parameters.shards"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.fixture, func(t *testing.T) {
			t.Parallel()
			res := v.Run(validationtest.Load(t, tc.fixture), nil)

			var got []string
			for _, err := range res.Errors {
				got = append(got, err.Field)
			}
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestLoadRejectsUnknownFields(t *testing.T) {
	t.Parallel()
	ft := &fakeT{TB: t}
	func() {
		defer func() { _ = recover() }()
		validationtest.Load(ft, "testdata/misspelt-field.yaml")
	}()
	require.True(t, ft.failed, "a fixture with a misspelt field must fail the test")
}

// fakeT records a failure instead of failing the test that owns it.
type fakeT struct {
	testing.TB

	failed bool
}

func (f *fakeT) Errorf(string, ...any) { f.failed = true }
func (f *fakeT) FailNow()              { f.failed = true; panic("FailNow") }
func (f *fakeT) Helper()               {}
