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
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"

	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
)

func TestSuspendAwareStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		suspend    bool
		wasStopped bool
		reported   controller.Status
		wantReason string
		wantMsg    string
		converged  bool
	}{
		{
			name:       "running instance is untouched",
			reported:   controller.Progressing("scaling"),
			wantReason: "",
			wantMsg:    "scaling",
		},
		{
			name:       "suspend requested, operator not stopped yet",
			suspend:    true,
			reported:   controller.Ready(),
			wantReason: v1alpha1.ReasonSuspending,
			wantMsg:    suspendingMessage,
		},
		{
			name:       "provider's own suspending message is kept",
			suspend:    true,
			reported:   controller.Progressing("2 of 3 pods stopped"),
			wantReason: v1alpha1.ReasonSuspending,
			wantMsg:    "2 of 3 pods stopped",
		},
		{
			name:       "suspended",
			suspend:    true,
			reported:   controller.Suspended(),
			wantReason: v1alpha1.ReasonSuspended,
			wantMsg:    "the engine is stopped",
			converged:  true,
		},
		{
			name:       "failure while suspending is reported",
			suspend:    true,
			reported:   controller.Failed("operator error"),
			wantReason: v1alpha1.ReasonSuspending,
			wantMsg:    "operator error",
		},
		{
			name:       "resume requested, operator still stopped",
			wasStopped: true,
			reported:   controller.Suspended(),
			wantReason: v1alpha1.ReasonResuming,
			wantMsg:    resumingMessage,
		},
		{
			name:       "starting after a suspension",
			wasStopped: true,
			reported:   controller.Progressing("1 of 3 pods ready"),
			wantReason: v1alpha1.ReasonResuming,
			wantMsg:    "1 of 3 pods ready",
		},
		{
			name:       "resumed",
			wasStopped: true,
			reported:   controller.Ready(),
			converged:  true,
		},
		{
			name:     "start-up of a new instance is not resuming",
			reported: controller.Progressing("initializing"),
			wantMsg:  "initializing",
		},
		{
			name:     "stopped outside OpenEverest is a problem",
			reported: controller.Suspended(),
			wantMsg:  "the engine is stopped",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			in := &v1alpha1.Instance{}
			if tc.suspend {
				in.Spec.Suspend = &v1alpha1.SuspendSpec{Suspended: true}
			}
			if tc.wasStopped {
				in.Status.Conditions = []metav1.Condition{{
					Type:   v1alpha1.ConditionReady,
					Status: metav1.ConditionTrue,
					Reason: v1alpha1.ReasonSuspended,
				}}
			}

			status := suspendAwareStatus(in, tc.reported)

			assert.Equal(t, tc.wantReason, status.Reason)
			assert.Equal(t, tc.wantMsg, status.Message)
			assert.Equal(t, tc.converged, status.Converged)
			if tc.name == "stopped outside OpenEverest is a problem" {
				require.Len(t, status.Problems, 1)
				assert.Equal(t, "StoppedOutsideOpenEverest", status.Problems[0].Reason)
			}
		})
	}
}

// maintenanceRequestingProvider asks for one approved disruptive action on
// every Sync.
type maintenanceRequestingProvider struct{ readyTestProvider }

func (p *maintenanceRequestingProvider) Sync(c *controller.Context) error {
	c.RequestMaintenance("upgrade-to-0.3", "converge", controller.MaintenanceRollingRestart)
	return nil
}

func TestReconcile_SuspendKeepsExhaustedRetries(t *testing.T) {
	t.Parallel()
	p := &maintenanceRequestingProvider{readyTestProvider{status: controller.Suspended()}}
	provider := &v1alpha1.Provider{
		ObjectMeta: metav1.ObjectMeta{Name: "test-provider"},
		Spec:       v1alpha1.ProviderSpec{Suspend: &v1alpha1.ProviderSuspend{}},
	}
	in := readyTestInstance(func(in *v1alpha1.Instance) {
		in.Spec.Maintenance = &v1alpha1.MaintenanceSpec{Approved: "upgrade-to-0.3"}
		in.Spec.Suspend = &v1alpha1.SuspendSpec{Suspended: true}
	})
	r := newReadyTestReconciler(p, provider, in)
	nn := client.ObjectKeyFromObject(in)
	for range maintenanceBreakerThreshold {
		r.breaker.recordFailure(nn, "upgrade-to-0.3", []string{"upgrade-to-0.3"})
	}

	got, err := reconcileAndGet(t, r)
	require.NoError(t, err)

	ready := meta.FindStatusCondition(got.Status.Conditions, v1alpha1.ConditionReady)
	require.NotNil(t, ready)
	assert.Equal(t, metav1.ConditionTrue, ready.Status)
	assert.Equal(t, v1alpha1.ReasonSuspended, ready.Reason)
	assert.Equal(t, []string{"upgrade-to-0.3"}, r.breaker.blockedTokens(nn, "upgrade-to-0.3"),
		"a suspended pass must not re-arm the retries of an exhausted action",
	)
}

func TestReconcile_SuspendTransitionsUseStatusReasons(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status controller.Status
		mutate func(*v1alpha1.Instance)
		reason string
	}{
		{
			name:   "suspending",
			status: controller.Ready(),
			mutate: func(in *v1alpha1.Instance) {
				in.Spec.Suspend = &v1alpha1.SuspendSpec{Suspended: true}
			},
			reason: v1alpha1.ReasonSuspending,
		},
		{
			name:   "resuming",
			status: controller.Progressing("1 of 3 pods ready"),
			mutate: func(in *v1alpha1.Instance) {
				in.Status.ObservedGeneration = in.Generation
				in.Status.Phase = v1alpha1.InstancePhaseProvisioned
				in.Status.Conditions = []metav1.Condition{{
					Type:               v1alpha1.ConditionReady,
					Status:             metav1.ConditionTrue,
					Reason:             v1alpha1.ReasonSuspended,
					ObservedGeneration: in.Generation,
				}}
			},
			reason: v1alpha1.ReasonResuming,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := &readyTestProvider{status: tc.status}
			provider := readyTestProviderObject()
			provider.Spec.Suspend = &v1alpha1.ProviderSuspend{}
			r := newReadyTestReconciler(p, provider, readyTestInstance(tc.mutate))

			got, err := reconcileAndGet(t, r)
			require.NoError(t, err)
			requireReady(t, got, metav1.ConditionFalse, tc.reason)
		})
	}
}

func TestSuspendChanged(t *testing.T) {
	t.Parallel()

	running := readyTestInstance(nil)
	suspended := readyTestInstance(func(in *v1alpha1.Instance) {
		in.Spec.Suspend = &v1alpha1.SuspendSpec{Suspended: true}
	})
	relabelled := suspended.DeepCopy()
	relabelled.Labels["team"] = "a"
	p := suspendChanged()

	assert.True(t, p.Update(event.UpdateEvent{ObjectOld: running, ObjectNew: suspended}), "suspend")
	assert.True(t, p.Update(event.UpdateEvent{ObjectOld: suspended, ObjectNew: running}), "resume")
	assert.False(t, p.Update(event.UpdateEvent{ObjectOld: suspended, ObjectNew: relabelled}), "other changes")
}
