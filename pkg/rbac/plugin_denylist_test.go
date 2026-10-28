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

package rbac

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/openeverest/openeverest/v2/pkg/plugintoken"
)

func TestIsPluginWriteDenied(t *testing.T) {
	t.Parallel()

	plugin := plugintoken.SubjectPrefix + "pmm"
	testCases := []struct {
		desc     string
		subject  string
		resource string
		action   string
		want     bool
	}{
		{"plugin creates a monitoring destination", plugin, ResourceMonitoringDestinations, ActionCreate, true},
		{"plugin updates a monitoring destination", plugin, ResourceMonitoringDestinations, ActionUpdate, true},
		{"plugin deletes a monitoring destination", plugin, ResourceMonitoringDestinations, ActionDelete, true},
		{"plugin holds every verb on monitoring destinations", plugin, ResourceMonitoringDestinations, ActionAll, true},
		{"plugin writes a monitoring class", plugin, ResourceMonitoringClasses, ActionCreate, true},
		{"plugin writes a monitoring binding", plugin, ResourceMonitoringBindings, ActionUpdate, true},
		{"plugin reads a monitoring destination", plugin, ResourceMonitoringDestinations, ActionRead, false},
		{"plugin reads a monitoring binding", plugin, ResourceMonitoringBindings, ActionRead, false},
		{"user creates a monitoring destination", "alice", ResourceMonitoringDestinations, ActionCreate, false},
		{"plugin creates an instance", plugin, ResourceInstances, ActionCreate, false},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, IsPluginWriteDenied(tc.subject, tc.resource, tc.action))
		})
	}
}
