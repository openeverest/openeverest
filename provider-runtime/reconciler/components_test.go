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
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
)

func componentPod(name, instance, component string, ready bool, mutate ...func(*corev1.Pod)) *corev1.Pod {
	readyStatus := corev1.ConditionFalse
	if ready {
		readyStatus = corev1.ConditionTrue
	}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "ns",
			Labels: map[string]string{
				controller.InstanceLabel:  instance,
				controller.ComponentLabel: component,
			},
		},
		Status: corev1.PodStatus{
			Phase:      corev1.PodRunning,
			Conditions: []corev1.PodCondition{{Type: corev1.PodReady, Status: readyStatus}},
		},
	}
	for _, m := range mutate {
		m(pod)
	}

	return pod
}

func TestSetPodStatus(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	pods := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		componentPod("db-rs0-0", "db", "engine", true),
		componentPod("db-rs0-1", "db", "engine", false, func(p *corev1.Pod) {
			p.Status.Phase = corev1.PodPending
			p.Status.Conditions = append(p.Status.Conditions, corev1.PodCondition{
				Type:               corev1.PodScheduled,
				Status:             corev1.ConditionFalse,
				Reason:             corev1.PodReasonUnschedulable,
				Message:            "0/3 nodes are available: 3 Insufficient memory.",
				LastTransitionTime: metav1.NewTime(now.Add(-40 * time.Second)),
			})
		}),
		componentPod("db-rs0-2", "db", "engine", true, func(p *corev1.Pod) {
			deleted := metav1.Now()
			p.DeletionTimestamp = &deleted
			p.Finalizers = []string{"test"}
		}),
		componentPod("db-backup", "db", "engine", false, func(p *corev1.Pod) { p.Status.Phase = corev1.PodSucceeded }),
		componentPod("db-mongos-0", "db", "proxy", true),
		componentPod("other-rs0-0", "other", "engine", true),
	).Build()
	r := &ProviderReconciler{}
	r.pods.Store(client.Reader(pods))
	in := &corev1alpha1.Instance{ObjectMeta: metav1.ObjectMeta{Name: "db", Namespace: "ns"}}

	recheck := r.setPodStatus(t.Context(), in, []string{"configServer", "engine", "proxy"}, now)

	want := []corev1alpha1.ComponentStatus{
		{
			Name:          "configServer",
			Selector:      "core.openeverest.io/component=configServer,core.openeverest.io/instance=db",
			Replicas:      new(int32(0)),
			ReadyReplicas: new(int32(0)),
		},
		{
			Name:          "engine",
			Selector:      "core.openeverest.io/component=engine,core.openeverest.io/instance=db",
			Replicas:      new(int32(2)),
			ReadyReplicas: new(int32(1)),
		},
		{
			Name:          "proxy",
			Selector:      "core.openeverest.io/component=proxy,core.openeverest.io/instance=db",
			Replicas:      new(int32(1)),
			ReadyReplicas: new(int32(1)),
		},
	}
	assert.Equal(t, want, in.Status.Components)
	assert.Equal(t, 20*time.Second, recheck)
	scheduled := meta.FindStatusCondition(in.Status.Conditions, corev1alpha1.ConditionPodsScheduled)
	require.NotNil(t, scheduled)
	assert.Equal(t, metav1.ConditionTrue, scheduled.Status)
	readiness := meta.FindStatusCondition(in.Status.Conditions, corev1alpha1.ConditionPodsReady)
	require.NotNil(t, readiness)
	assert.Equal(t, metav1.ConditionFalse, readiness.Status)
	assert.Equal(t, "engine: 1 of 2 pods are not ready", readiness.Message)

	recheck = r.setPodStatus(t.Context(), in, nil, now)

	assert.Empty(t, in.Status.Components)
	assert.Empty(t, in.Status.Conditions)
	assert.Zero(t, recheck)
}

func TestSetPodStatus_BeforePodWatch(t *testing.T) {
	t.Parallel()

	r := &ProviderReconciler{}
	in := &corev1alpha1.Instance{Status: corev1alpha1.InstanceStatus{
		Components: []corev1alpha1.ComponentStatus{{Name: "engine"}},
	}}

	recheck := r.setPodStatus(t.Context(), in, []string{"engine"}, time.Now())

	assert.Equal(t, []corev1alpha1.ComponentStatus{{Name: "engine"}}, in.Status.Components)
	assert.Empty(t, in.Status.Conditions)
	assert.Zero(t, recheck)
}

func TestPodChanged(t *testing.T) {
	t.Parallel()

	ready := componentPod("p", "db", "engine", true)
	tests := []struct {
		name    string
		updated *corev1.Pod
		want    bool
	}{
		{name: "becomes unready", updated: componentPod("p", "db", "engine", false), want: true},
		{name: "starts terminating", updated: componentPod("p", "db", "engine", true, func(p *corev1.Pod) {
			now := metav1.Now()
			p.DeletionTimestamp = &now
		}), want: true},
		{name: "starts crash looping", updated: componentPod("p", "db", "engine", false, func(p *corev1.Pod) {
			p.Status.ContainerStatuses = []corev1.ContainerStatus{{
				Name:  "mongod",
				State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff"}},
			}}
		}), want: true},
		{name: "unrelated change", updated: componentPod("p", "db", "engine", true, func(p *corev1.Pod) {
			p.Annotations = map[string]string{"a": "b"}
		}), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := event.TypedUpdateEvent[*corev1.Pod]{ObjectOld: ready, ObjectNew: tt.updated}

			assert.Equal(t, tt.want, podChanged(e))
		})
	}
}
