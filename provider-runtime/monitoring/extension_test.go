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

package monitoring_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"

	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/monitoring"
)

const (
	testFieldOwner = "openeverest.io/monitoring-test"
	testClassName  = "prometheus"
)

func newClass(name string) *monitoringv1alpha1.MonitoringClass {
	return &monitoringv1alpha1.MonitoringClass{
		ObjectMeta: metav1.ObjectMeta{Name: name, Generation: 1},
		Spec: monitoringv1alpha1.MonitoringClassSpec{
			ExtensionManaged: &monitoringv1alpha1.ExtensionManagedFulfilment{},
		},
	}
}

func TestClaimKeepsCoreCompatibilityMatrix(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	c := newFakeClient(t, newClass(testClassName), newClass("pmm"))

	ours := &monitoringv1alpha1.MonitoringClass{}
	require.NoError(t, c.Get(ctx, client.ObjectKey{Name: testClassName}, ours))
	require.NoError(t, monitoring.ApplyStatus(ctx, c, ours, monitoring.CompatibilityFieldOwner, map[string]any{
		"providers": []any{map[string]any{"name": "percona-server-mongodb", "mode": "ExtensionManaged"}},
	}))

	require.NoError(t, monitoring.ClassController{Client: c, FieldOwner: testFieldOwner, ClassName: testClassName}.Claim(ctx))

	require.NoError(t, c.Get(ctx, client.ObjectKey{Name: testClassName}, ours))
	accepted := meta.FindStatusCondition(ours.Status.Conditions, monitoringv1alpha1.MonitoringClassConditionAccepted)
	require.NotNil(t, accepted)
	assert.Equal(t, metav1.ConditionTrue, accepted.Status)
	assert.Equal(t, monitoringv1alpha1.ReasonAccepted, accepted.Reason)
	wantProviders := []monitoringv1alpha1.ProviderCompatibility{
		{Name: "percona-server-mongodb", Mode: corev1alpha1.MonitoringExecutionModeExtensionManaged},
	}
	assert.Equal(t, wantProviders, ours.Status.Providers)
	var appliers []string
	for _, mf := range ours.ManagedFields {
		if mf.Operation == metav1.ManagedFieldsOperationApply {
			appliers = append(appliers, mf.Manager)
		}
	}
	assert.ElementsMatch(t, []string{monitoring.CompatibilityFieldOwner, testFieldOwner}, appliers)

	other := &monitoringv1alpha1.MonitoringClass{}
	require.NoError(t, c.Get(ctx, client.ObjectKey{Name: "pmm"}, other))
	assert.Empty(t, other.Status.Conditions)
}

func TestClaimWithoutServedClass(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	c := newFakeClient(t, newClass("pmm"))

	require.NoError(t, monitoring.ClassController{Client: c, FieldOwner: testFieldOwner, ClassName: testClassName}.Claim(ctx))

	other := &monitoringv1alpha1.MonitoringClass{}
	require.NoError(t, c.Get(ctx, client.ObjectKey{Name: "pmm"}, other))
	assert.Empty(t, other.Status.Conditions)
}

func TestClaimClass(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		class        string
		wantAccepted bool
	}{
		{name: "served class", class: testClassName, wantAccepted: true},
		{name: "another class", class: "pmm", wantAccepted: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			c := newFakeClient(t, newClass(tt.class))
			class := &monitoringv1alpha1.MonitoringClass{}
			require.NoError(t, c.Get(ctx, client.ObjectKey{Name: tt.class}, class))

			require.NoError(t, monitoring.ClassController{Client: c, FieldOwner: testFieldOwner, ClassName: testClassName}.ClaimClass(ctx, class))

			require.NoError(t, c.Get(ctx, client.ObjectKey{Name: tt.class}, class))
			accepted := meta.FindStatusCondition(class.Status.Conditions, monitoringv1alpha1.MonitoringClassConditionAccepted)
			if !tt.wantAccepted {
				assert.Nil(t, accepted)
				return
			}
			require.NotNil(t, accepted)
			assert.Equal(t, metav1.ConditionTrue, accepted.Status)
			assert.Equal(t, monitoringv1alpha1.ReasonAccepted, accepted.Reason)
			assert.Equal(t, int64(1), accepted.ObservedGeneration)
		})
	}
}

