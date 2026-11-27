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

import "github.com/openeverest/openeverest/v2/api/core/v1alpha1"

// checkMonitoring warns when a source policy is Always but no component of
// the Instance can serve that source. It is a warning, not an error, because
// monitoring never blocks the database.
func checkMonitoring(in *v1alpha1.Instance, spec *v1alpha1.ProviderSpec) []string {
	var warnings []string
	types := instanceComponentTypes(in, spec)
	if in.GetMetricsSourcePolicy() == v1alpha1.MonitoringSourcePolicyAlways && !anyServes(spec, types, servesMetrics) {
		warnings = append(warnings, "spec.monitoring.sources.metrics.policy: Always has no effect: the provider serves no metrics for the components of this Instance")
	}
	if in.GetLogsSourcePolicy() == v1alpha1.MonitoringSourcePolicyAlways && !anyServes(spec, types, servesLogs) {
		warnings = append(warnings, "spec.monitoring.sources.logs.policy: Always has no effect: the provider publishes no logs for the components of this Instance")
	}
	return warnings
}

func servesMetrics(ct v1alpha1.MonitoringComponentType) bool {
	return ct.Metrics == v1alpha1.MetricsSourceNative || ct.Metrics == v1alpha1.MetricsSourceExporter
}

func servesLogs(ct v1alpha1.MonitoringComponentType) bool {
	return ct.Logs == v1alpha1.LogsSourceNative || ct.Logs == v1alpha1.LogsSourceSidecar
}

func anyServes(spec *v1alpha1.ProviderSpec, types map[string]struct{}, serves func(v1alpha1.MonitoringComponentType) bool) bool {
	if spec.Monitoring == nil {
		return false
	}
	for t := range types {
		if serves(spec.Monitoring.ComponentTypes[t]) {
			return true
		}
	}
	return false
}

// instanceComponentTypes returns the types of the Instance's components, or
// of every component the Provider declares when the Instance lists none.
func instanceComponentTypes(in *v1alpha1.Instance, spec *v1alpha1.ProviderSpec) map[string]struct{} {
	types := make(map[string]struct{})
	for name, component := range in.Spec.Components {
		if component.Type != "" {
			types[component.Type] = struct{}{}
		} else if declared, ok := spec.Components[name]; ok {
			types[declared.Type] = struct{}{}
		}
	}
	if len(in.Spec.Components) == 0 {
		for _, declared := range spec.Components {
			types[declared.Type] = struct{}{}
		}
	}
	return types
}
