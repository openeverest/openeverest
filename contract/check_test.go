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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openeverest/openeverest/v2/contract"
)

// assertUserFacing checks the rules every Message follows: it never names
// the contract, which users do not know about, and never shows a
// development value, which no user can install.
func assertUserFacing(t *testing.T, msg string) {
	t.Helper()

	assert.NotEmpty(t, msg)
	assert.NotContains(t, strings.ToLower(msg), "contract")
	for word := range strings.FieldsSeq(msg) {
		assert.False(t, contract.IsDevelopment(strings.Trim(word, `"';,`)), "%q shows the development value %s", msg, word)
	}
}

func TestGeneratedWindow(t *testing.T) {
	t.Parallel()

	assert.True(t, contract.Valid(contract.Version), "Version %q", contract.Version)
	assert.True(t, contract.Valid(contract.MinVersion), "MinVersion %q", contract.MinVersion)
	assert.Equal(t, contract.ReasonSupported, contract.Check(contract.Version, contract.MinVersion, contract.Version).Reason)
	assert.Equal(t, contract.ReasonSupported, contract.Check(contract.MinVersion, contract.MinVersion, contract.Version).Reason)
}

func TestValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		v    string
		want bool
	}{
		{v: "2.0.0", want: true},
		{v: "2.0.0-alpha.1.next", want: true},
		{v: "2.1.0-alpha.0", want: true},
		{v: "2.0.0-alpha-1", want: true},
		{v: "2.0.0+build.01", want: true},
		{v: "2.0.0-rc.1+build.5", want: true},
		{v: "v2.0.0", want: false},
		{v: "2.0", want: false},
		{v: "2.0.0-01", want: false},
		{v: "", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.v, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, contract.Valid(tt.v))
		})
	}
}

func TestCompare(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		a, b string
		want int
	}{
		{name: "build metadata is ignored", a: "2.0.0+build.1", b: "2.0.0+build.2", want: 0},
		{name: "build metadata does not make a release newer", a: "2.0.0+build.1", b: "2.0.1", want: -1},
		{name: "invalid sorts before valid", a: "latest", b: "0.0.0", want: -1},
		{name: "valid sorts after invalid", a: "0.0.0", b: "v2.0.0", want: 1},
		{name: "invalid values compare by bytes", a: "latest", b: "v2.0.0", want: -1},
		{name: "equal invalid values", a: "", b: "", want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, contract.Compare(tt.a, tt.b))
		})
	}
}

func TestAcknowledge(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in, want contract.Result
	}{
		{
			in:   contract.Result{Status: metav1.ConditionFalse, Reason: contract.ReasonCoreTooOld},
			want: contract.Result{Status: metav1.ConditionFalse, Reason: contract.ReasonCoreTooOldAcknowledged},
		},
		{
			in:   contract.Result{Status: metav1.ConditionFalse, Reason: contract.ReasonExtensionTooOld},
			want: contract.Result{Status: metav1.ConditionFalse, Reason: contract.ReasonExtensionTooOldAcknowledged},
		},
		{
			in:   contract.Result{Status: metav1.ConditionTrue, Reason: contract.ReasonSupported},
			want: contract.Result{Status: metav1.ConditionTrue, Reason: contract.ReasonSupported},
		},
		{
			in:   contract.Result{Status: metav1.ConditionUnknown, Reason: contract.ReasonInvalidVersion},
			want: contract.Result{Status: metav1.ConditionUnknown, Reason: contract.ReasonInvalidVersion},
		},
		{
			in:   contract.Result{Status: metav1.ConditionFalse, Reason: contract.ReasonCoreTooOldAcknowledged},
			want: contract.Result{Status: metav1.ConditionFalse, Reason: contract.ReasonCoreTooOldAcknowledged},
		},
	}
	for _, tt := range tests {
		t.Run(tt.in.Reason, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, contract.Acknowledge(tt.in))
		})
	}
}

func TestMessage(t *testing.T) {
	t.Parallel()

	released := contract.Window{Min: "2.0.0", Current: "2.1.0"}
	unreleased := contract.Window{Min: "2.1.0-alpha.1.next", Current: "2.1.0-alpha.1.next"}
	tests := []struct {
		name string
		ext  string
		w    contract.Window
		ack  bool
		want string
	}{
		{
			name: "supported",
			ext:  "2.1.0", w: released,
			want: "works with this version of OpenEverest",
		},
		{
			name: "core too old",
			ext:  "2.2.0", w: released,
			want: "needs OpenEverest 2.2.0 or later",
		},
		{
			name: "core too old for a stamp of an unreleased core",
			ext:  "2.2.0-alpha.0", w: released,
			want: "needs a newer, unreleased OpenEverest build; install a version of it built for a released OpenEverest",
		},
		{
			name: "core too old for a stamp of a dev build",
			ext:  "2.0.0-dev.4", w: contract.Window{Min: "2.0.0-alpha.1", Current: "2.0.0-alpha.2"},
			want: "needs OpenEverest 2.0.0-dev.4 or later",
		},
		{
			name: "core too old, acknowledged",
			ext:  "2.2.0", w: released, ack: true,
			want: "needs OpenEverest 2.2.0 or later; allowed anyway because the incompatibility was acknowledged",
		},
		{
			name: "extension too old",
			ext:  "1.16.0", w: released,
			want: "was built for an older OpenEverest; update it to a release built for OpenEverest 2.0.0 or later",
		},
		{
			name: "extension too old for an unreleased core",
			ext:  "2.0.0", w: unreleased,
			want: "was built for an older OpenEverest; update it to a version built for this unreleased OpenEverest build",
		},
		{
			name: "extension too old, acknowledged",
			ext:  "1.16.0", w: released, ack: true,
			want: "was built for an older OpenEverest; update it to a release built for OpenEverest 2.0.0 or later; " +
				"allowed anyway because the incompatibility was acknowledged",
		},
		{
			name: "unversioned",
			ext:  "", w: released,
			want: "does not declare the OpenEverest it was built for, so its compatibility cannot be checked",
		},
		{
			name: "invalid stamp",
			ext:  "v2.1.0", w: released,
			want: `declares an invalid OpenEverest version "v2.1.0", so its compatibility cannot be checked`,
		},
		{
			name: "invalid window",
			ext:  "2.1.0", w: contract.Window{Min: "2.0.0"},
			want: "cannot be checked for compatibility because this OpenEverest publishes invalid supported versions",
		},
		{
			name: "core version skew",
			ext:  "2.1.0", w: contract.Window{Min: "2.1.0", Current: "2.0.0"},
			want: "cannot be checked for compatibility while the parts of this OpenEverest disagree on the versions " +
				"they support, as they do during an upgrade",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := tt.w.Check(tt.ext)
			if tt.ack {
				r = contract.Acknowledge(r)
			}
			msg := contract.Message(r, tt.ext, tt.w)
			assert.Equal(t, tt.want, msg)
			assertUserFacing(t, msg)
		})
	}
}

func TestMessageOfAnUnknownReason(t *testing.T) {
	t.Parallel()

	msg := contract.Message(contract.Result{Status: metav1.ConditionUnknown, Reason: "Whatever"}, "2.0.0", contract.Window{})
	assert.Equal(t, "cannot be checked for compatibility (Whatever)", msg)
}
