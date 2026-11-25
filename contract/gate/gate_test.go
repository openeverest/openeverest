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

package gate_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openeverest/openeverest/v2/contract"
	"github.com/openeverest/openeverest/v2/contract/gate"
)

func window(minimum, current string) gate.Core {
	return gate.Core{Window: contract.Window{Min: minimum, Current: current}, HasWindow: true}
}

func released(minimum, current, version string) gate.Core {
	c := window(minimum, current)
	c.Version = version
	return c
}

func TestCoreFromCRD(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		labels      map[string]string
		annotations map[string]string
		want        gate.Core
	}{
		{
			name: "no metadata",
			want: gate.Core{},
		},
		{
			name:   "window, release and acknowledgements",
			labels: map[string]string{contract.CoreVersionKey: "2.1.1"},
			annotations: map[string]string{
				contract.MinVersionAnnotation:             "2.0.0",
				contract.VersionAnnotation:                "2.1.0",
				contract.AcknowledgedExtensionsAnnotation: "psmdb@2.1.0",
			},
			want: gate.Core{
				Window:                 contract.Window{Min: "2.0.0", Current: "2.1.0"},
				HasWindow:              true,
				Version:                "2.1.1",
				AcknowledgedExtensions: "psmdb@2.1.0",
			},
		},
		{
			name:        "half a window",
			annotations: map[string]string{contract.VersionAnnotation: "2.1.0"},
			want:        gate.Core{Window: contract.Window{Current: "2.1.0"}, HasWindow: true},
		},
		{
			name:        "the core release in an annotation is not read",
			annotations: map[string]string{contract.CoreVersionKey: "2.1.1"},
			want:        gate.Core{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, gate.CoreFromCRD(tt.labels, tt.annotations))
		})
	}
}

