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

package instance

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// midRolloutJSON is an Instance whose engine grows from one member to three:
// two pods exist and are Ready, and the provider reports three to come.
const midRolloutJSON = `{"metadata":{"name":"orders","generation":1},"status":{
	"observedGeneration":1,"phase":"Pending",
	"components":[
		{"name":"engine","readyReplicas":2,"replicas":2,"desiredReplicas":3},
		{"name":"proxy","readyReplicas":1,"replicas":1}
	]
}}`

// A component is read against the replicas the provider says it will have,
// and against the pods that exist when the provider does not say.
func TestComponents_MidRollout(t *testing.T) {
	t.Parallel()

	inst := instFromJSON(t, midRolloutJSON)

	t.Run("status", func(t *testing.T) {
		t.Parallel()
		var out bytes.Buffer

		printComponentTable(&out, inst)

		assert.Equal(t, "\nComponents:\n  NAME    READY\n  engine  2/3\n  proxy   1/1\n", out.String())
	})

	t.Run("wait", func(t *testing.T) {
		t.Parallel()

		_, msg := instanceCondition(inst)

		assert.Equal(t, "Creating (engine 2/3 ready, proxy 1/1 ready)", msg)
	})
}

// everestctl reads a component through the API client's type. It has to read
// every component case of the shared fixtures as the Kubernetes type is read.
func TestComponents_Fixtures(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile("../../instancestatus/testdata/display.json")
	require.NoError(t, err)
	var fixtures struct {
		Components []struct {
			Name      string          `json:"name"`
			Component json.RawMessage `json:"component"`
			Want      struct {
				Text string `json:"text"`
			} `json:"want"`
		} `json:"components"`
	}
	require.NoError(t, json.Unmarshal(raw, &fixtures))
	require.NotEmpty(t, fixtures.Components)

	for _, tc := range fixtures.Components {
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			inst := instFromJSON(t, `{"status":{"components":[`+string(tc.Component)+`]}}`)
			name := (*inst.Status.Components)[0].Name

			assert.Equal(t, name+" "+tc.Want.Text+" ready", componentSummary(inst))
		})
	}
}
