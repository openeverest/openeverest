// everest
// Copyright (C) 2025 Percona LLC
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

package telemetry

import (
	"context"
	"sort"
	"strconv"
	"strings"

	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/percona/everest/pkg/common"
	"github.com/percona/everest/pkg/kubernetes"
	"github.com/percona/everest/pkg/rbac"
)

// Probe collects a set of telemetry properties.
// Each Probe is independent: a failure in one must not abort the others.
type Probe interface {
	Name() string
	Collect(ctx context.Context, k kubernetes.KubernetesConnector) (map[string]string, error)
}

// defaultProbes returns the probes used by the v1 schema.
func defaultProbes() []Probe {
	return []Probe{
		&dbClustersProbe{},
		&backupStoragesProbe{},
		&monitoringConfigsProbe{},
		&lbConfigsProbe{},
		&pspProbe{},
		&dataImportersProbe{},
		&splitHorizonDNSProbe{},
		&authProbe{},
		&providersProbe{},
	}
}

// ---------------------------------------------------------------------------
// dbClustersProbe: counts databases across all watched namespaces and derives
// feature-usage booleans. Engine-keyed property names use the open string
// `string(cl.Spec.Engine.Type)` so 2.0's `Spec.Provider` slots in unchanged.
// ---------------------------------------------------------------------------

type dbClustersProbe struct{}

func (dbClustersProbe) Name() string { return "db_clusters" }

func (dbClustersProbe) Collect(ctx context.Context, k kubernetes.KubernetesConnector) (map[string]string, error) {
	out := map[string]string{}

	namespaces, err := k.GetDBNamespaces(ctx)
	if err != nil {
		return nil, err
	}
	out["db_namespaces_count"] = strconv.Itoa(len(namespaces.Items))

	counts := map[string]int{}
	versions := map[string]map[string]struct{}{}

	var total int
	var anyPITR, anySchedules, anyExternal, anyDataImport, anyPSP bool

	for _, ns := range namespaces.Items {
		clusters, err := k.ListDatabaseClusters(ctx, ctrlclient.InNamespace(ns.GetName()))
		if err != nil {
			return nil, err
		}
		for _, cl := range clusters.Items {
			total++
			engine := string(cl.Spec.Engine.Type)
			counts[engine]++
			if v := cl.Spec.Engine.Version; v != "" {
				if versions[engine] == nil {
					versions[engine] = map[string]struct{}{}
				}
				versions[engine][v] = struct{}{}
			}
			if cl.Spec.Backup.PITR.Enabled {
				anyPITR = true
			}
			if len(cl.Spec.Backup.Schedules) > 0 {
				anySchedules = true
			}
			if exposeType := string(cl.Spec.Proxy.Expose.Type); exposeType != "" && exposeType != "internal" {
				anyExternal = true
			}
			if cl.Spec.DataSource != nil {
				anyDataImport = true
			}
			if cl.Spec.PodSchedulingPolicyName != "" {
				anyPSP = true
			}
		}
	}

	out["db_clusters_total"] = strconv.Itoa(total)
	for engine, count := range counts {
		out["engine_"+engine+"_count"] = strconv.Itoa(count)
	}
	for engine, set := range versions {
		out["engine_"+engine+"_versions"] = joinSortedSet(set)
	}
	out["feat_pitr"] = boolStr(anyPITR)
	out["feat_scheduled_backups"] = boolStr(anySchedules)
	out["feat_external_access"] = boolStr(anyExternal)
	out["feat_data_import_in_use"] = boolStr(anyDataImport)
	out["feat_psp_in_use"] = boolStr(anyPSP)

	return out, nil
}

// ---------------------------------------------------------------------------
// Object-count probes
// ---------------------------------------------------------------------------

type backupStoragesProbe struct{}

func (backupStoragesProbe) Name() string { return "backup_storages" }
func (backupStoragesProbe) Collect(ctx context.Context, k kubernetes.KubernetesConnector) (map[string]string, error) {
	list, err := k.ListBackupStorages(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]string{"backup_storages_count": strconv.Itoa(len(list.Items))}, nil
}

