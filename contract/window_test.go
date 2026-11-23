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
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openeverest/openeverest/v2/contract"
)

func TestWindowFromAnnotations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		annotations map[string]string
		want        contract.Window
		wantOK      bool
	}{
		{
			name: "both",
			annotations: map[string]string{
				contract.MinVersionAnnotation: "2.0.0-alpha.1",
				contract.VersionAnnotation:    "2.0.0-alpha.1.dev",
				"other":                       "ignored",
			},
			want:   contract.Window{Min: "2.0.0-alpha.1", Current: "2.0.0-alpha.1.dev"},
			wantOK: true,
		},
		{
			name:        "only the current version",
			annotations: map[string]string{contract.VersionAnnotation: "2.0.0"},
			want:        contract.Window{Current: "2.0.0"},
			wantOK:      true,
		},
		{
			name:        "empty values",
			annotations: map[string]string{contract.MinVersionAnnotation: "", contract.VersionAnnotation: ""},
			want:        contract.Window{},
			wantOK:      true,
		},
		{
			name:        "none",
			annotations: map[string]string{contract.CoreVersionKey: "2.0.0-alpha.1"},
		},
		{
			name: "nil",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := contract.WindowFromAnnotations(tt.annotations)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestWindowFromAnnotationsWithOneValueIsInvalid(t *testing.T) {
	t.Parallel()

	w, ok := contract.WindowFromAnnotations(map[string]string{contract.VersionAnnotation: "2.0.0"})
	assert.True(t, ok)
	assert.Equal(t, contract.Result{Status: metav1.ConditionUnknown, Reason: contract.ReasonInvalidVersion}, w.Check("2.0.0"))
}

func TestIntersect(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ws   []contract.Window
		want contract.Window
	}{
		{
			name: "none",
			want: contract.Window{},
		},
		{
			name: "one",
			ws:   []contract.Window{{Min: "2.0.0", Current: "2.1.0"}},
			want: contract.Window{Min: "2.0.0", Current: "2.1.0"},
		},
		{
			name: "highest minimum and lowest current",
			ws: []contract.Window{
				{Min: "2.0.0", Current: "2.2.0"},
				{Min: "2.1.0", Current: "2.3.0"},
				{Min: "2.0.0-alpha.1", Current: "2.2.0-alpha.1.dev"},
			},
			want: contract.Window{Min: "2.1.0", Current: "2.2.0-alpha.1.dev"},
		},
		{
			name: "precedence, not bytes",
			ws: []contract.Window{
				{Min: "2.0.0-alpha.9", Current: "2.10.0"},
				{Min: "2.0.0-alpha.10", Current: "2.9.0"},
			},
			want: contract.Window{Min: "2.0.0-alpha.10", Current: "2.9.0"},
		},
		{
			name: "an invalid minimum carries over",
			ws: []contract.Window{
				{Min: "2.0.0", Current: "2.1.0"},
				{Min: "", Current: "2.1.0"},
				{Min: "2.0.1", Current: "2.1.0"},
			},
			want: contract.Window{Min: "", Current: "2.1.0"},
		},
		{
			name: "an invalid current carries over",
			ws: []contract.Window{
				{Min: "2.0.0", Current: "2.1.0"},
				{Min: "2.0.0", Current: "latest"},
			},
			want: contract.Window{Min: "2.0.0", Current: "latest"},
		},
		{
			name: "windows that do not overlap",
			ws: []contract.Window{
				{Min: "2.0.0", Current: "2.1.0"},
				{Min: "2.2.0", Current: "2.3.0"},
			},
			want: contract.Window{Min: "2.2.0", Current: "2.1.0"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, contract.Intersect(tt.ws...))
		})
	}
}

func TestIntersectedWindowCheck(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ws   []contract.Window
		ext  string
		want contract.Result
	}{
		{
			name: "a stamp only the newer CRD serves",
			ws:   []contract.Window{{Min: "2.0.0", Current: "2.1.0"}, {Min: "2.0.0", Current: "2.2.0"}},
			ext:  "2.2.0",
			want: contract.Result{Status: metav1.ConditionFalse, Reason: contract.ReasonCoreTooOld},
		},
		{
			name: "a stamp only the older CRD serves",
			ws:   []contract.Window{{Min: "2.0.0", Current: "2.2.0"}, {Min: "2.1.0", Current: "2.2.0"}},
			ext:  "2.0.0",
			want: contract.Result{Status: metav1.ConditionFalse, Reason: contract.ReasonExtensionTooOld},
		},
		{
			name: "a stamp every CRD serves",
			ws:   []contract.Window{{Min: "2.0.0", Current: "2.2.0"}, {Min: "2.1.0", Current: "2.3.0"}},
			ext:  "2.1.0",
			want: contract.Result{Status: metav1.ConditionTrue, Reason: contract.ReasonSupported},
		},
		{
			name: "windows that do not overlap",
			ws:   []contract.Window{{Min: "2.0.0", Current: "2.1.0"}, {Min: "2.2.0", Current: "2.3.0"}},
			ext:  "2.1.0",
			want: contract.Result{Status: metav1.ConditionUnknown, Reason: contract.ReasonCoreVersionSkew},
		},
		{
			name: "no window",
			ext:  "2.1.0",
			want: contract.Result{Status: metav1.ConditionUnknown, Reason: contract.ReasonInvalidVersion},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, contract.Intersect(tt.ws...).Check(tt.ext))
		})
	}
}