func TestDecide(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		linked     string
		root       gate.Root
		core       gate.Core
		start      bool
		reason     string
		status     metav1.ConditionStatus
		ack        bool
		waitingFor string
		message    string
	}{
		{
			name:    "the build that last started starts again without a window",
			linked:  "2.1.0",
			root:    gate.Root{Observed: "2.1.0"},
			start:   true,
			reason:  gate.ReasonAlreadyRunning,
			message: "this build (for OpenEverest 2.1.0) is the one that last started, so it starts again without a compatibility check",
		},
		{
			name:   "the build that last started starts again on a core rolled back below it",
			linked: "2.1.0",
			root:   gate.Root{Observed: "2.1.0"},
			core:   window("2.0.0", "2.0.0"),
			start:  true,
			reason: gate.ReasonAlreadyRunning,
		},
		{
			name:    "a first build inside the window",
			linked:  "2.1.0",
			core:    window("2.0.0", "2.1.0"),
			start:   true,
			reason:  contract.ReasonSupported,
			status:  metav1.ConditionTrue,
			message: "a new build (for OpenEverest 2.1.0) starts: it works with this version of OpenEverest",
		},
		{
			name:   "a new build inside the window",
			linked: "2.1.0",
			root:   gate.Root{Observed: "2.0.0"},
			core:   window("2.0.0", "2.2.0"),
			start:  true,
			reason: contract.ReasonSupported,
			status: metav1.ConditionTrue,
		},
		{
			name:       "a new build for a newer core",
			linked:     "2.2.0",
			root:       gate.Root{Observed: "2.1.0"},
			core:       window("2.0.0", "2.1.0"),
			reason:     contract.ReasonCoreTooOld,
			status:     metav1.ConditionFalse,
			waitingFor: "2.2.0",
			message:    "a new build (for OpenEverest 2.2.0) is waiting until OpenEverest 2.2.0 is installed",
		},
		{
			name:       "a core that publishes no window",
			linked:     "2.1.0",
			root:       gate.Root{Observed: "2.0.0"},
			core:       gate.Core{Version: "2.1.0"},
			reason:     contract.ReasonCoreTooOld,
			status:     metav1.ConditionFalse,
			waitingFor: "2.1.0",
			message:    "a new build (for OpenEverest 2.1.0) is waiting until OpenEverest 2.1.0 is installed; the OpenEverest in this cluster does not publish the versions it supports, which OpenEverest 2.1.0 does",
		},
		{
			name:       "no core and no root object",
			linked:     "2.1.0",
			reason:     contract.ReasonCoreTooOld,
			status:     metav1.ConditionFalse,
			waitingFor: "2.1.0",
		},
		{
			name:    "a build for an unreleased core",
			linked:  "2.1.0-alpha.0",
			core:    window("2.0.0", "2.0.1"),
			reason:  contract.ReasonCoreTooOld,
			status:  metav1.ConditionFalse,
			message: "a new build (for an unreleased OpenEverest) is waiting until the installed OpenEverest supports it: it needs a newer, unreleased OpenEverest build; install a version of it built for a released OpenEverest",
		},
		{
			name:    "a build too old for the core",
			linked:  "2.0.0",
			root:    gate.Root{Observed: "2.1.0"},
			core:    window("2.1.0", "2.2.0"),
			reason:  contract.ReasonExtensionTooOld,
			status:  metav1.ConditionFalse,
			message: "a new build (for OpenEverest 2.0.0) is waiting until the installed OpenEverest supports it: it was built for an older OpenEverest; update it to a release built for OpenEverest 2.1.0 or later",
		},
		{
			name:    "a malformed window",
			linked:  "2.1.0",
			core:    window("2.0", "2.1.0"),
			reason:  contract.ReasonInvalidVersion,
			status:  metav1.ConditionUnknown,
			message: "a new build (for OpenEverest 2.1.0) is waiting until the installed OpenEverest supports it: it cannot be checked for compatibility because this OpenEverest publishes invalid supported versions",
		},
		{
			name:    "half a window",
			linked:  "2.1.0",
			core:    window("", "2.1.0"),
			reason:  contract.ReasonInvalidVersion,
			status:  metav1.ConditionUnknown,
			message: "a new build (for OpenEverest 2.1.0) is waiting until the installed OpenEverest supports it: it cannot be checked for compatibility because this OpenEverest publishes invalid supported versions",
		},
		{
			name:    "a malformed build version",
			linked:  "v2.1.0",
			core:    window("2.0.0", "2.1.0"),
			reason:  contract.ReasonInvalidVersion,
			status:  metav1.ConditionUnknown,
			message: `a new build is waiting until the installed OpenEverest supports it: it declares an invalid OpenEverest version "v2.1.0", so its compatibility cannot be checked`,
		},
		{
			name:   "a window that serves nothing",
			linked: "2.1.0",
			core:   window("2.2.0", "2.1.0"),
			reason: contract.ReasonCoreVersionSkew,
			status: metav1.ConditionUnknown,
		},
		{
			name:    "acknowledged on the root object",
			linked:  "2.2.0",
			root:    gate.Root{Observed: "2.1.0", Acknowledged: "2.1.1"},
			core:    released("2.0.0", "2.1.0", "2.1.1"),
			start:   true,
			reason:  contract.ReasonCoreTooOldAcknowledged,
			status:  metav1.ConditionFalse,
			ack:     true,
			message: "a new build (for OpenEverest 2.2.0) starts: it needs OpenEverest 2.2.0 or later; allowed anyway because the incompatibility was acknowledged",
		},
		{
			name:   "acknowledged in core's list",
			linked: "2.0.0",
			core: func() gate.Core {
				c := released("2.1.0", "2.1.0", "2.1.0")
				c.AcknowledgedExtensions = "other@2.1.0, psmdb@2.1.0"
				return c
			}(),
			start:  true,
			reason: contract.ReasonExtensionTooOldAcknowledged,
			status: metav1.ConditionFalse,
			ack:    true,
		},
		{
			name:       "an acknowledgement the contract has moved past",
			linked:     "2.3.0",
			root:       gate.Root{Acknowledged: "2.1.0"},
			core:       released("2.0.0", "2.2.0", "2.2.0"),
			reason:     contract.ReasonCoreTooOld,
			status:     metav1.ConditionFalse,
			waitingFor: "2.3.0",
		},
		{
			name:       "an acknowledgement of a core not installed yet",
			linked:     "2.3.0",
			root:       gate.Root{Acknowledged: "2.2.0"},
			core:       released("2.0.0", "2.1.0", "2.1.0"),
			reason:     contract.ReasonCoreTooOld,
			status:     metav1.ConditionFalse,
			waitingFor: "2.3.0",
		},
		{
			name:       "an acknowledgement on a core that publishes no release",
			linked:     "2.2.0",
			root:       gate.Root{Acknowledged: "2.1.0"},
			core:       window("2.0.0", "2.1.0"),
			reason:     contract.ReasonCoreTooOld,
			status:     metav1.ConditionFalse,
			waitingFor: "2.2.0",
		},
		{
			name:       "an acknowledgement on a core that publishes no window",
			linked:     "2.2.0",
			root:       gate.Root{Acknowledged: "2.1.0"},
			core:       gate.Core{Version: "2.1.0", AcknowledgedExtensions: "psmdb@2.1.0"},
			reason:     contract.ReasonCoreTooOld,
			status:     metav1.ConditionFalse,
			waitingFor: "2.2.0",
		},
		{
			name:   "core's list names another extension",
			linked: "2.2.0",
			core: func() gate.Core {
				c := released("2.0.0", "2.1.0", "2.1.0")
				c.AcknowledgedExtensions = "other@2.1.0"
				return c
			}(),
			reason:     contract.ReasonCoreTooOld,
			status:     metav1.ConditionFalse,
			waitingFor: "2.2.0",
		},
		{
			name:   "core's list does not parse",
			linked: "2.2.0",
			core: func() gate.Core {
				c := released("2.0.0", "2.1.0", "2.1.0")
				c.AcknowledgedExtensions = "psmdb@2.1.0,psmdb@2.1.0"
				return c
			}(),
			reason:     contract.ReasonCoreTooOld,
			status:     metav1.ConditionFalse,
			waitingFor: "2.2.0",
		},
		{
			name:    "an acknowledged malformed window",
			linked:  "2.1.0",
			root:    gate.Root{Acknowledged: "2.1.0"},
			core:    released("2.0", "2.1.0", "2.1.0"),
			start:   true,
			reason:  contract.ReasonInvalidVersion,
			status:  metav1.ConditionUnknown,
			ack:     true,
			message: "a new build (for OpenEverest 2.1.0) starts: it cannot be checked for compatibility because this OpenEverest publishes invalid supported versions; allowed anyway because the incompatibility was acknowledged",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d := gate.Decide("psmdb", tt.linked, tt.root, tt.core)
			assert.Equal(t, tt.start, d.Start, d.Message)
			assert.Equal(t, tt.reason, d.Reason)
			assert.Equal(t, tt.status, d.Result.Status)
			if tt.reason != gate.ReasonAlreadyRunning {
				assert.Equal(t, tt.reason, d.Result.Reason)
			}
			assert.Equal(t, tt.ack, d.Acknowledged)
			assert.Equal(t, tt.waitingFor, d.WaitingFor)
			assert.NotEmpty(t, d.Message)
			if tt.message != "" {
				assert.Equal(t, tt.message, d.Message)
			}
		})
	}
}

