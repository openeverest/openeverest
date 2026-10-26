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

package v1alpha1_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/yaml"

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

func TestMonitoringKindPatternMatchesCRDs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		crd  string
		path []string
	}{
		{
			name: "Provider componentTypes kind",
			crd:  "core.openeverest.io_providers.yaml",
			path: []string{"spec", "monitoring", "componentTypes", "{}", "kind"},
		},
		{
			name: "Instance metrics endpoint kind",
			crd:  "core.openeverest.io_instances.yaml",
			path: []string{"status", "monitoring", "sources", "metrics", "[]", "kind"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, corev1alpha1.MonitoringKindPattern, crdSchemaAt(t, tt.crd, tt.path).Pattern)
		})
	}
}

// crdSchemaAt walks a generated CRD schema: "[]" steps into list items,
// "{}" into map values, anything else into a property.
func crdSchemaAt(t *testing.T, crdFile string, path []string) apiextensionsv1.JSONSchemaProps {
	t.Helper()

	raw, err := fs.ReadFile(os.DirFS(filepath.Join("..", "..", "..", "config", "crd", "bases")), crdFile)
	require.NoError(t, err)
	crd := &apiextensionsv1.CustomResourceDefinition{}
	require.NoError(t, yaml.Unmarshal(raw, crd))
	require.Len(t, crd.Spec.Versions, 1)

	schema := *crd.Spec.Versions[0].Schema.OpenAPIV3Schema
	for _, step := range path {
		switch step {
		case "[]":
			require.NotNil(t, schema.Items, "%s: no list items before %v", crdFile, path)
			schema = *schema.Items.Schema
		case "{}":
			require.NotNil(t, schema.AdditionalProperties, "%s: no map values before %v", crdFile, path)
			schema = *schema.AdditionalProperties.Schema
		default:
			child, ok := schema.Properties[step]
			require.True(t, ok, "%s: no property %q in %v", crdFile, step, path)
			schema = child
		}
	}

	return schema
}
