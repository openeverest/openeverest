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

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
)

func findMaintenanceCondition(in *corev1alpha1.Instance) *metav1.Condition {
	for i := range in.Status.Conditions {
		if in.Status.Conditions[i].Type == corev1alpha1.ConditionMaintenancePending {
			return &in.Status.Conditions[i]
		}
	}
	return nil
}

func TestFlushPendingMaintenance(t *testing.T) {
	t.Parallel()

	t.Run("held actions land on status with condition True", func(t *testing.T) {
		t.Parallel()

		in := &corev1alpha1.Instance{ObjectMeta: metav1.ObjectMeta{Name: "my-db", Namespace: "default"}}
		syncCtx := controller.NewContext(t.Context(), nil, in.DeepCopy(), "test-provider")
		require.False(t, syncCtx.RequestMaintenance(
			"upgrade-to-0.3", "brief rolling restart", controller.MaintenanceRollingRestart))

		flushPendingMaintenance(syncCtx, in)

		require.Len(t, in.Status.PendingMaintenance, 1)
		assert.Equal(t, "upgrade-to-0.3", in.Status.PendingMaintenance[0].ApprovalToken)
		cond := findMaintenanceCondition(in)
		require.NotNil(t, cond)
		assert.Equal(t, metav1.ConditionTrue, cond.Status)
		assert.Equal(t, corev1alpha1.ReasonAwaitingApproval, cond.Reason)
	})

	t.Run("no requests leaves untouched instance without the condition", func(t *testing.T) {
		t.Parallel()

		in := &corev1alpha1.Instance{ObjectMeta: metav1.ObjectMeta{Name: "my-db", Namespace: "default"}}
		syncCtx := controller.NewContext(t.Context(), nil, in.DeepCopy(), "test-provider")

		flushPendingMaintenance(syncCtx, in)

		assert.Empty(t, in.Status.PendingMaintenance)
		assert.Nil(t, findMaintenanceCondition(in))
	})

	t.Run("cleared hold empties the list and flips the condition to False", func(t *testing.T) {
		t.Parallel()

		in := &corev1alpha1.Instance{ObjectMeta: metav1.ObjectMeta{Name: "my-db", Namespace: "default"}}
		in.Status.PendingMaintenance = []corev1alpha1.PendingMaintenanceAction{
			{Description: "brief rolling restart", Severity: corev1alpha1.MaintenanceRollingRestart, ApprovalToken: "upgrade-to-0.3"},
		}
		in.Status.Conditions = []metav1.Condition{{
			Type:   corev1alpha1.ConditionMaintenancePending,
			Status: metav1.ConditionTrue,
			Reason: corev1alpha1.ReasonAwaitingApproval,
		}}
		// This pass the provider no longer requests anything (e.g. the action
		// was approved and applied, or is no longer needed).
		syncCtx := controller.NewContext(t.Context(), nil, in.DeepCopy(), "test-provider")

		flushPendingMaintenance(syncCtx, in)

		assert.Empty(t, in.Status.PendingMaintenance)
		cond := findMaintenanceCondition(in)
		require.NotNil(t, cond)
		assert.Equal(t, metav1.ConditionFalse, cond.Status)
		assert.Equal(t, corev1alpha1.ReasonNoActionsPending, cond.Reason)
	})
}
