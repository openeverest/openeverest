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
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/managedfields"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/kube-openapi/pkg/validation/spec"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"

	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/monitoring"
)

const testNamespace = "team-a"

func newScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	require.NoError(t, corev1alpha1.AddToScheme(s))
	require.NoError(t, monitoringv1alpha1.AddToScheme(s))
	return s
}

func newFakeClient(t *testing.T, objs ...client.Object) client.Client { //nolint:ireturn // the fake is used through the client interface
	t.Helper()
	return fake.NewClientBuilder().
		WithScheme(newScheme(t)).
		WithTypeConverters(crdTypeConverter(t), managedfields.NewDeducedTypeConverter()).
		WithObjects(objs...).
		WithStatusSubresource(
			&monitoringv1alpha1.MonitoringClass{},
			&monitoringv1alpha1.MonitoringDestination{},
			&monitoringv1alpha1.MonitoringBinding{},
		).
		Build()
}

// crdTypeConverter makes the fake client's server-side apply follow the
// generated CRD schemas (listType=map conditions) instead of treating every
// list as atomic.
func crdTypeConverter(t *testing.T) managedfields.TypeConverter { //nolint:ireturn // the fake client takes the interface
	t.Helper()
	crds := os.DirFS(filepath.Join("..", "..", "config", "crd", "bases"))
	models := map[string]*spec.Schema{}
	for _, plural := range []string{"monitoringclasses", "monitoringdestinations", "monitoringbindings"} {
		raw, err := fs.ReadFile(crds, "monitoring.openeverest.io_"+plural+".yaml")
		require.NoError(t, err)
		crd := &apiextensionsv1.CustomResourceDefinition{}
		require.NoError(t, yaml.Unmarshal(raw, crd))
		for _, v := range crd.Spec.Versions {
			js, err := json.Marshal(v.Schema.OpenAPIV3Schema)
			require.NoError(t, err)
			s := &spec.Schema{}
			require.NoError(t, json.Unmarshal(js, s))
			s.AddExtension("x-kubernetes-group-version-kind", []any{
				map[string]any{"group": crd.Spec.Group, "version": v.Name, "kind": crd.Spec.Names.Kind},
			})
			models[crd.Spec.Group+"."+v.Name+"."+crd.Spec.Names.Kind] = s
		}
	}
	tc, err := managedfields.NewTypeConverter(models, false)
	require.NoError(t, err)
	return tc
}

func newBinding() *monitoringv1alpha1.MonitoringBinding {
	return &monitoringv1alpha1.MonitoringBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "orders-pmm-1a2b3c4d", Namespace: testNamespace, Generation: 2},
		Spec: monitoringv1alpha1.MonitoringBindingSpec{
			InstanceRef:    common.ObjectRef{Name: "orders"},
			DestinationRef: common.MonitoringDestinationRef{Kind: "MonitoringDestination", Name: "pmm-prod"},
		},
	}
}

func TestBindingName(t *testing.T) {
	t.Parallel()

	t.Run("short names keep instance and entry", func(t *testing.T) {
		t.Parallel()
		name := monitoring.BindingName("orders", "pmm")
		assert.Equal(t, name, monitoring.BindingName("orders", "pmm"))
		assert.Regexp(t, `^orders-pmm-[0-9a-f]{8}$`, name)
	})

	t.Run("differs per entry", func(t *testing.T) {
		t.Parallel()
		assert.NotEqual(t, monitoring.BindingName("orders", "pmm"), monitoring.BindingName("orders", "prometheus"))
	})

	t.Run("long names fit a label value and stay distinct", func(t *testing.T) {
		t.Parallel()
		instance := strings.Repeat("a", 63)
		first := monitoring.BindingName(instance, strings.Repeat("b", 62)+"1")
		second := monitoring.BindingName(instance, strings.Repeat("b", 62)+"2")
		assert.LessOrEqual(t, len(first), 63)
		assert.Empty(t, validation.IsDNS1123Label(first))
		assert.NotEqual(t, first, second)
	})

	t.Run("truncation never leaves a separator before the hash", func(t *testing.T) {
		t.Parallel()
		name := monitoring.BindingName(strings.Repeat("a", 53)+".b", "pmm")
		assert.LessOrEqual(t, len(name), 63)
		assert.Empty(t, validation.IsDNS1123Subdomain(name))
		assert.Regexp(t, `^a{53}-[0-9a-f]{8}$`, name)
	})
}

func TestBindingLabels(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		class string
		want  map[string]string
	}{
		{
			name:  "resolved class",
			class: "pmm",
			want: map[string]string{
				monitoringv1alpha1.BindingInstanceLabel:    "orders",
				monitoringv1alpha1.BindingEntryLabel:       "metrics",
				monitoringv1alpha1.BindingDestinationLabel: "pmm-prod",
				monitoringv1alpha1.BindingClassLabel:       "pmm",
			},
		},
		{
			name:  "unresolved class",
			class: "",
			want: map[string]string{
				monitoringv1alpha1.BindingInstanceLabel:    "orders",
				monitoringv1alpha1.BindingEntryLabel:       "metrics",
				monitoringv1alpha1.BindingDestinationLabel: "pmm-prod",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, monitoring.BindingLabels("orders", "metrics", "pmm-prod", tt.class))
		})
	}
}

