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

package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
)

func TestRun_MetricsSourcePolicy(t *testing.T) {
	t.Parallel()

	const noEffect = "spec.monitoring.sources.metrics.policy: Always has no effect: " +
		"the provider serves no metrics for the components of this Instance"
	withPolicy := func(in *v1alpha1.Instance, policy v1alpha1.MonitoringSourcePolicy) *v1alpha1.Instance {
		in.Spec.Monitoring = &v1alpha1.InstanceMonitoringSpec{Sources: &v1alpha1.InstanceMonitoringSources{
			Metrics: &v1alpha1.InstanceMetricsSource{Policy: policy},
		}}
		return in
	}
	tests := []struct {
		name       string
		instance   *v1alpha1.Instance
		monitoring *v1alpha1.ProviderMonitoring
		wantNote   bool
	}{
		{
			name:     "OnDemand without metrics",
			instance: withPolicy(replicaSet(nil), v1alpha1.MonitoringSourcePolicyOnDemand),
		},
		{
			name:     "Always on a provider without a monitoring contract",
			instance: withPolicy(replicaSet(nil), v1alpha1.MonitoringSourcePolicyAlways),
			wantNote: true,
		},
		{
			name:     "Always on a component with an exporter",
			instance: withPolicy(replicaSet(nil), v1alpha1.MonitoringSourcePolicyAlways),
			monitoring: &v1alpha1.ProviderMonitoring{ComponentTypes: map[string]v1alpha1.MonitoringComponentType{
				"mongod": {Metrics: v1alpha1.MetricsSourceExporter},
			}},
		},
		{
			name:     "Always where only another component type serves metrics",
			instance: withPolicy(replicaSet(nil), v1alpha1.MonitoringSourcePolicyAlways),
			monitoring: &v1alpha1.ProviderMonitoring{ComponentTypes: map[string]v1alpha1.MonitoringComponentType{
				"mongod": {Metrics: v1alpha1.MetricsSourceNone},
				"mongos": {Metrics: v1alpha1.MetricsSourceNative},
			}},
			wantNote: true,
		},
		{
			name: "Always on a component whose chosen type serves metrics",
			instance: withPolicy(
				sharded(map[string]v1alpha1.ComponentSpec{"proxy": {Type: "mongos"}}),
				v1alpha1.MonitoringSourcePolicyAlways,
			),
			monitoring: &v1alpha1.ProviderMonitoring{ComponentTypes: map[string]v1alpha1.MonitoringComponentType{
				"mongos": {Metrics: v1alpha1.MetricsSourceNative},
			}},
		},
		{
			name:     "Always without components checks every declared component",
			instance: withPolicy(&v1alpha1.Instance{}, v1alpha1.MonitoringSourcePolicyAlways),
			monitoring: &v1alpha1.ProviderMonitoring{ComponentTypes: map[string]v1alpha1.MonitoringComponentType{
				"mongod": {Metrics: v1alpha1.MetricsSourceExporter},
			}},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			provider := testProvider(t)
			provider.Spec.Monitoring = tc.monitoring
			v := &recordingValidator{}

			res := Run(t.Context(), v, controller.ValidationInput{Instance: tc.instance, Provider: provider})

			require.Empty(t, res.Errors, "the policy never fails validation")
			want := []string{"from the provider"}
			if tc.wantNote {
				want = append([]string{noEffect}, want...)
			}
			assert.Equal(t, want, res.Warnings)
		})
	}
}
