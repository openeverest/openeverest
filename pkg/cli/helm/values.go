package helm

import "github.com/openeverest/openeverest/v2/pkg/kubernetes"

// Values contains the different values that can be set in the Helm chart.
type Values struct {
	ClusterType        kubernetes.ClusterType
	VersionMetadataURL string
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
	return values
}
