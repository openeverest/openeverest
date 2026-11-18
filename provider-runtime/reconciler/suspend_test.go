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
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"

	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
)

func TestSuspendAwarePhase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		suspend     bool
		previous    v1alpha1.InstancePhase
		reported    v1alpha1.InstancePhase
		message     string
		wantPhase   v1alpha1.InstancePhase
		wantMessage string
	}{
		{
			name:      "running instance is untouched",
			previous:  v1alpha1.InstancePhaseReady,
			reported:  v1alpha1.InstancePhaseUpdating,
			message:   "scaling",
			wantPhase: v1alpha1.InstancePhaseUpdating, wantMessage: "scaling",
		},
		{
			name:      "suspend requested, operator not stopped yet",
			suspend:   true,
			previous:  v1alpha1.InstancePhaseReady,
			reported:  v1alpha1.InstancePhaseReady,
			wantPhase: v1alpha1.InstancePhaseSuspending, wantMessage: suspendingMessage,
		},
		{
			name:      "provider's own suspending message is kept",
			suspend:   true,
			reported:  v1alpha1.InstancePhaseSuspending,
			message:   "2 of 3 pods stopped",
			wantPhase: v1alpha1.InstancePhaseSuspending, wantMessage: "2 of 3 pods stopped",
		},
		{
			name:      "suspended",
			suspend:   true,
			previous:  v1alpha1.InstancePhaseSuspending,
			reported:  v1alpha1.InstancePhaseSuspended,
			wantPhase: v1alpha1.InstancePhaseSuspended,
		},
		{
			name:      "failure while suspending is reported",
			suspend:   true,
			reported:  v1alpha1.InstancePhaseFailed,
			message:   "operator error",
			wantPhase: v1alpha1.InstancePhaseFailed, wantMessage: "operator error",
		},
		{
			name:      "resume requested, operator still stopped",
			previous:  v1alpha1.InstancePhaseSuspended,
			reported:  v1alpha1.InstancePhaseSuspended,
			wantPhase: v1alpha1.InstancePhaseResuming, wantMessage: resumingMessage,
		},
		{
			name:      "starting after a suspension",
			previous:  v1alpha1.InstancePhaseResuming,
			reported:  v1alpha1.InstancePhaseInitializing,
			message:   "1 of 3 pods ready",
			wantPhase: v1alpha1.InstancePhaseResuming, wantMessage: "1 of 3 pods ready",
		},
		{
			name:      "resumed",
			previous:  v1alpha1.InstancePhaseResuming,
			reported:  v1alpha1.InstancePhaseReady,
			wantPhase: v1alpha1.InstancePhaseReady,
		},
		{
			name:      "start-up of a new instance is not resuming",
			previous:  v1alpha1.InstancePhasePending,
			reported:  v1alpha1.InstancePhaseInitializing,
			wantPhase: v1alpha1.InstancePhaseInitializing,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			in := &v1alpha1.Instance{Status: v1alpha1.InstanceStatus{Phase: tc.previous}}
			if tc.suspend {
				in.Spec.Suspend = &v1alpha1.SuspendSpec{Suspended: true}
			}

			phase, message := suspendAwarePhase(in, tc.reported, tc.message)

			assert.Equal(t, tc.wantPhase, phase)
			assert.Equal(t, tc.wantMessage, message)
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

	assert.Equal(t, v1alpha1.InstancePhaseSuspended, got.Status.Phase)
	assert.Equal(t, []string{"upgrade-to-0.3"}, r.breaker.blockedTokens(nn, "upgrade-to-0.3"),
		"a suspended pass must not re-arm the retries of an exhausted action",
	)
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
