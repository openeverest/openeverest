// everest
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

package instancestatus

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

// componentCase is one entry of the "components" of testdata/display.json.
type componentCase struct {
	Name      string                       `json:"name"`
	About     string                       `json:"about"`
	Component corev1alpha1.ComponentStatus `json:"component"`
	Want      struct {
		Ready          int32  `json:"ready"`
		Total          int32  `json:"total"`
		TotalIsDesired bool   `json:"totalIsDesired"`
		Text           string `json:"text"`
	} `json:"want"`
}

func TestCountReady_Fixtures(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile("testdata/display.json")
	require.NoError(t, err)
	var fixtures struct {
		Components []componentCase `json:"components"`
	}
	require.NoError(t, json.Unmarshal(raw, &fixtures))
	require.NotEmpty(t, fixtures.Components)

	for _, tc := range fixtures.Components {
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			c := tc.Component

			got := CountReady(c.ReadyReplicas, c.Replicas, c.DesiredReplicas)

			assert.Equal(t, ReadyCount{Ready: tc.Want.Ready, Total: tc.Want.Total, TotalIsDesired: tc.Want.TotalIsDesired}, got)
			assert.Equal(t, tc.Want.Text, got.String())
		})
	}
}
