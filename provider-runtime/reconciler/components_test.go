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
	corev1 "k8s.io/api/core/v1"
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

func TestSetComponentStatuses(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	pods := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		componentPod("db-rs0-0", "db", "engine", true),
		componentPod("db-rs0-1", "db", "engine", false),
		componentPod("db-rs0-2", "db", "engine", true, func(p *corev1.Pod) {
			now := metav1.Now()
			p.DeletionTimestamp = &now
			p.Finalizers = []string{"test"}
		}),
		componentPod("db-backup", "db", "engine", false, func(p *corev1.Pod) { p.Status.Phase = corev1.PodSucceeded }),
		componentPod("db-mongos-0", "db", "proxy", true),
		componentPod("other-rs0-0", "other", "engine", true),
	).Build()
	r := &ProviderReconciler{}
	r.pods.Store(client.Reader(pods))
	in := &corev1alpha1.Instance{ObjectMeta: metav1.ObjectMeta{Name: "db", Namespace: "ns"}}

	r.setComponentStatuses(t.Context(), in, []string{"configServer", "engine", "proxy"})

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

	r.setComponentStatuses(t.Context(), in, nil)

	assert.Empty(t, in.Status.Components)
}

func TestSetComponentStatuses_BeforePodWatch(t *testing.T) {
	t.Parallel()

	r := &ProviderReconciler{}
	in := &corev1alpha1.Instance{Status: corev1alpha1.InstanceStatus{
		Components: []corev1alpha1.ComponentStatus{{Name: "engine"}},
	}}

	r.setComponentStatuses(t.Context(), in, []string{"engine"})

	assert.Equal(t, []corev1alpha1.ComponentStatus{{Name: "engine"}}, in.Status.Components)
}

func TestPodCountChanged(t *testing.T) {
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
		{name: "unrelated change", updated: componentPod("p", "db", "engine", true, func(p *corev1.Pod) {
			p.Annotations = map[string]string{"a": "b"}
		}), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := event.TypedUpdateEvent[*corev1.Pod]{ObjectOld: ready, ObjectNew: tt.updated}

			assert.Equal(t, tt.want, podCountChanged(e))
		})
	}
}