type vectors struct {
	Check []struct {
		Ext    string                 `json:"ext"`
		Min    string                 `json:"min"`
		Cur    string                 `json:"cur"`
		Status metav1.ConditionStatus `json:"status"`
		Reason string                 `json:"reason"`
	} `json:"check"`
	Acknowledge []struct {
		Ack             string `json:"ack"`
		ContractVersion string `json:"contractVersion"`
		CoreVersion     string `json:"coreVersion"`
		Holds           bool   `json:"holds"`
	} `json:"acknowledge"`
}

func loadVectors(t *testing.T) vectors {
	t.Helper()

	var v vectors
	require.NoError(t, json.NewDecoder(bytes.NewReader(contract.OrderVectors)).Decode(&v))
	require.NotEmpty(t, v.Check)
	require.NotEmpty(t, v.Acknowledge)
	return v
}

// TestDecideCheckVectors runs the shared check cases through a first
// rollout: the gate lets a build start exactly when the check supports it.
func TestDecideCheckVectors(t *testing.T) {
	t.Parallel()

	for _, v := range loadVectors(t).Check {
		d := gate.Decide("psmdb", v.Ext, gate.Root{}, window(v.Min, v.Cur))
		assert.Equal(t, v.Status == metav1.ConditionTrue, d.Start, "ext %q in [%q, %q]", v.Ext, v.Min, v.Cur)
		assert.Equal(t, contract.Result{Status: v.Status, Reason: v.Reason}, d.Result, "ext %q in [%q, %q]", v.Ext, v.Min, v.Cur)
	}
}

// TestDecideAcknowledgeVectors runs the shared acknowledgement cases for a
// build no window supports: it starts exactly when the acknowledgement
// holds, whether the extension chart or the core chart wrote it.
func TestDecideAcknowledgeVectors(t *testing.T) {
	t.Parallel()

	const linked = "999.0.0"
	for _, v := range loadVectors(t).Acknowledge {
		core := released(v.ContractVersion, v.ContractVersion, v.CoreVersion)
		d := gate.Decide("psmdb", linked, gate.Root{Acknowledged: v.Ack}, core)
		assert.Equal(t, v.Holds, d.Start, "on the root object: ack %q on %q, contract %q", v.Ack, v.CoreVersion, v.ContractVersion)
		assert.Equal(t, v.Holds, d.Acknowledged)

		if strings.TrimSpace(v.Ack) != v.Ack {
			continue // core's list ignores the spaces around its entries
		}
		core.AcknowledgedExtensions = "psmdb@" + v.Ack
		d = gate.Decide("psmdb", linked, gate.Root{}, core)
		assert.Equal(t, v.Holds, d.Start, "in core's list: ack %q on %q, contract %q", v.Ack, v.CoreVersion, v.ContractVersion)
	}
}
