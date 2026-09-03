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

package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

func maintenanceContext(t *testing.T, m *v1alpha1.MaintenanceSpec) *Context {
	t.Helper()
	in := &v1alpha1.Instance{}
	in.Spec.Maintenance = m
	return NewContext(t.Context(), nil, in, "test-provider")
}

func TestRequestMaintenance_Gating(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		maintenance *v1alpha1.MaintenanceSpec
		severity    MaintenanceSeverity
		token       string
		want        bool
	}{
		{
			name:     "nil maintenance holds a rolling restart",
			severity: MaintenanceRollingRestart,
			token:    "upgrade-to-0.3",
			want:     false,
		},
		{
			name:     "nil maintenance auto-applies non-disruptive",
			severity: MaintenanceNonDisruptive,
			token:    "upgrade-to-0.3",
			want:     true,
		},
		{
			name:        "tolerance at rolling restart auto-applies rolling restart",
			maintenance: &v1alpha1.MaintenanceSpec{AutoApproveUpTo: v1alpha1.MaintenanceRollingRestart},
			severity:    MaintenanceRollingRestart,
			token:       "upgrade-to-0.3",
			want:        true,
		},
		{
			name:        "tolerance at rolling restart holds downtime",
			maintenance: &v1alpha1.MaintenanceSpec{AutoApproveUpTo: v1alpha1.MaintenanceRollingRestart},
			severity:    MaintenanceDowntime,
			token:       "upgrade-to-0.3",
			want:        false,
		},
		{
			name:        "matching token approves above tolerance",
			maintenance: &v1alpha1.MaintenanceSpec{Approved: "upgrade-to-0.3"},
			severity:    MaintenanceDowntime,
			token:       "upgrade-to-0.3",
			want:        true,
		},
		{
			name:        "stale token from an earlier action does not authorize a new one",
			maintenance: &v1alpha1.MaintenanceSpec{Approved: "upgrade-to-0.3"},
			severity:    MaintenanceRollingRestart,
			token:       "rotate-certs-2026H2",
			want:        false,
		},
		{
			name:        "empty token never matches an empty approval",
			maintenance: &v1alpha1.MaintenanceSpec{},
			severity:    MaintenanceRollingRestart,
			token:       "",
			want:        false,
		},
		{
			name:        "unknown severity is held even at the widest tolerance",
			maintenance: &v1alpha1.MaintenanceSpec{AutoApproveUpTo: v1alpha1.MaintenanceDowntime},
			severity:    MaintenanceSeverity("Catastrophic"),
			token:       "upgrade-to-0.3",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := maintenanceContext(t, tt.maintenance)

			got := c.RequestMaintenance(tt.token, "some action", tt.severity)

			assert.Equal(t, tt.want, got)
			if tt.want {
				assert.Empty(t, c.GetPendingMaintenance(), "approved actions must not be recorded as pending")
			} else {
				require.Len(t, c.GetPendingMaintenance(), 1)
				held := c.GetPendingMaintenance()[0]
				assert.Equal(t, tt.token, held.ApprovalToken)
				assert.Equal(t, tt.severity, held.Severity)
				assert.Equal(t, "some action", held.Description)
			}
		})
	}
}

func TestRequestMaintenance_MultipleActionsKeepOrder(t *testing.T) {
	t.Parallel()

	c := maintenanceContext(t, nil)

	assert.False(t, c.RequestMaintenance("upgrade-to-0.3", "converge", MaintenanceRollingRestart))
	assert.False(t, c.RequestMaintenance("rotate-certs-2026H2", "rotate certs", MaintenanceDowntime))

	pending := c.GetPendingMaintenance()
	require.Len(t, pending, 2)
	assert.Equal(t, "upgrade-to-0.3", pending[0].ApprovalToken)
	assert.Equal(t, "rotate-certs-2026H2", pending[1].ApprovalToken)
}

func TestRequestMaintenance_ReArmsPerOccurrence(t *testing.T) {
	t.Parallel()

	// The owner approved the 0.3 convergence; a later 0.5 convergence is a
	// new occurrence with a new token and must be held again.
	c := maintenanceContext(t, &v1alpha1.MaintenanceSpec{Approved: "upgrade-to-0.3"})

	assert.True(t, c.RequestMaintenance("upgrade-to-0.3", "converge to 0.3", MaintenanceRollingRestart))
	assert.False(t, c.RequestMaintenance("upgrade-to-0.5", "converge to 0.5", MaintenanceRollingRestart))
}
