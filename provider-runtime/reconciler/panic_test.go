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
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
)

// panickingProvider is readyTestProvider whose call named in panics. Sync
// runs readyTestProvider's sync hook first.
type panickingProvider struct {
	readyTestProvider

	in    string
	value any
}

func (p *panickingProvider) Sync(c *controller.Context) error {
	if p.in == "Sync" {
		if p.sync != nil {
			p.sync(c)
		}
		panic(p.value)
	}
	return p.readyTestProvider.Sync(c)
}

func (p *panickingProvider) Status(c *controller.Context) (controller.Status, error) {
	if p.in == "Status" {
		panic(p.value)
	}
	return p.readyTestProvider.Status(c)
}

func (p *panickingProvider) Cleanup(c *controller.Context) error {
	if p.in == "Cleanup" {
		panic(p.value)
	}
	return p.readyTestProvider.Cleanup(c)
}

// A provider that panics is a provider whose calls fail: the status is
// written as for an error, and Reconcile returns one.
func TestReconcile_ProviderPanics(t *testing.T) {
	t.Parallel()

	errBoom := errors.New("boom")
	tests := []struct {
		name  string
		call  string
		value any
	}{
		{name: "Sync", call: "Sync", value: "boom"},
		{name: "Sync panics with an error", call: "Sync", value: errBoom},
		{name: "Status", call: "Status", value: "boom"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := &panickingProvider{readyTestProvider: readyTestProvider{status: controller.Ready()}, in: tc.call, value: tc.value}
			st := newStatusTest(t, p, readyTestProviderObject(), readyTestInstance(nil))

			got, _, err := st.reconcile()
			require.Error(t, err)
			assert.Contains(t, err.Error(), "panic: boom")
			if cause, ok := tc.value.(error); ok {
				require.ErrorIs(t, err, cause)
			}
			assert.Equal(t, 1, st.writes, "the status is written")
			assert.Equal(t, got.Generation, got.Status.ObservedGeneration)
			requireCondition(t, got, v1alpha1.ConditionSynced, metav1.ConditionUnknown, v1alpha1.ReasonReconcileError)
			assert.Equal(t, syncGrace, st.lastRequeue())

			st.advance(syncGrace)
			got, _, err = st.reconcile()
			require.Error(t, err)
			synced := requireCondition(t, got, v1alpha1.ConditionSynced, metav1.ConditionFalse, v1alpha1.ReasonReconcileError)
			assert.Contains(t, synced.Message, "panic: boom")
			requireReady(t, got, metav1.ConditionFalse, v1alpha1.ReasonReconcileError)
		})
	}

	t.Run("Sync panics like it fails: the approved action is held after as many tries", func(t *testing.T) {
		t.Parallel()
		p := &panickingProvider{readyTestProvider: readyTestProvider{status: controller.Ready()}, in: "Sync", value: "boom"}
		in := readyTestInstance(func(in *v1alpha1.Instance) {
			in.Spec.Maintenance = &v1alpha1.MaintenanceSpec{Approved: "restart-1"}
		})
		p.sync = func(c *controller.Context) {
			c.RequestMaintenance("restart-1", "rolling restart", controller.MaintenanceRollingRestart)
		}
		st := newStatusTest(t, p, readyTestProviderObject(), in)
		for range maintenanceBreakerThreshold {
			_, _, err := st.reconcile()
			require.Error(t, err)
		}
		assert.Equal(t, []string{"restart-1"}, st.r.breaker.blockedTokens(st.key(), "restart-1"))
	})

	t.Run("Cleanup", func(t *testing.T) {
		t.Parallel()
		p := &panickingProvider{in: "Cleanup", value: "boom"}
		in := readyTestInstance(func(in *v1alpha1.Instance) {
			deleted := metav1.NewTime(time.Date(2026, 3, 1, 11, 0, 0, 0, time.UTC))
			in.DeletionTimestamp = &deleted
			in.Status.Phase = v1alpha1.InstancePhaseProvisioned
		})
		st := newStatusTest(t, p, readyTestProviderObject(), in)

		got, _, err := st.reconcile()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "panic: boom")
		assert.Equal(t, v1alpha1.InstancePhaseTerminating, got.Status.Phase)
		assert.Equal(t, []string{finalizerName}, got.Finalizers, "cleanup is not done")

		st.advance(syncGrace)
		got, _, err = st.reconcile()
		require.Error(t, err)
		synced := requireCondition(t, got, v1alpha1.ConditionSynced, metav1.ConditionFalse, v1alpha1.ReasonReconcileError)
		assert.Contains(t, synced.Message, "panic: boom")
	})
}
