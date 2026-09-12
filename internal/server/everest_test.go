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

package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTrimWebhookErrorText(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		description string
		input       string
		expected    string
	}{
		{
			description: "legacy v1 monitoringconfig webhook prefix is stripped",
			input:       `admission webhook "vmonitoringconfig-v1alpha1.everest.percona.com" denied the request: field is required`,
			expected:    "field is required",
		},
		{
			description: "legacy v1 loadbalancerconfig webhook prefix is stripped",
			input:       `admission webhook "vloadbalancerconfig-v1alpha1.everest.percona.com" denied the request: invalid config`,
			expected:    "invalid config",
		},
		{
			description: "legacy v1 databasecluster webhook prefix is stripped",
			input:       `admission webhook "vdatabasecluster-v1alpha1.everest.percona.com" denied the request: spec is invalid`,
			expected:    "spec is invalid",
		},
		{
			description: "v2 monitoring.openeverest.io webhook prefix is stripped",
			input:       `admission webhook "vmonitoringconfig-v1alpha1.monitoring.openeverest.io" denied the request: field is required`,
			expected:    "field is required",
		},
		{
			description: "v2 kb.io webhook prefix is stripped",
			input:       `admission webhook "vmonitoringconfig-v1alpha1.kb.io" denied the request: field is required`,
			expected:    "field is required",
		},
		{
			description: "an admission webhook name never seen before is still stripped",
			input:       `admission webhook "vsomefutureresource-v2.some.new.domain" denied the request: nope`,
			expected:    "nope",
		},
		{
			description: "text with no admission webhook prefix is left untouched",
			input:       "some other backend error with no webhook boilerplate",
			expected:    "some other backend error with no webhook boilerplate",
		},
		{
			description: "empty string stays empty",
			input:       "",
			expected:    "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, trimWebhookErrorText(tc.input))
		})
	}
}
