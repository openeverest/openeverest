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
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/monitoring"
)

func annotationDiscoveryClass(name string, prefixes ...string) *monitoringv1alpha1.MonitoringClass {
	class := extensionManagedClass(name, false)
	class.Spec.ExtensionManaged.Requires.Connection = true
	class.Spec.ExtensionManaged.PodMetadata = &monitoringv1alpha1.PodMetadataPolicy{AnnotationPrefixes: prefixes}
	return class
}

// requestingPodMetadata is an ExtensionManaged binding of entry whose class
// controller requests annotations for component.
func requestingPodMetadata(entry, class string, accepted metav1.ConditionStatus, component string, annotations map[string]string) *monitoringv1alpha1.MonitoringBinding {
	b := monitoringBinding(entry, v1alpha1.MonitoringExecutionModeExtensionManaged, class, accepted)
	b.Status.PodMetadata = []monitoringv1alpha1.ComponentPodMetadata{{Component: component, Annotations: annotations}}
	return b
}

// firstBindingEntry returns the entry whose binding name sorts first; binding
// names carry a hash, so the order is not the entries' order.
func firstBindingEntry(a, b string) string {
	if monitoring.BindingName("orders", b) < monitoring.BindingName("orders", a) {
		return b
	}
	return a
}

func TestMonitoringPodAnnotations(t *testing.T) {
	t.Parallel()

	coroot := annotationDiscoveryClass("coroot", "coroot.com/")
	tests := []struct {
		name    string
		entries []string
		objs    []client.Object
		want    map[string]string
	}{
		{
			name:    "accepted binding: only allowed keys of the component",
			entries: []string{"coroot"},
			objs: []client.Object{coroot, func() *monitoringv1alpha1.MonitoringBinding {
				b := requestingPodMetadata("coroot", "coroot", metav1.ConditionTrue, "engine", map[string]string{
					"coroot.com/mongodb-scrape": "true",
					"sidecar.istio.io/inject":   "true",
				})
				b.Status.PodMetadata = append(b.Status.PodMetadata, monitoringv1alpha1.ComponentPodMetadata{
					Component: "proxy", Annotations: map[string]string{"coroot.com/mongodb-scrape": "false"},
				})
				return b
			}()},
			want: map[string]string{"coroot.com/mongodb-scrape": "true"},
		},
		{
			name:    "pending binding keeps contributing",
			entries: []string{"coroot"},
			objs: []client.Object{coroot, requestingPodMetadata("coroot", "coroot", "", "engine",
				map[string]string{"coroot.com/mongodb-scrape": "true"})},
			want: map[string]string{"coroot.com/mongodb-scrape": "true"},
		},
		{
			name:    "rejected binding contributes nothing",
			entries: []string{"coroot"},
			objs: []client.Object{coroot, requestingPodMetadata("coroot", "coroot", metav1.ConditionFalse, "engine",
				map[string]string{"coroot.com/mongodb-scrape": "true"})},
			want: map[string]string{},
		},
		{
			name: "binding whose entry was removed contributes nothing",
			objs: []client.Object{coroot, requestingPodMetadata("coroot", "coroot", metav1.ConditionTrue, "engine",
				map[string]string{"coroot.com/mongodb-scrape": "true"})},
			want: map[string]string{},
		},
		{
			name:    "class without a pod metadata policy contributes nothing",
			entries: []string{"prometheus"},
			objs: []client.Object{extensionManagedClass("prometheus", true), requestingPodMetadata("prometheus", "prometheus", metav1.ConditionTrue, "engine",
				map[string]string{"coroot.com/mongodb-scrape": "true"})},
			want: map[string]string{},
		},
		{
			name:    "conflicting keys: the binding whose name sorts first wins",
			entries: []string{"a", "b"},
			objs: []client.Object{
				coroot,
				requestingPodMetadata("a", "coroot", metav1.ConditionTrue, "engine", map[string]string{"coroot.com/x": "a"}),
				requestingPodMetadata("b", "coroot", metav1.ConditionTrue, "engine", map[string]string{"coroot.com/x": "b"}),
			},
			want: map[string]string{"coroot.com/x": firstBindingEntry("a", "b")},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := monitoringTestContext(t, monitoredInstance(tt.entries...), tt.objs...)

			got, err := c.MonitoringPodAnnotations("engine")

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRequiredSourcesConnections(t *testing.T) {
	t.Parallel()

	const em = v1alpha1.MonitoringExecutionModeExtensionManaged
	tests := []struct {
		name    string
		entries []string
		objs    []client.Object
		want    Requirement
	}{
		{
			name:    "class requiring connections",
			entries: []string{"coroot"},
			objs: []client.Object{
				annotationDiscoveryClass("coroot", "coroot.com/"),
				monitoringBinding("coroot", em, "coroot", metav1.ConditionFalse),
			},
			want: Required,
		},
		{
			name:    "class requiring metrics only",
			entries: []string{"prometheus"},
			objs: []client.Object{
				extensionManagedClass("prometheus", true),
				monitoringBinding("prometheus", em, "prometheus", metav1.ConditionTrue),
			},
			want: NotRequired,
		},
		{
			name:    "entry without a binding",
			entries: []string{"coroot"},
			want:    Unresolved,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := monitoringTestContext(t, monitoredInstance(tt.entries...), tt.objs...)

			req, err := c.RequiredSources()

			require.NoError(t, err)
			assert.Equal(t, tt.want, req.Connections)
		})
	}
}
