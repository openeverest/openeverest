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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"

	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
	"github.com/openeverest/openeverest/v2/contract/monitoring"
)

const testNamespace = "team-a"

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
		name               string
		status             metav1.ConditionStatus
		observedGeneration int64
		want               bool
	}{
		{name: "claimed", status: metav1.ConditionTrue, observedGeneration: 3, want: true},
		{name: "claimed at an older generation", status: metav1.ConditionTrue, observedGeneration: 2, want: false},
		{name: "claimed without an observed generation", status: metav1.ConditionTrue, want: false},
		{name: "rejected", status: metav1.ConditionFalse, observedGeneration: 3, want: false},
		{name: "waiting for controller", status: metav1.ConditionUnknown, want: false},
		{name: "no condition", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			class := &monitoringv1alpha1.MonitoringClass{ObjectMeta: metav1.ObjectMeta{Generation: 3}}
			if tt.status != "" {
				class.Status.Conditions = []metav1.Condition{{Type: "Accepted", Status: tt.status, ObservedGeneration: tt.observedGeneration}}
			}
			assert.Equal(t, tt.want, monitoring.ClassIsAccepted(class))
		})
	}
}