func TestIsAccepted(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		conditions []metav1.Condition
		want       bool
	}{
		{name: "no condition", want: false},
		{
			name:       "accepted for the current generation",
			conditions: []metav1.Condition{{Type: "Accepted", Status: metav1.ConditionTrue, ObservedGeneration: 2}},
			want:       true,
		},
		{
			name:       "accepted for an older generation",
			conditions: []metav1.Condition{{Type: "Accepted", Status: metav1.ConditionTrue, ObservedGeneration: 1}},
			want:       false,
		},
		{
			name:       "not accepted",
			conditions: []metav1.Condition{{Type: "Accepted", Status: metav1.ConditionFalse, ObservedGeneration: 2}},
			want:       false,
		},
		{
			name:       "only configured",
			conditions: []metav1.Condition{{Type: "Configured", Status: metav1.ConditionTrue, ObservedGeneration: 2}},
			want:       false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			b := newBinding()
			b.Status.Conditions = tt.conditions
			assert.Equal(t, tt.want, monitoring.IsAccepted(b))
		})
	}
}

func TestClassIsAccepted(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status metav1.ConditionStatus
		want   bool
	}{
		{name: "claimed", status: metav1.ConditionTrue, want: true},
		{name: "rejected", status: metav1.ConditionFalse, want: false},
		{name: "waiting for controller", status: metav1.ConditionUnknown, want: false},
		{name: "no condition", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			class := &monitoringv1alpha1.MonitoringClass{}
			if tt.status != "" {
				class.Status.Conditions = []metav1.Condition{{Type: "Accepted", Status: tt.status}}
			}
			assert.Equal(t, tt.want, monitoring.ClassIsAccepted(class))
		})
	}
}

func TestApplyConditionKeepsOtherWritersConditions(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	c := newFakeClient(t, newBinding())

	b := &monitoringv1alpha1.MonitoringBinding{}
	require.NoError(t, c.Get(ctx, client.ObjectKeyFromObject(newBinding()), b))
	accepted := monitoring.NewCondition(b, monitoringv1alpha1.MonitoringBindingConditionAccepted, metav1.ConditionTrue, monitoringv1alpha1.ReasonAccepted, "")
	require.NoError(t, monitoring.ApplyCondition(ctx, c, b, b.Status.Conditions, monitoring.MaterializerFieldOwner, accepted))

	require.NoError(t, c.Get(ctx, client.ObjectKeyFromObject(b), b))
	ext := monitoring.ClassController{Client: c, ControllerName: "openeverest.io/monitoring-test"}
	require.NoError(t, ext.SetConfigured(ctx, b, monitoringv1alpha1.ReasonRenderFailed, "bad url"))

	require.NoError(t, c.Get(ctx, client.ObjectKeyFromObject(b), b))
	gotAccepted := meta.FindStatusCondition(b.Status.Conditions, monitoringv1alpha1.MonitoringBindingConditionAccepted)
	require.NotNil(t, gotAccepted)
	assert.Equal(t, metav1.ConditionTrue, gotAccepted.Status)
	assert.Equal(t, int64(2), gotAccepted.ObservedGeneration)
	gotConfigured := meta.FindStatusCondition(b.Status.Conditions, monitoringv1alpha1.MonitoringBindingConditionConfigured)
	require.NotNil(t, gotConfigured)
	assert.Equal(t, metav1.ConditionFalse, gotConfigured.Status)
	assert.Equal(t, "RenderFailed", gotConfigured.Reason)
	assert.Equal(t, "bad url", gotConfigured.Message)
}

func TestApplyConditionLastTransitionTime(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	c := newFakeClient(t, newBinding())
	since := metav1.NewTime(time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC))

	b := &monitoringv1alpha1.MonitoringBinding{}
	require.NoError(t, c.Get(ctx, client.ObjectKeyFromObject(newBinding()), b))
	first := monitoring.NewCondition(b, monitoringv1alpha1.MonitoringBindingConditionAccepted, metav1.ConditionFalse, monitoringv1alpha1.ReasonDestinationNotFound, "")
	first.LastTransitionTime = since
	require.NoError(t, monitoring.ApplyCondition(ctx, c, b, b.Status.Conditions, monitoring.MaterializerFieldOwner, first))

	require.NoError(t, c.Get(ctx, client.ObjectKeyFromObject(b), b))
	sameStatus := monitoring.NewCondition(b, monitoringv1alpha1.MonitoringBindingConditionAccepted, metav1.ConditionFalse, monitoringv1alpha1.ReasonClassNotFound, "")
	require.NoError(t, monitoring.ApplyCondition(ctx, c, b, b.Status.Conditions, monitoring.MaterializerFieldOwner, sameStatus))

	require.NoError(t, c.Get(ctx, client.ObjectKeyFromObject(b), b))
	got := meta.FindStatusCondition(b.Status.Conditions, monitoringv1alpha1.MonitoringBindingConditionAccepted)
	require.NotNil(t, got)
	assert.Equal(t, "ClassNotFound", got.Reason)
	assert.True(t, since.Equal(&got.LastTransitionTime), "unchanged status keeps lastTransitionTime, got %s", got.LastTransitionTime)

	flipped := monitoring.NewCondition(b, monitoringv1alpha1.MonitoringBindingConditionAccepted, metav1.ConditionTrue, monitoringv1alpha1.ReasonAccepted, "")
	require.NoError(t, monitoring.ApplyCondition(ctx, c, b, b.Status.Conditions, monitoring.MaterializerFieldOwner, flipped))

	require.NoError(t, c.Get(ctx, client.ObjectKeyFromObject(b), b))
	got = meta.FindStatusCondition(b.Status.Conditions, monitoringv1alpha1.MonitoringBindingConditionAccepted)
	require.NotNil(t, got)
	assert.Equal(t, metav1.ConditionTrue, got.Status)
	assert.True(t, got.LastTransitionTime.After(since.Time), "a status change moves lastTransitionTime, got %s", got.LastTransitionTime)
}
