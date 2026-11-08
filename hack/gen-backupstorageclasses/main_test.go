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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

func TestReasonFor(t *testing.T) {
	t.Parallel()

	obj := func(props map[string]apiextensionsv1.JSONSchemaProps) apiextensionsv1.JSONSchemaProps {
		return apiextensionsv1.JSONSchemaProps{Type: "object", Properties: props}
	}
	str := apiextensionsv1.JSONSchemaProps{Type: "string"}
	view := obj(map[string]apiextensionsv1.JSONSchemaProps{
		"location":       obj(map[string]apiextensionsv1.JSONSchemaProps{"objectStore": obj(map[string]apiextensionsv1.JSONSchemaProps{"bucket": str})}),
		"parameters":     obj(map[string]apiextensionsv1.JSONSchemaProps{"endpointURL": str}),
		"tls":            obj(nil),
		"authentication": obj(map[string]apiextensionsv1.JSONSchemaProps{"mode": str}),
	})

	tests := []struct {
		path    string
		want    string
		wantErr string
	}{
		{path: ".location.objectStore.bucket", want: reasonInvalidLocation},
		{path: ".location", want: reasonInvalidLocation},
		{path: ".parameters.endpointURL", want: reasonInvalidParameters},
		{path: ".tls", want: reasonInvalidParameters},
		{path: "", wantErr: "must name the field"},
		{path: "location", wantErr: "must name the field"},
		{path: ".location..bucket", wantErr: "must name the field"},
		{path: ".location.objectStore.region", wantErr: `no field "region"`},
		{path: ".authentication.mode", wantErr: "judges location, parameters or tls"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			t.Parallel()

			got, err := reasonFor(&view, tt.path)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestValidationsRejectTypographicQuotes(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	crd := `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: bads.schema.backupstorageclasses.openeverest.io
spec:
  group: schema.backupstorageclasses.openeverest.io
  names: {kind: Bad, plural: bads}
  scope: Cluster
  versions:
  - name: v1
    served: true
    storage: true
    schema:
      openAPIV3Schema:
        type: object
        properties:
          spec:
            type: object
            properties:
              parameters:
                type: object
                properties:
                  port: {type: string}
            x-kubernetes-validations:
            - rule: "self.parameters.port == ` + "\u201d" + `"
              message: m
              fieldPath: .parameters.port
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "schema.backupstorageclasses.openeverest.io_bads.yaml"), []byte(crd), 0o600))
	_, err := validations(dir, "Bad")
	require.ErrorContains(t, err, "typographic quote")
}
