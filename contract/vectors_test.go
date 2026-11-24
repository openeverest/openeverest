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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openeverest/openeverest/v2/contract"
)

type vectors struct {
	Order   []string      `json:"order"`
	Invalid []string      `json:"invalid"`
	Check   []checkVector `json:"check"`
}

type checkVector struct {
	Ext    string                 `json:"ext"`
	Min    string                 `json:"min"`
	Cur    string                 `json:"cur"`
	Status metav1.ConditionStatus `json:"status"`
	Reason string                 `json:"reason"`
}

func loadVectors(t *testing.T) vectors {
	t.Helper()

	dec := json.NewDecoder(bytes.NewReader(contract.OrderVectors))
	dec.DisallowUnknownFields()
	var v vectors
	require.NoError(t, dec.Decode(&v))
	require.NotEmpty(t, v.Order)
	require.NotEmpty(t, v.Invalid)
	require.NotEmpty(t, v.Check)
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