func TestOwnsClass(t *testing.T) {
	t.Parallel()

	tests := []struct {
		class string
		want  bool
	}{
		{class: testClassName, want: true},
		{class: "pmm", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.class, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, monitoring.ClassController{ClassName: testClassName}.OwnsClass(newClass(tt.class)))
		})
	}
}

func TestSetDestinationReadyKeepsServerVersionAndAccepted(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	c := newFakeClient(t, &monitoringv1alpha1.MonitoringDestination{
		ObjectMeta: metav1.ObjectMeta{Name: "pmm-prod", Namespace: testNamespace, Generation: 1},
		Spec:       monitoringv1alpha1.MonitoringDestinationSpec{ClassRef: common.ObjectRef{Name: "pmm"}},
	})
	key := client.ObjectKey{Namespace: testNamespace, Name: "pmm-prod"}
	ext := monitoring.ClassController{Client: c, FieldOwner: testFieldOwner, ClassName: "pmm"}

	dst := &monitoringv1alpha1.MonitoringDestination{}
	require.NoError(t, c.Get(ctx, key, dst))
	accepted := monitoring.NewCondition(dst, monitoringv1alpha1.MonitoringDestinationConditionAccepted, metav1.ConditionTrue, monitoringv1alpha1.ReasonAccepted, "")
	require.NoError(t, monitoring.ApplyCondition(ctx, c, dst, dst.Status.Conditions, monitoring.MaterializerFieldOwner, accepted))

	require.NoError(t, c.Get(ctx, key, dst))
	require.NoError(t, ext.SetDestinationReady(ctx, dst, monitoringv1alpha1.ReasonReady, "", "3.8.0"))

	require.NoError(t, c.Get(ctx, key, dst))
	require.NoError(t, ext.SetDestinationReady(ctx, dst, monitoringv1alpha1.ReasonUnreachable, "connection refused", ""))

	require.NoError(t, c.Get(ctx, key, dst))
	assert.Equal(t, "3.8.0", dst.Status.ServerVersion)
	ready := meta.FindStatusCondition(dst.Status.Conditions, monitoringv1alpha1.MonitoringDestinationConditionReady)
	require.NotNil(t, ready)
	assert.Equal(t, metav1.ConditionFalse, ready.Status)
	assert.Equal(t, "Unreachable", ready.Reason)
	assert.True(t, meta.IsStatusConditionTrue(dst.Status.Conditions, monitoringv1alpha1.MonitoringDestinationConditionAccepted))
}

func TestOwnedBy(t *testing.T) {
	t.Parallel()
	b := newBinding()
	b.UID = "5f0c6a52-0d6e-4c1b-9a8e-2b8f3d1e7c44"

	want := metav1.OwnerReference{
		APIVersion: "monitoring.openeverest.io/v1alpha1",
		Kind:       "MonitoringBinding",
		Name:       "orders-pmm-1a2b3c4d",
		UID:        "5f0c6a52-0d6e-4c1b-9a8e-2b8f3d1e7c44",
		Controller: new(true),
	}
	assert.Equal(t, want, monitoring.OwnedBy(b))
}

func TestBindingPredicate(t *testing.T) {
	t.Parallel()

	binding := func(mode corev1alpha1.MonitoringExecutionMode, class string) *monitoringv1alpha1.MonitoringBinding {
		b := newBinding()
		b.Spec.ExecutionMode = mode
		if class != "" {
			b.Labels = map[string]string{monitoringv1alpha1.BindingClassLabel: class}
		}
		return b
	}
	tests := []struct {
		name string
		obj  client.Object
		want bool
	}{
		{name: "own ExtensionManaged binding", obj: binding(corev1alpha1.MonitoringExecutionModeExtensionManaged, "prometheus"), want: true},
		{name: "own class but ProviderManaged", obj: binding(corev1alpha1.MonitoringExecutionModeProviderManaged, "prometheus"), want: false},
		{name: "mode not resolved yet", obj: binding("", "prometheus"), want: false},
		{name: "other class", obj: binding(corev1alpha1.MonitoringExecutionModeExtensionManaged, "datadog"), want: false},
		{name: "class not resolved yet", obj: binding(corev1alpha1.MonitoringExecutionModeExtensionManaged, ""), want: false},
		{name: "not a binding", obj: &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{monitoringv1alpha1.BindingClassLabel: "prometheus"}}}, want: false},
	}
	p := monitoring.BindingPredicate("prometheus", "victoriametrics")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, p.Create(event.CreateEvent{Object: tt.obj}))
		})
	}
}
