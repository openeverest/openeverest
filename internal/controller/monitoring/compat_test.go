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

package monitoring

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
)

// eligibilityProvider is a Provider with the given monitoring component
// types and component type kinds. Every monitoring component type is a
// component type, with no kind unless kinds names one.
func eligibilityProvider(types map[string]corev1alpha1.MonitoringComponentType, kinds map[string]string) *corev1alpha1.Provider {
	p := newProvider("example", nil, kinds, &corev1alpha1.ProviderMonitoring{ComponentTypes: types})
	for t := range types {
		if _, ok := p.Spec.ComponentTypes[t]; !ok {
			p.Spec.ComponentTypes[t] = corev1alpha1.ComponentType{}
		}
	}
	return p
}

func TestExtensionManagedEligibility(t *testing.T) {
	t.Parallel()

	const classGeneration = int64(3)
	class := func(requires monitoringv1alpha1.SourceRequirements, podMetadata bool) *monitoringv1alpha1.MonitoringClass {
		em := &monitoringv1alpha1.ExtensionManagedFulfilment{Requires: requires}
		if podMetadata {
			em.PodMetadata = &monitoringv1alpha1.PodMetadataPolicy{AnnotationPrefixes: []string{"example.com/"}}
		}
		return &monitoringv1alpha1.MonitoringClass{
			ObjectMeta: metav1.ObjectMeta{Name: "agent", Generation: classGeneration},
			Spec: monitoringv1alpha1.MonitoringClassSpec{
				ControllerName:   "example.com/agent",
				ExtensionManaged: em,
				AuthModes:        []common.AuthMode{{Name: common.AuthModeAnonymous}},
			},
		}
	}
	connections := monitoringv1alpha1.SourceRequirements{Connection: true}
	databases := monitoringv1alpha1.SourceRequirements{Connection: true, Kinds: []string{"mongodb", "postgresql"}}
	connectable := corev1alpha1.MonitoringComponentType{Connection: corev1alpha1.ConnectionSupported}
	annotatable := corev1alpha1.MonitoringComponentType{Connection: corev1alpha1.ConnectionSupported, PodMetadata: corev1alpha1.PodMetadataSupported}
	compatible := func(disruption monitoringv1alpha1.Disruption) monitoringv1alpha1.ProviderCompatibility {
		return monitoringv1alpha1.ProviderCompatibility{
			Name:            "example",
			Mode:            corev1alpha1.MonitoringExecutionModeExtensionManaged,
			Disruption:      disruption,
			ClassGeneration: classGeneration,
		}
	}
	incompatible := func(reason, message string) monitoringv1alpha1.ProviderCompatibility {
		return monitoringv1alpha1.ProviderCompatibility{Name: "example", Reason: reason, Message: message, ClassGeneration: classGeneration}
	}

	tests := []struct {
		name     string
		class    *monitoringv1alpha1.MonitoringClass
		provider *corev1alpha1.Provider
		want     monitoringv1alpha1.ProviderCompatibility
	}{
		{
			name:  "no requires.kinds: a type without a kind is eligible",
			class: class(connections, false),
			provider: eligibilityProvider(
				map[string]corev1alpha1.MonitoringComponentType{"engine": connectable},
				nil,
			),
			want: compatible(monitoringv1alpha1.DisruptionNone),
		},
		{
			name:  "no requires.kinds: any kind is eligible",
			class: class(connections, false),
			provider: eligibilityProvider(
				map[string]corev1alpha1.MonitoringComponentType{"pxc": connectable},
				map[string]string{"pxc": "mysql"},
			),
			want: compatible(monitoringv1alpha1.DisruptionNone),
		},
		{
			name:  "listed kind on one of several component types",
			class: class(databases, false),
			provider: eligibilityProvider(
				map[string]corev1alpha1.MonitoringComponentType{"pxc": connectable, "haproxy": connectable, "postgresql": connectable},
				map[string]string{"pxc": "mysql", "haproxy": "haproxy", "postgresql": "postgresql"},
			),
			want: compatible(monitoringv1alpha1.DisruptionNone),
		},
		{
			name:  "no component type of a listed kind",
			class: class(databases, false),
			provider: eligibilityProvider(
				map[string]corev1alpha1.MonitoringComponentType{"pxc": connectable, "haproxy": connectable},
				map[string]string{"pxc": "mysql", "haproxy": "haproxy"},
			),
			want: incompatible(monitoringv1alpha1.ReasonKindUnsupported,
				`no component is of a kind MonitoringClass "agent" supports (mongodb, postgresql); found haproxy, mysql`,
			),
		},
		{
			name:  "a type without a kind matches no requires.kinds",
			class: class(databases, false),
			provider: eligibilityProvider(
				map[string]corev1alpha1.MonitoringComponentType{"engine": connectable},
				nil,
			),
			want: incompatible(monitoringv1alpha1.ReasonKindUnsupported,
				`no component is of a kind MonitoringClass "agent" supports (mongodb, postgresql); no component type declares a kind`,
			),
		},
		{
			name:  "a type of a listed kind outside the monitoring contract is not eligible",
			class: class(databases, false),
			provider: eligibilityProvider(
				map[string]corev1alpha1.MonitoringComponentType{"pxc": connectable},
				map[string]string{"pxc": "mysql", "mongod": "mongodb"},
			),
			want: incompatible(monitoringv1alpha1.ReasonIncompatible, `no component satisfies the requirements of MonitoringClass "agent"`),
		},
		{
			name:  "listed kind without the required connections",
			class: class(databases, false),
			provider: eligibilityProvider(
				map[string]corev1alpha1.MonitoringComponentType{"pxc": connectable, "postgresql": {Metrics: corev1alpha1.MetricsSourceExporter}},
				map[string]string{"pxc": "mysql", "postgresql": "postgresql"},
			),
			want: incompatible(monitoringv1alpha1.ReasonIncompatible, `no component satisfies the requirements of MonitoringClass "agent"`),
		},
		{
			name:  "pod metadata requested and rendered",
			class: class(connections, true),
			provider: eligibilityProvider(
				map[string]corev1alpha1.MonitoringComponentType{"mongod": annotatable},
				map[string]string{"mongod": "mongodb"},
			),
			want: compatible(monitoringv1alpha1.DisruptionRollingRestart),
		},
		{
			name:  "pod metadata rendered by one of several eligible component types",
			class: class(connections, true),
			provider: eligibilityProvider(
				map[string]corev1alpha1.MonitoringComponentType{"mongod": annotatable, "mongos": connectable},
				map[string]string{"mongod": "mongodb", "mongos": "mongodb"},
			),
			want: compatible(monitoringv1alpha1.DisruptionRollingRestart),
		},
		{
			name:  "pod metadata requested and not rendered",
			class: class(connections, true),
			provider: eligibilityProvider(
				map[string]corev1alpha1.MonitoringComponentType{
					"mongod": connectable,
					"mongos": {Connection: corev1alpha1.ConnectionSupported, PodMetadata: corev1alpha1.PodMetadataNone},
				},
				map[string]string{"mongod": "mongodb", "mongos": "mongodb"},
			),
			want: incompatible(monitoringv1alpha1.ReasonPodMetadataUnsupported,
				`MonitoringClass "agent" requests pod annotations, which Provider "example" does not render into the pods of mongod, mongos (podMetadata is not Supported)`,
			),
		},
		{
			name:  "pod metadata is checked on the component types of a listed kind",
			class: class(databases, true),
			provider: eligibilityProvider(
				map[string]corev1alpha1.MonitoringComponentType{"pxc": annotatable, "mongod": connectable},
				map[string]string{"pxc": "mysql", "mongod": "mongodb"},
			),
			want: incompatible(monitoringv1alpha1.ReasonPodMetadataUnsupported,
				`MonitoringClass "agent" requests pod annotations, which Provider "example" does not render into the pods of mongod (podMetadata is not Supported)`,
			),
		},
		{
			name:  "pod metadata rendered by the component type of a listed kind",
			class: class(databases, true),
			provider: eligibilityProvider(
				map[string]corev1alpha1.MonitoringComponentType{"pxc": connectable, "mongod": annotatable},
				map[string]string{"pxc": "mysql", "mongod": "mongodb"},
			),
			want: compatible(monitoringv1alpha1.DisruptionRollingRestart),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, providerCompatibility(tt.provider, tt.class))
		})
	}
}