type monitoringConfigsProbe struct{}

func (monitoringConfigsProbe) Name() string { return "monitoring_configs" }
func (monitoringConfigsProbe) Collect(ctx context.Context, k kubernetes.KubernetesConnector) (map[string]string, error) {
	list, err := k.ListMonitoringConfigs(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]string{
		"monitoring_configs_count": strconv.Itoa(len(list.Items)),
		"feat_monitoring_in_use":   boolStr(len(list.Items) > 0),
	}, nil
}

type lbConfigsProbe struct{}

func (lbConfigsProbe) Name() string { return "load_balancer_configs" }
func (lbConfigsProbe) Collect(ctx context.Context, k kubernetes.KubernetesConnector) (map[string]string, error) {
	list, err := k.ListLoadBalancerConfigs(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]string{"load_balancer_configs_count": strconv.Itoa(len(list.Items))}, nil
}

type pspProbe struct{}

func (pspProbe) Name() string { return "pod_scheduling_policies" }
func (pspProbe) Collect(ctx context.Context, k kubernetes.KubernetesConnector) (map[string]string, error) {
	list, err := k.ListPodSchedulingPolicies(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]string{"pod_scheduling_policies_count": strconv.Itoa(len(list.Items))}, nil
}

type dataImportersProbe struct{}

func (dataImportersProbe) Name() string { return "data_importers" }
func (dataImportersProbe) Collect(ctx context.Context, k kubernetes.KubernetesConnector) (map[string]string, error) {
	list, err := k.ListDataImporters(ctx)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(list.Items))
	for _, di := range list.Items {
		names = append(names, di.GetName())
	}
	sort.Strings(names)
	return map[string]string{
		"data_importers_count":     strconv.Itoa(len(list.Items)),
		"data_importers_installed": strings.Join(names, ","),
	}, nil
}

type splitHorizonDNSProbe struct{}

func (splitHorizonDNSProbe) Name() string { return "split_horizon_dns" }
func (splitHorizonDNSProbe) Collect(ctx context.Context, k kubernetes.KubernetesConnector) (map[string]string, error) {
	list, err := k.ListSplitHorizonDNSConfigs(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]string{"split_horizon_dns_configs_count": strconv.Itoa(len(list.Items))}, nil
}

// ---------------------------------------------------------------------------
// authProbe reports whether RBAC and OIDC are configured, without leaking
// any actual policy content.
// ---------------------------------------------------------------------------

type authProbe struct{}

func (authProbe) Name() string { return "auth" }
func (authProbe) Collect(ctx context.Context, k kubernetes.KubernetesConnector) (map[string]string, error) {
	out := map[string]string{
		"auth_rbac_enabled":    "false",
		"auth_oidc_configured": "false",
	}

	rbacCM, err := k.GetConfigMap(ctx, ctrlclient.ObjectKey{
		Namespace: common.SystemNamespace,
		Name:      common.EverestRBACConfigMapName,
	})
	if err == nil && rbac.IsEnabled(rbacCM) {
		out["auth_rbac_enabled"] = "true"
	}

	settings, err := k.GetEverestSettings(ctx)
	if err == nil && settings.OIDCConfigRaw != "" {
		out["auth_oidc_configured"] = "true"
	}

	return out, nil
}

// ---------------------------------------------------------------------------
// providersProbe: 1.x emits an empty value to reserve the property name for
// 2.0 (where it will enumerate installed Provider CRs).
// ---------------------------------------------------------------------------

type providersProbe struct{}

func (providersProbe) Name() string { return "providers" }
func (providersProbe) Collect(_ context.Context, _ kubernetes.KubernetesConnector) (map[string]string, error) {
	return map[string]string{"providers_installed": ""}, nil
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func joinSortedSet(set map[string]struct{}) string {
	out := make([]string, 0, len(set))
	for v := range set {
		out = append(out, v)
	}
	sort.Strings(out)
	return strings.Join(out, ",")
}
