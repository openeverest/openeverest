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

package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/monitoring"
)

const (
	monitoringTestNamespace = "team-a"
	monitoringTestUID       = "7d2f4c1e-0000-4000-8000-0000000000aa"
)

func monitoringTestContext(t *testing.T, in *v1alpha1.Instance, objs ...client.Object) *Context {
	t.Helper()
	s := runtime.NewScheme()
	require.NoError(t, v1alpha1.AddToScheme(s))
	require.NoError(t, monitoringv1alpha1.AddToScheme(s))
	c := fake.NewClientBuilder().WithScheme(s).WithObjects(objs...).Build()
	return NewContext(t.Context(), c, in, "test-provider")
}

// monitoredInstance returns the Instance "orders" with one destination entry
// per name, each pointing at the destination "<name>-dest".
func monitoredInstance(entries ...string) *v1alpha1.Instance {
	in := &v1alpha1.Instance{ObjectMeta: metav1.ObjectMeta{Name: "orders", Namespace: monitoringTestNamespace, UID: monitoringTestUID}}
	if len(entries) == 0 {
		return in
	}
	in.Spec.Monitoring = &v1alpha1.InstanceMonitoringSpec{}
	for _, e := range entries {
		in.Spec.Monitoring.Destinations = append(in.Spec.Monitoring.Destinations, v1alpha1.InstanceMonitoringDestination{
			Name:           e,
			DestinationRef: common.MonitoringDestinationRef{Kind: "MonitoringDestination", Name: e + "-dest"},
		})
	}
	return in
}

// monitoringBinding returns the binding core creates for entry of "orders".
// class and mode are left unset when empty; accepted adds an Accepted
// condition for the binding's generation.
func monitoringBinding(entry string, mode v1alpha1.MonitoringExecutionMode, class string, accepted metav1.ConditionStatus) *monitoringv1alpha1.MonitoringBinding {
	b := &monitoringv1alpha1.MonitoringBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:       monitoring.BindingName("orders", entry),
			Namespace:  monitoringTestNamespace,
			Generation: 3,
			Labels:     monitoring.BindingLabels("orders", entry, entry+"-dest", class),
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "core.openeverest.io/v1alpha1", Kind: "Instance", Name: "orders", UID: monitoringTestUID, Controller: new(true),
			}},
		},
		Spec: monitoringv1alpha1.MonitoringBindingSpec{
			InstanceRef:    common.ObjectRef{Name: "orders"},
			DestinationRef: common.MonitoringDestinationRef{Kind: "MonitoringDestination", Name: entry + "-dest"},
			ExecutionMode:  mode,
		},
	}
	if class != "" {
		b.Spec.ClassRef = &common.ObjectRef{Name: class}
	}
	if accepted != "" {
		b.Status.Conditions = []metav1.Condition{{
			Type: monitoringv1alpha1.MonitoringBindingConditionAccepted, Status: accepted,
			Reason: "Test", ObservedGeneration: 3, LastTransitionTime: metav1.Now(),
		}}
	}
	return b
}

func pmmDestination(name string) *monitoringv1alpha1.MonitoringDestination {
	return &monitoringv1alpha1.MonitoringDestination{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: monitoringTestNamespace},
		Spec:       monitoringv1alpha1.MonitoringDestinationSpec{ClassRef: common.ObjectRef{Name: "pmm"}},
	}
}

func pmmClass() *monitoringv1alpha1.MonitoringClass {
	return &monitoringv1alpha1.MonitoringClass{
		ObjectMeta: metav1.ObjectMeta{Name: "pmm"},
		Spec: monitoringv1alpha1.MonitoringClassSpec{
			ControllerName:  "openeverest.io/monitoring-pmm",
			ProviderManaged: &monitoringv1alpha1.ProviderManagedFulfilment{Integration: "pmm"},
		},
	}
}

func extensionManagedClass(name string, metrics bool) *monitoringv1alpha1.MonitoringClass {
	return &monitoringv1alpha1.MonitoringClass{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: monitoringv1alpha1.MonitoringClassSpec{
			ControllerName: "openeverest.io/monitoring-" + name,
			ExtensionManaged: &monitoringv1alpha1.ExtensionManagedFulfilment{
				Requires: monitoringv1alpha1.SourceRequirements{Metrics: metrics},
			},
		},
	}
}

