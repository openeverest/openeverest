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

package helm

import "github.com/percona/everest/pkg/kubernetes"

// Values contains the different values that can be set in the Helm chart.
type Values struct {
	ClusterType        kubernetes.ClusterType
	VersionMetadataURL string
	DisableTelemetry   bool
}

// NewValues creates a map of values that can be used to render the Helm chart.
func NewValues(v Values) map[string]string {
	values := make(map[string]string)
	// the CLI does the preflight checks already,
	// no need to re-run them during the upgrade.
	values["upgrade.preflightChecks"] = "false"

	// No need to deploy the default DB namespace with the helm chart.
	// We will create it separately so that we're able to provide its
	// details as a separate step and also to avoid any potential issues.
	values["dbNamespace.enabled"] = "false"

	if v.ClusterType == kubernetes.ClusterTypeOpenShift {
		values["compatibility.openshift"] = "true"
		values["olm.install"] = "false"
		values["kube-state-metrics.rbac.create"] = "false"
		values["kube-state-metrics.securityContext.enabled"] = "false"
	}
	if v.VersionMetadataURL != "" {
		values["versionMetadataURL"] = v.VersionMetadataURL
	}
	if v.DisableTelemetry {
		values["telemetry"] = "false"
	}
	return values
}
