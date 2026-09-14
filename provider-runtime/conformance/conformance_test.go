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

package conformance

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"

	commonv1alpha1 "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
)

func testSpec() *corev1alpha1.ProviderSpec {
	stringSchema := func(property string) *commonv1alpha1.ParametersSchema {
		return &commonv1alpha1.ParametersSchema{
			OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
				Type: "object",
				Properties: map[string]apiextensionsv1.JSONSchemaProps{
					property: {Type: "string"},
				},
			},
		}
	}
	return &corev1alpha1.ProviderSpec{
		Components: map[string]corev1alpha1.Component{
			"engine":     {Type: "db", ParametersSchema: stringSchema("configuration")},
			"monitoring": {Type: "pmm"},
		},
		Topologies: map[string]corev1alpha1.Topology{
			"simple": {
				Components: map[string]corev1alpha1.TopologyComponent{
					"engine":     {},
					"monitoring": {Optional: true},
				},
			},
		},
	}
}

func TestCollectPaths(t *testing.T) {
	t.Parallel()

	var paths []string
	collectPaths(map[string]any{
		"sections": map[string]any{
			"resources": map[string]any{
				"uiType": "group",
				"components": map[string]any{
					"cpu": map[string]any{"path": "spec.components.engine.resources.limits.cpu"},
				},
			},
			"nodes": map[string]any{
				"path": "spec.components.engine.replicas",
				"validation": map[string]any{
					"celExpressions": []any{
						// Read by the form, not written by it.
						map[string]any{"celExpr": "spec.components.engine.replicas % 2 == 1"},
					},
				},
			},
		},
		// Addresses the options payload, not an Instance field.
		"labelPath": "name",
	}, &paths)

	assert.ElementsMatch(t, []string{
		"spec.components.engine.resources.limits.cpu",
		"spec.components.engine.replicas",
	}, paths)
}

func TestResolveLeaf(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		path string
		want leafKind
	}{
		"scalar on the instance":          {"spec.version", leafString},
		"through the components map":      {"spec.components.engine.replicas", leafInteger},
		"quantity":                        {"spec.components.engine.storage.size", leafQuantity},
		"quantity behind a resource name": {"spec.components.engine.resources.limits.cpu", leafQuantity},
		"declared parameters schema":      {"spec.components.engine.parameters.configuration", leafString},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := resolveLeaf(testSpec(), "simple", tt.path)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestResolveLeafErrors(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		path string
		want string
	}{
		"field that does not exist": {
			"spec.components.engine.config",
			`no field "config" on ComponentSpec`,
		},
		"property absent from the declared schema": {
			"spec.components.engine.parameters.nope",
			`no property "nope" in the declared parameters schema`,
		},
		"component with no declared schema": {
			"spec.components.monitoring.parameters.monitoringConfigName",
			`no parametersSchema declared for "components.monitoring.parameters"`,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := resolveLeaf(testSpec(), "simple", tt.path)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.want)
		})
	}
}

// The baseline stays as small as the topology allows, so a provider is never
// asked to render an optional component the probe is not even about.
func TestBuildInstanceIncludesOptionalComponentsOnlyWhenAddressed(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		path string
		want []string
	}{
		"unrelated path":     {"spec.version", []string{"engine"}},
		"addresses required": {"spec.components.engine.replicas", []string{"engine"}},
		"addresses optional": {"spec.components.monitoring.parameters.x", []string{"engine", "monitoring"}},
		"no path at all":     {"", []string{"engine"}},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			instance, err := buildInstance(testSpec(), "test-provider", "simple", tt.path, nil)
			require.NoError(t, err)

			got := make([]string, 0, len(instance.Spec.Components))
			for name := range instance.Spec.Components {
				got = append(got, name)
			}
			assert.ElementsMatch(t, tt.want, got)
		})
	}
}

func TestBuildInstanceSetsTheProbeValue(t *testing.T) {
	t.Parallel()

	instance, err := buildInstance(testSpec(), "test-provider", "simple", "spec.components.engine.replicas", int64(7919))
	require.NoError(t, err)

	engine := instance.Spec.Components["engine"]
	require.NotNil(t, engine.Replicas)
	assert.Equal(t, int32(7919), *engine.Replicas)
}

// boolProbeProvider always writes an unrelated true, which is what a value
// token cannot tell apart from a provider that reads the probed field.
type boolProbeProvider struct {
	readsBackupEnabled bool
}

func (p *boolProbeProvider) Name() string                       { return "bool-probe" }
func (p *boolProbeProvider) Types() func(*runtime.Scheme) error { return nil }
func (p *boolProbeProvider) Validate(*controller.Context) error { return nil }
func (p *boolProbeProvider) Cleanup(*controller.Context) error  { return nil }

func (p *boolProbeProvider) Status(*controller.Context) (controller.Status, error) {
	return controller.Status{}, nil
}

func (p *boolProbeProvider) Sync(c *controller.Context) error {
	data := map[string]string{"unrelatedFlag": "true"}
	if p.readsBackupEnabled && c.Instance().Spec.Backup != nil && c.Instance().Spec.Backup.Enabled {
		data["backup"] = "on"
	}
	return c.Apply(&corev1.ConfigMap{ObjectMeta: c.ObjectMeta("probe"), Data: data})
}

func TestProbeBoolComparesRendersRatherThanSearchingForTrue(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		reads bool
		want  outcome
	}{
		"reading the field changes what is applied": {true, reconciled},
		"an unrelated true is not evidence":         {false, ignored},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := probeBool(
				Config{Provider: &boolProbeProvider{readsBackupEnabled: tt.reads}},
				testSpec(), "simple", "spec.backup.enabled",
			)
			assert.Equal(t, tt.want, got.outcome, got.detail)
		})
	}
}
