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

package handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestParseSecretPatch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		patch string
		want  *SecretPatch
		err   string
	}{
		{
			name:  "stringData sets keys",
			patch: `{"metadata": {"resourceVersion": "7"}, "stringData": {"token": "new"}}`,
			want:  &SecretPatch{ResourceVersion: "7", Set: map[string][]byte{"token": []byte("new")}},
		},
		{
			name:  "data sets base64 values and null removes a key",
			patch: `{"metadata": {"resourceVersion": "7"}, "data": {"token": "bmV3", "orgID": null, "empty": ""}}`,
			want: &SecretPatch{
				ResourceVersion: "7",
				Set:             map[string][]byte{"token": []byte("new"), "empty": {}},
				Remove:          []string{"orgID"},
			},
		},
		{
			name:  "stringData wins over data for the same key",
			patch: `{"metadata": {"resourceVersion": "7"}, "data": {"token": null}, "stringData": {"token": "new"}}`,
			want:  &SecretPatch{ResourceVersion: "7", Set: map[string][]byte{"token": []byte("new")}},
		},
		{
			name: "a Secret returned by GET is a patch that changes no data",
			patch: `{
				"apiVersion": "v1", "kind": "Secret",
				"metadata": {
					"name": "pmm", "namespace": "ns", "uid": "1", "resourceVersion": "7",
					"creationTimestamp": "2026-11-10T10:00:00Z",
					"labels": {"openeverest.io/managed": "true"},
					"annotations": {"example.com/note": "x"},
					"managedFields": [{"manager": "everest", "operation": "Update", "fieldsType": "FieldsV1", "fieldsV1": {"f:data": {}}}]
				},
				"type": "Opaque",
				"immutable": false
			}`,
			want: &SecretPatch{
				ResourceVersion: "7",
				Set:             map[string][]byte{},
				Labels:          map[string]*string{"openeverest.io/managed": new("true")},
				Type:            new(corev1.SecretTypeOpaque),
				Immutable:       new(false),
			},
		},
		{
			name:  "a label set to null is recorded as removed",
			patch: `{"metadata": {"resourceVersion": "7", "labels": {"team": null}}}`,
			want:  &SecretPatch{ResourceVersion: "7", Set: map[string][]byte{}, Labels: map[string]*string{"team": nil}},
		},
		{
			name:  "not an object",
			patch: `["token"]`,
			err:   "patch must be a JSON object",
		},
		{
			name:  "null",
			patch: `null`,
			err:   "patch must be a JSON object",
		},
		{
			name:  "a member a Secret does not have",
			patch: `{"metadata": {"resourceVersion": "7"}, "stringdata": {"token": "new"}}`,
			err:   `unknown field "stringdata"`,
		},
		{
			name:  "a misspelt metadata member",
			patch: `{"metadata": {"resourceVersion": "7", "lables": {}}, "stringData": {"token": "new"}}`,
			err:   `unknown field "metadata.lables"`,
		},
		{
			name:  "a member in the wrong case",
			patch: `{"metadata": {"resourceversion": "7"}, "stringData": {"token": "new"}}`,
			err:   `unknown field "metadata.resourceversion"`,
		},
		{
			name:  "no resourceVersion",
			patch: `{"stringData": {"token": "new"}}`,
			err:   "metadata.resourceVersion is required",
		},
		{
			name:  "an empty resourceVersion",
			patch: `{"metadata": {"resourceVersion": ""}, "stringData": {"token": "new"}}`,
			err:   "metadata.resourceVersion is required",
		},
		{
			name:  "data that is not base64",
			patch: `{"metadata": {"resourceVersion": "7"}, "data": {"token": "not base64!"}}`,
			err:   "patch is not a Secret",
		},
		{
			name:  "null data",
			patch: `{"metadata": {"resourceVersion": "7"}, "data": null}`,
			err:   "data can't be null",
		},
		{
			name:  "null stringData",
			patch: `{"metadata": {"resourceVersion": "7"}, "stringData": null}`,
			err:   "stringData can't be null",
		},
		{
			name:  "a stringData key set to null",
			patch: `{"metadata": {"resourceVersion": "7"}, "stringData": {"orgID": null}}`,
			err:   `stringData["orgID"] is null; remove a key with data: {"orgID": null}`,
		},
		{
			name:  "null labels",
			patch: `{"metadata": {"resourceVersion": "7", "labels": null}}`,
			err:   "metadata.labels can't be null",
		},
		{
			name:  "null type",
			patch: `{"metadata": {"resourceVersion": "7"}, "type": null}`,
			err:   "type can't be null",
		},
		{
			name:  "null immutable",
			patch: `{"metadata": {"resourceVersion": "7"}, "immutable": null}`,
			err:   "immutable can't be null",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseSecretPatch([]byte(tt.patch))
			if tt.err != "" {
				require.ErrorContains(t, err, tt.err)
				assert.NotContains(t, err.Error(), "not base64!", "errors name keys, never values")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSecretPatchApply(t *testing.T) {
	t.Parallel()

	p := &SecretPatch{
		Set:    map[string][]byte{"token": []byte("new"), "region": []byte("eu")},
		Remove: []string{"orgID", "absent"},
	}
	data := map[string][]byte{"token": []byte("old"), "orgID": []byte("1"), "kept": []byte("k")}

	assert.Equal(t, map[string][]byte{
		"token":  []byte("new"),
		"region": []byte("eu"),
		"kept":   []byte("k"),
	}, p.Apply(data))
	assert.Equal(t, []byte("old"), data["token"], "the stored data is not modified")
	assert.Equal(t, map[string][]byte{"token": []byte("new"), "region": []byte("eu")}, p.Apply(nil))
}

func TestSecretPatchDataMergePatch(t *testing.T) {
	t.Parallel()

	p := &SecretPatch{
		ResourceVersion: "7",
		Set:             map[string][]byte{"token": []byte("new")},
		Remove:          []string{"orgID"},
		Labels:          map[string]*string{"team": new("a")},
		Type:            new(corev1.SecretTypeOpaque),
	}

	got, err := p.DataMergePatch()
	require.NoError(t, err)
	assert.JSONEq(t, `{"metadata": {"resourceVersion": "7"}, "data": {"token": "bmV3", "orgID": null}}`, string(got),
		"only the precondition and the data reach Kubernetes")
}
