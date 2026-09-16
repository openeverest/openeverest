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
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/yaml"

	commonv1alpha1 "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

// specDeclaring builds a spec whose "simple" topology gives engine the schema.
func specDeclaring(t *testing.T, schema string) *corev1alpha1.ProviderSpec {
	t.Helper()

	props := &apiextensionsv1.JSONSchemaProps{}
	require.NoError(t, yaml.Unmarshal([]byte(schema), props))

	spec := testSpec()
	spec.Topologies["simple"] = corev1alpha1.Topology{
		Components: map[string]corev1alpha1.TopologyComponent{
			"engine": {SupportedFields: &commonv1alpha1.ParametersSchema{OpenAPIV3Schema: props}},
		},
	}
	return spec
}

func TestDeclaredPaths(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		schema string
		want   []string
	}{
		"a scalar is probed directly": {
			schema: "properties:\n  replicas: {}",
			want:   []string{"spec.components.engine.replicas"},
		},
		"a struct expands to its scalars": {
			schema: "properties:\n  storage: {}",
			want: []string{
				"spec.components.engine.storage.size",
				"spec.components.engine.storage.storageClass",
			},
		},
		// The case that makes an over-broad declaration observable: promising
		// the whole policy has to hold for schedulerName, not only for the
		// affinity a provider happens to read.
		"a group declared whole expands past what a provider may read": {
			schema: "properties:\n  schedulingPolicy: {}",
			want:   []string{"spec.components.engine.schedulingPolicy.schedulerName"},
		},
		"a group narrowed to an unprobeable member yields nothing": {
			schema: "properties:\n  schedulingPolicy:\n    properties:\n      affinity: {}",
			want:   nil,
		},
		// Maps have no fixed member to address, so resources is left to the UI
		// probe, which knows the key the form binds.
		"a map yields nothing": {
			schema: "properties:\n  resources: {}",
			want:   nil,
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := declaredPaths(specDeclaring(t, tc.schema), "simple")
			if len(tc.want) == 0 {
				assert.Empty(t, got)
				return
			}
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestDeclaredPathsIgnoresComponentsWithoutADeclaration(t *testing.T) {
	t.Parallel()

	assert.Empty(t, declaredPaths(testSpec(), "simple"))
}
