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

package uischema

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
)

type expectedIssue struct {
	Rule     Rule   `json:"rule"`
	Topology string `json:"topology"`
	Group    string `json:"group"`
	Field    string `json:"field,omitempty"`
}

type sharedCase struct {
	Name     string          `json:"name"`
	UISchema map[string]any  `json:"uiSchema"`
	Issues   []expectedIssue `json:"issues"`
}

// The same file drives the UI preprocess tests; see the package doc.
func loadToggleableCases(t *testing.T) []sharedCase {
	t.Helper()
	raw, err := os.ReadFile("testdata/toggleable.yaml")
	require.NoError(t, err)
	var doc struct {
		Cases []sharedCase `json:"cases"`
	}
	require.NoError(t, yaml.Unmarshal(raw, &doc))
	require.NotEmpty(t, doc.Cases)
	return doc.Cases
}

func TestValidateToggleableSharedCases(t *testing.T) {
	t.Parallel()
	for _, tc := range loadToggleableCases(t) {
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			issues := Validate(tc.UISchema)
			got := make([]expectedIssue, 0, len(issues))
			for _, issue := range issues {
				assert.NotEmpty(t, issue.Message)
				got = append(got, expectedIssue{Rule: issue.Rule, Topology: issue.Topology, Group: issue.Group, Field: issue.Field})
			}
			want := tc.Issues
			if want == nil {
				want = []expectedIssue{}
			}
			assert.Equal(t, want, got)
		})
	}
}

func TestValidateIgnoresMalformedTopologies(t *testing.T) {
	t.Parallel()
	assert.Empty(t, Validate(map[string]any{
		"replicaSet": "not a topology",
		"sharded":    map[string]any{"sections": []any{}},
	}))
}