func TestRequiredSources(t *testing.T) {
	t.Parallel()

	const (
		pm = v1alpha1.MonitoringExecutionModeProviderManaged
		em = v1alpha1.MonitoringExecutionModeExtensionManaged
	)
	tests := []struct {
		name    string
		entries []string
		objs    []client.Object
		want    Requirement
	}{
		{
			name:    "a not Accepted ExtensionManaged binding whose class requires metrics",
			entries: []string{"prometheus"},
			objs: []client.Object{
				monitoringBinding("prometheus", em, "prometheus", metav1.ConditionFalse),
				extensionManagedClass("prometheus", true),
			},
			want: Required,
		},
		{
			name:    "a metrics requirement wins over an unresolved entry",
			entries: []string{"prometheus", "pending"},
			objs: []client.Object{
				monitoringBinding("prometheus", em, "prometheus", metav1.ConditionTrue),
				extensionManagedClass("prometheus", true),
			},
			want: Required,
		},
		{
			name:    "only a ProviderManaged binding",
			entries: []string{"pmm"},
			objs: []client.Object{
				monitoringBinding("pmm", pm, "pmm", metav1.ConditionTrue),
				pmmClass(),
			},
			want: NotRequired,
		},
		{
			name:    "an ExtensionManaged class that does not require metrics",
			entries: []string{"logs"},
			objs: []client.Object{
				monitoringBinding("logs", em, "logs", metav1.ConditionTrue),
				extensionManagedClass("logs", false),
			},
			want: NotRequired,
		},
		{
			name:    "an entry without a binding",
			entries: []string{"pmm", "prometheus"},
			objs: []client.Object{
				monitoringBinding("pmm", pm, "pmm", metav1.ConditionTrue),
				pmmClass(),
			},
			want: Unresolved,
		},
		{
			name:    "a binding whose execution mode is not resolved yet",
			entries: []string{"prometheus"},
			objs:    []client.Object{monitoringBinding("prometheus", "", "", metav1.ConditionFalse)},
			want:    Unresolved,
		},
		{
			name:    "an ExtensionManaged binding whose class is missing",
			entries: []string{"prometheus"},
			objs:    []client.Object{monitoringBinding("prometheus", em, "prometheus", metav1.ConditionFalse)},
			want:    Unresolved,
		},
		{
			name: "no entries",
			want: NotRequired,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := monitoringTestContext(t, monitoredInstance(tc.entries...), tc.objs...)
			got, err := c.RequiredSources()
			require.NoError(t, err)
			assert.Equal(t, tc.want, got.Metrics)
		})
	}
}

func TestRequirementElse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		requirement Requirement
		current     bool
		want        bool
	}{
		{requirement: Required, current: false, want: true},
		{requirement: Required, current: true, want: true},
		{requirement: NotRequired, current: true, want: false},
		{requirement: NotRequired, current: false, want: false},
		{requirement: Unresolved, current: true, want: true},
		{requirement: Unresolved, current: false, want: false},
	}
	for _, tc := range tests {
		assert.Equal(t, tc.want, tc.requirement.Else(tc.current), "%s.Else(%t)", tc.requirement, tc.current)
	}
}

func TestMonitoringBindings(t *testing.T) {
	t.Parallel()

	const pm = v1alpha1.MonitoringExecutionModeProviderManaged
	tests := []struct {
		name            string
		binding         *monitoringv1alpha1.MonitoringBinding
		objs            []client.Object
		wantApply       bool
		wantEntry       bool
		wantDestination bool
		wantClass       bool
	}{
		{
			name:            "Accepted ProviderManaged binding with destination and class",
			binding:         monitoringBinding("pmm", pm, "pmm", metav1.ConditionTrue),
			objs:            []client.Object{pmmDestination("pmm-dest"), pmmClass()},
			wantApply:       true,
			wantEntry:       true,
			wantDestination: true,
			wantClass:       true,
		},
		{
			name: "Accepted for an older generation",
			binding: func() *monitoringv1alpha1.MonitoringBinding {
				b := monitoringBinding("pmm", pm, "pmm", metav1.ConditionTrue)
				b.Generation = 4
				return b
			}(),
			objs:            []client.Object{pmmDestination("pmm-dest"), pmmClass()},
			wantEntry:       true,
			wantDestination: true,
			wantClass:       true,
		},
		{
			name:            "not Accepted",
			binding:         monitoringBinding("pmm", pm, "pmm", metav1.ConditionFalse),
			objs:            []client.Object{pmmDestination("pmm-dest"), pmmClass()},
			wantEntry:       true,
			wantDestination: true,
			wantClass:       true,
		},
		{
			name:      "destination deleted",
			binding:   monitoringBinding("pmm", pm, "pmm", metav1.ConditionTrue),
			objs:      []client.Object{pmmClass()},
			wantEntry: true,
			wantClass: true,
		},
		{
			name:            "class deleted",
			binding:         monitoringBinding("pmm", pm, "pmm", metav1.ConditionTrue),
			objs:            []client.Object{pmmDestination("pmm-dest")},
			wantEntry:       true,
			wantDestination: true,
		},
		{
			name:            "ExtensionManaged",
			binding:         monitoringBinding("pmm", v1alpha1.MonitoringExecutionModeExtensionManaged, "pmm", metav1.ConditionTrue),
			objs:            []client.Object{pmmDestination("pmm-dest"), pmmClass()},
			wantEntry:       true,
			wantDestination: true,
			wantClass:       true,
		},
		{
			name:            "entry removed from the Instance",
			binding:         monitoringBinding("old", pm, "pmm", metav1.ConditionTrue),
			objs:            []client.Object{pmmDestination("old-dest"), pmmClass()},
			wantDestination: true,
			wantClass:       true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			other := monitoringBinding("pmm", pm, "pmm", metav1.ConditionTrue)
			other.Name, other.Labels[monitoringv1alpha1.BindingInstanceLabel] = "payments-pmm-00000000", "payments"
			c := monitoringTestContext(t, monitoredInstance("pmm"), append(tc.objs, tc.binding, other)...)

			got, err := c.MonitoringBindings()
			require.NoError(t, err)
			require.Len(t, got, 1, "only the Instance's own bindings are listed")
			rb := got[0]
			assert.Equal(t, tc.binding.Name, rb.Binding.Name)
			assert.Equal(t, tc.wantApply, rb.Apply)
			assert.Equal(t, tc.wantDestination, rb.Destination != nil)
			assert.Equal(t, tc.wantClass, rb.Class != nil)
			if tc.wantEntry {
				require.NotNil(t, rb.Entry)
				assert.Equal(t, "pmm", rb.Entry.Name)
			} else {
				assert.Nil(t, rb.Entry)
			}
			if tc.wantClass {
				assert.Equal(t, "pmm", rb.Integration())
			}
		})
	}
}
