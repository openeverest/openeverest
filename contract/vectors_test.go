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

package contract_test

import (
	"bytes"
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openeverest/openeverest/v2/contract"
)

type vectors struct {
	Order       []string            `json:"order"`
	Invalid     []string            `json:"invalid"`
	Development []developmentVector `json:"development"`
	Check       []checkVector       `json:"check"`
	Acknowledge []acknowledgeVector `json:"acknowledge"`
}

type developmentVector struct {
	Version     string `json:"version"`
	Development bool   `json:"development"`
}

type checkVector struct {
	Ext    string                 `json:"ext"`
	Min    string                 `json:"min"`
	Cur    string                 `json:"cur"`
	Status metav1.ConditionStatus `json:"status"`
	Reason string                 `json:"reason"`
}

type acknowledgeVector struct {
	Ack             string `json:"ack"`
	ContractVersion string `json:"contractVersion"`
	CoreVersion     string `json:"coreVersion"`
	Holds           bool   `json:"holds"`
}

func loadVectors(t *testing.T) vectors {
	t.Helper()

	dec := json.NewDecoder(bytes.NewReader(contract.OrderVectors))
	dec.DisallowUnknownFields()
	var v vectors
	require.NoError(t, dec.Decode(&v))
	require.NotEmpty(t, v.Order)
	require.NotEmpty(t, v.Invalid)
	require.NotEmpty(t, v.Development)
	require.NotEmpty(t, v.Check)
	require.NotEmpty(t, v.Acknowledge)
	return v
}

func TestOrderVectorsAreStrictlyAscending(t *testing.T) {
	t.Parallel()

	order := loadVectors(t).Order
	for i, a := range order {
		assert.True(t, contract.Valid(a), "order[%d] %q must be valid", i, a)
		assert.Equal(t, 0, contract.Compare(a, a), "%q must equal itself", a)
		for _, b := range order[i+1:] {
			assert.Equal(t, -1, contract.Compare(a, b), "%q must sort before %q", a, b)
			assert.Equal(t, 1, contract.Compare(b, a), "%q must sort after %q", b, a)
		}
	}
}

func TestInvalidVectorsAreRejected(t *testing.T) {
	t.Parallel()

	const valid = "2.0.0"
	for _, v := range loadVectors(t).Invalid {
		assert.False(t, contract.Valid(v), "%q must be invalid", v)

		wantExt := contract.ReasonInvalidVersion
		if v == "" {
			wantExt = contract.ReasonUnversioned
		}
		assert.Equal(t, contract.Result{Status: metav1.ConditionUnknown, Reason: wantExt},
			contract.Check(v, valid, valid), "stamp %q", v,
		)
		assert.Equal(t, contract.Result{Status: metav1.ConditionUnknown, Reason: contract.ReasonInvalidVersion},
			contract.Check(valid, v, valid), "minimum %q", v,
		)
		assert.Equal(t, contract.Result{Status: metav1.ConditionUnknown, Reason: contract.ReasonInvalidVersion},
			contract.Check(valid, valid, v), "current %q", v,
		)
	}
}

func TestDevelopmentVectors(t *testing.T) {
	t.Parallel()

	for _, tc := range loadVectors(t).Development {
		assert.Equal(t, tc.Development, contract.IsDevelopment(tc.Version), "IsDevelopment(%q)", tc.Version)
	}
}

func TestDevelopmentVectorsCoverEveryCase(t *testing.T) {
	t.Parallel()

	pre := func(v developmentVector) string {
		v.Version, _, _ = strings.Cut(v.Version, "+")
		_, p, _ := strings.Cut(v.Version, "-")
		return p
	}
	cases := []struct {
		name  string
		match func(v developmentVector) bool
	}{
		{
			name:  "the value after a pre-release",
			match: func(v developmentVector) bool { return v.Development && strings.HasSuffix(pre(v), ".next") },
		},
		{
			name:  "the value after a release",
			match: func(v developmentVector) bool { return v.Development && pre(v) == "alpha.0" },
		},
		{
			name: "a core dev build, which users can install",
			match: func(v developmentVector) bool {
				return !v.Development && contract.Valid(v.Version) && strings.HasPrefix(pre(v), "dev.")
			},
		},
		{
			name:  "an invalid version",
			match: func(v developmentVector) bool { return !v.Development && !contract.Valid(v.Version) },
		},
	}
	devs := loadVectors(t).Development
	for _, c := range cases {
		assert.True(t, slices.ContainsFunc(devs, c.match), "no development vector for %s", c.name)
	}
}

