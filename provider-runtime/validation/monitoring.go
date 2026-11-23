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

// checkMonitoring warns when the metrics source policy is Always but no
// component of the Instance can serve metrics. It is a warning, not an error,
// because monitoring never blocks the database.
func checkMonitoring(in *v1alpha1.Instance, spec *v1alpha1.ProviderSpec) []string {
	if in.GetMetricsSourcePolicy() != v1alpha1.MonitoringSourcePolicyAlways {
		return nil
	}
	if spec.Monitoring != nil {
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
		for t := range types {
			if m := spec.Monitoring.ComponentTypes[t].Metrics; m == v1alpha1.MetricsSourceNative || m == v1alpha1.MetricsSourceExporter {
				return nil
			}
		}
	}
	return []string{"spec.monitoring.sources.metrics.policy: Always has no effect: the provider serves no metrics for the components of this Instance"}
}
