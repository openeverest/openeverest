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

package reconciler

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
)

func TestMaintenanceBreaker(t *testing.T) {
	t.Parallel()

	nn := types.NamespacedName{Namespace: "default", Name: "my-db"}

	t.Run("trips only at the threshold", func(t *testing.T) {
		t.Parallel()

		var b maintenanceBreaker
		for range maintenanceBreakerThreshold - 1 {
			b.recordFailure(nn, "upgrade-to-0.3", []string{"upgrade-to-0.3"})
		}
		assert.Empty(t, b.blockedTokens(nn, "upgrade-to-0.3"))

		b.recordFailure(nn, "upgrade-to-0.3", []string{"upgrade-to-0.3"})
		assert.Equal(t, []string{"upgrade-to-0.3"}, b.blockedTokens(nn, "upgrade-to-0.3"))
	})

	t.Run("changing the approval resets the counts", func(t *testing.T) {
		t.Parallel()

		var b maintenanceBreaker
		for range maintenanceBreakerThreshold {
			b.recordFailure(nn, "upgrade-to-0.3", []string{"upgrade-to-0.3"})
		}
		require.NotEmpty(t, b.blockedTokens(nn, "upgrade-to-0.3"))

		// The user clears the approval (clear-stuck), then re-approves.
		assert.Empty(t, b.blockedTokens(nn, ""))
		assert.Empty(t, b.blockedTokens(nn, "upgrade-to-0.3"))
	})

	t.Run("successful sync resets the counts", func(t *testing.T) {
		t.Parallel()

		var b maintenanceBreaker
		for range maintenanceBreakerThreshold {
			b.recordFailure(nn, "upgrade-to-0.3", []string{"upgrade-to-0.3"})
		}
		b.reset(nn)
		assert.Empty(t, b.blockedTokens(nn, "upgrade-to-0.3"))
	})

	t.Run("failures without approved actions are not counted", func(t *testing.T) {
		t.Parallel()

		var b maintenanceBreaker
		for range maintenanceBreakerThreshold + 1 {
			b.recordFailure(nn, "upgrade-to-0.3", nil)
		}
		assert.Empty(t, b.blockedTokens(nn, "upgrade-to-0.3"))
	})

	t.Run("instances are tracked independently", func(t *testing.T) {
		t.Parallel()

		var b maintenanceBreaker
		other := types.NamespacedName{Namespace: "default", Name: "other-db"}
		for range maintenanceBreakerThreshold {
			b.recordFailure(nn, "upgrade-to-0.3", []string{"upgrade-to-0.3"})
		}
		assert.Empty(t, b.blockedTokens(other, "upgrade-to-0.3"))
	})
}

func TestRequestMaintenance_BreakerHoldsApprovedAction(t *testing.T) {
	t.Parallel()

	in := &corev1alpha1.Instance{}
	in.Spec.Maintenance = &corev1alpha1.MaintenanceSpec{Approved: "upgrade-to-0.3"}
	syncCtx := controller.NewContext(t.Context(), nil, in, "test-provider")
	syncCtx.BlockMaintenance("upgrade-to-0.3")

	approved := syncCtx.RequestMaintenance("upgrade-to-0.3", "brief rolling restart", controller.MaintenanceRollingRestart)

	assert.False(t, approved, "a token with exhausted retries must be held despite approval")
	assert.True(t, syncCtx.MaintenanceBreakerHeld())
	require.Len(t, syncCtx.GetPendingMaintenance(), 1)
	assert.Empty(t, syncCtx.GetApprovedMaintenance())
}

func TestFlushPendingMaintenance_BreakerReason(t *testing.T) {
	t.Parallel()

	in := &corev1alpha1.Instance{ObjectMeta: metav1.ObjectMeta{Name: "my-db", Namespace: "default"}}
	in.Spec.Maintenance = &corev1alpha1.MaintenanceSpec{Approved: "upgrade-to-0.3"}
	syncCtx := controller.NewContext(t.Context(), nil, in.DeepCopy(), "test-provider")
	syncCtx.BlockMaintenance("upgrade-to-0.3")
	require.False(t, syncCtx.RequestMaintenance("upgrade-to-0.3", "brief rolling restart", controller.MaintenanceRollingRestart))

	flushPendingMaintenance(syncCtx, in)

	cond := findMaintenanceCondition(in)
	require.NotNil(t, cond)
	assert.Equal(t, metav1.ConditionTrue, cond.Status)
	assert.Equal(t, corev1alpha1.ReasonRetriesExhausted, cond.Reason)
}