func TestCheckVectors(t *testing.T) {
	t.Parallel()

	for _, tc := range loadVectors(t).Check {
		want := contract.Result{Status: tc.Status, Reason: tc.Reason}
		w := contract.Window{Min: tc.Min, Current: tc.Cur}
		assert.Equal(t, want, contract.Check(tc.Ext, tc.Min, tc.Cur), "Check(%q, %q, %q)", tc.Ext, tc.Min, tc.Cur)
		assert.Equal(t, want, w.Check(tc.Ext), "%+v.Check(%q)", w, tc.Ext)
		assertUserFacing(t, contract.Message(want, tc.Ext, w))
	}
}

func TestCheckVectorsCoverEveryReasonAndEdge(t *testing.T) {
	t.Parallel()

	reasons := make(map[string]bool)
	var atMin, atCurrent bool
	for _, tc := range loadVectors(t).Check {
		reasons[tc.Reason] = true
		if tc.Reason == contract.ReasonSupported {
			atMin = atMin || tc.Ext == tc.Min
			atCurrent = atCurrent || tc.Ext == tc.Cur
		}
	}
	for _, reason := range []string{
		contract.ReasonSupported,
		contract.ReasonCoreTooOld,
		contract.ReasonExtensionTooOld,
		contract.ReasonUnversioned,
		contract.ReasonInvalidVersion,
		contract.ReasonCoreVersionSkew,
	} {
		assert.True(t, reasons[reason], "no check vector for %s", reason)
	}
	assert.True(t, atMin, "no supported check vector with the stamp at the bottom of the window")
	assert.True(t, atCurrent, "no supported check vector with the stamp at the top of the window")
}

func TestAcknowledgeVectors(t *testing.T) {
	t.Parallel()

	for _, tc := range loadVectors(t).Acknowledge {
		assert.Equal(t, tc.Holds, contract.AcknowledgementHolds(tc.Ack, tc.ContractVersion, tc.CoreVersion),
			"AcknowledgementHolds(%q, %q, %q)", tc.Ack, tc.ContractVersion, tc.CoreVersion,
		)
	}
}

func TestAcknowledgeVectorsCoverEveryCase(t *testing.T) {
	t.Parallel()

	valid := func(v acknowledgeVector) bool {
		return contract.Valid(v.Ack) && contract.Valid(v.ContractVersion) && contract.Valid(v.CoreVersion)
	}
	cases := []struct {
		name  string
		match func(v acknowledgeVector) bool
	}{
		{
			name:  "an acknowledgement of the installed release",
			match: func(v acknowledgeVector) bool { return v.Holds && v.Ack == v.CoreVersion },
		},
		{
			name: "an acknowledgement kept by a patch release that keeps the contract",
			match: func(v acknowledgeVector) bool {
				return v.Holds && contract.Compare(v.Ack, v.CoreVersion) < 0
			},
		},
		{
			name: "an acknowledgement written ahead of the installed release",
			match: func(v acknowledgeVector) bool {
				return valid(v) && contract.Compare(v.Ack, v.CoreVersion) > 0 &&
					contract.Compare(v.ContractVersion, v.Ack) <= 0
			},
		},
		{
			name: "an acknowledgement expired by a release that moved the contract",
			match: func(v acknowledgeVector) bool {
				return valid(v) && contract.Compare(v.Ack, v.ContractVersion) < 0 &&
					contract.Compare(v.ContractVersion, v.CoreVersion) <= 0
			},
		},
		{
			name:  "an invalid acknowledgement",
			match: func(v acknowledgeVector) bool { return !contract.Valid(v.Ack) },
		},
		{
			name:  "an invalid contract version",
			match: func(v acknowledgeVector) bool { return !contract.Valid(v.ContractVersion) },
		},
		{
			name:  "an invalid core version",
			match: func(v acknowledgeVector) bool { return !contract.Valid(v.CoreVersion) },
		},
	}
	acks := loadVectors(t).Acknowledge
	for _, c := range cases {
		assert.True(t, slices.ContainsFunc(acks, c.match), "no acknowledge vector for %s", c.name)
	}
}
