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
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openeverest/openeverest/v2/client"
	"github.com/openeverest/openeverest/v2/pkg/cli/wait"
	"github.com/openeverest/openeverest/v2/pkg/instancestatus"
)

// displayFixture is one case of the fixtures that every reader of an
// Instance's status is held to, decoded the way everestctl receives an
// Instance: as the API client's type.
type displayFixture struct {
	Name     string          `json:"name"`
	Now      time.Time       `json:"now"`
	Instance client.Instance `json:"instance"`
	Want     struct {
		Tone      instancestatus.Tone         `json:"tone"`
		Reason    string                      `json:"reason"`
		Label     string                      `json:"label"`
		Available instancestatus.Availability `json:"available"`
		Sentence  string                      `json:"sentence"`
		Since     *time.Time                  `json:"since"`
		Wait      instancestatus.Outcome      `json:"wait"`
	} `json:"want"`
}

func loadDisplayFixtures(t *testing.T) []displayFixture {
	t.Helper()
	raw, err := os.ReadFile("../../instancestatus/testdata/display.json")
	require.NoError(t, err)
	var fixtures struct {
		Cases []displayFixture `json:"cases"`
	}
	require.NoError(t, json.Unmarshal(raw, &fixtures))
	require.NotEmpty(t, fixtures.Cases)
	return fixtures.Cases
}

// displayFixtureNamed returns the case of that name, with the Instance named
// after it so that several cases can be told apart in one list.
func displayFixtureNamed(t *testing.T, name string) displayFixture {
	t.Helper()
	for _, fixture := range loadDisplayFixtures(t) {
		if fixture.Name == name {
			fixture.Instance.Metadata.Name = name
			return fixture
		}
	}
	require.FailNow(t, "no such fixture", name)
	return displayFixture{}
}

// everestctl reads an Instance through the API client's type. It has to read
// every case exactly as the Kubernetes type is read, and its --wait has to
// end when that reading says so.
func TestStatusInput_Fixtures(t *testing.T) {
	t.Parallel()

	waitOutcomes := map[instancestatus.Outcome]wait.Outcome{
		instancestatus.Pending:   wait.Pending,
		instancestatus.Succeeded: wait.Succeeded,
		instancestatus.Failed:    wait.Failed,
	}

	for _, tc := range loadDisplayFixtures(t) {
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			in := statusInput(&tc.Instance)
			got := instancestatus.Describe(in, tc.Now)

			assert.Equal(t, tc.Want.Tone, got.Tone, "tone")
			assert.Equal(t, tc.Want.Reason, got.Reason, "reason")
			assert.Equal(t, tc.Want.Label, got.Label, "label")
			assert.Equal(t, tc.Want.Available, got.Available, "available")
			assert.Equal(t, tc.Want.Sentence, got.Sentence, "sentence")
			if tc.Want.Since == nil {
				assert.Nil(t, got.Since, "since")
			} else if assert.NotNil(t, got.Since, "since") {
				assert.True(t, tc.Want.Since.Equal(*got.Since), "since: want %s, got %s", tc.Want.Since, got.Since)
			}
			assert.Equal(t, tc.Want.Wait, instancestatus.WaitOutcome(in), "wait")

			outcome, _ := instanceCondition(&tc.Instance)
			assert.Equal(t, waitOutcomes[tc.Want.Wait], outcome, "--wait")
		})
	}
}
