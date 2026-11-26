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

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	extensionsv1alpha1 "github.com/openeverest/openeverest/v2/api/extensions/v1alpha1"
)

const (
	prefix = "openeverest_"

	// The labels that name an Instance, its namespace and its provider are
	// the identity labels providers put on database metrics. They don't
	// collide with the namespace and instance labels Prometheus gives every
	// scraped series, and they join OpenEverest metrics with database ones.
	labelNamespace = "openeverest_namespace"
	labelInstance  = "openeverest_instance"
	labelProvider  = "openeverest_provider"
)

func newDesc(name, help string, labels ...string) *prometheus.Desc {
	return prometheus.NewDesc(prefix+name, help, labels, nil)
}

//nolint:gochecknoglobals
var (
	buildInfo = newDesc("build_info",
		"OpenEverest build information. Always 1.",
		"version", "contract_version",
	)

	extensionInfo = newDesc("extension_info",
		"An installed extension (InstalledExtension). Always 1.",
		"type", "name", "version",
	)
	extensionPhase = newDesc("extension_phase",
		"Phase of an installed extension: 1 for the current phase, 0 for the others.",
		"type", "name", "phase",
	)
	providerInfo = newDesc("provider_info",
		"A Provider, whether installed from the catalog or with Helm. Always 1.",
		labelProvider, "contract_version",
	)
	providerCondition = newDesc("provider_condition",
		"Condition of a Provider: 1 for the current status, 0 for the others.",
		labelProvider, "condition", "status",
	)

	classInfo = newDesc("class_info",
		"A MonitoringClass, BackupClass or BackupStorageClass. controller is the MonitoringClass controllerName, "+
			"or the Provider of a provider-managed BackupClass. Always 1.",
		"kind", "name", "controller",
	)
	classAccepted = newDesc("class_accepted",
		"Accepted condition of a class: 1 for the current status, 0 for the others.",
		"kind", "name", "status",
	)
	backupStorageInfo = newDesc("backup_storage_info",
		"A BackupStorage. Always 1.",
		labelNamespace, "name", "class",
	)
	monitoringDestinationInfo = newDesc("monitoring_destination_info",
		"A MonitoringDestination. Always 1.",
		labelNamespace, "name", "class",
	)
	monitoringDestinationReady = newDesc("monitoring_destination_ready",
		"Ready condition of a MonitoringDestination: 1 for the current status, 0 for the others.",
		labelNamespace, "name", "status",
	)

	instanceInfo = newDesc("instance_info",
		"An Instance. version is the version it runs, or the one its spec asks for until the provider reports. Always 1.",
		labelNamespace, labelInstance, labelProvider, "version",
	)
	instancePhase = newDesc("instance_phase",
		"Lifecycle phase of an Instance: 1 for the current phase, 0 for the others. "+
			"The phase says nothing about health: alert on openeverest_instance_condition.",
		labelNamespace, labelInstance, "phase",
	)
	instanceCondition = newDesc("instance_condition",
		"Summary condition of an Instance: 1 for the current status, 0 for the others.",
		labelNamespace, labelInstance, "condition", "status",
	)
	instanceComponentReplicas = newDesc("instance_component_replicas",
		"Pods of an Instance component as the provider reports them: desired, current (not terminating) and ready.",
		labelNamespace, labelInstance, "component", "kind",
	)
	instanceResourceRequests = newDesc("instance_resource_requests",
		"Resources an Instance component requests in its spec, times its replicas. CPU in cores, memory and "+
			"storage in bytes. A lower bound: components that come only from defaults, sidecars and exporters "+
			"are not counted.",
		labelNamespace, labelInstance, "component", "resource",
	)
	instanceResourceLimits = newDesc("instance_resource_limits",
		"Resource limits an Instance component sets in its spec, times its replicas. CPU in cores, memory in bytes.",
		labelNamespace, labelInstance, "component", "resource",
	)
	instancePITRAvailable = newDesc("instance_pitr_available",
		"1 while the point-in-time recovery window on a backup storage is available, 0 otherwise. "+
			"Present only for storages with PITR enabled.",
		labelNamespace, labelInstance, "storage",
	)
	instanceLastSuccessfulBackup = newDesc("instance_last_successful_backup_timestamp_seconds",
		"Unix time the latest successful Backup of an Instance to a backup storage completed.",
		labelNamespace, labelInstance, "storage",
	)

	backupsCount = newDesc("backups",
		"Number of Backups by state. instance is empty for backups imported from external data.",
		labelNamespace, labelInstance, "storage", "state",
	)
	restoresCount = newDesc("restores",
		"Number of Restores by state.",
		labelNamespace, labelInstance, "state",
	)

	allDescs = []*prometheus.Desc{
		buildInfo,
		extensionInfo, extensionPhase, providerInfo, providerCondition,
		classInfo, classAccepted, backupStorageInfo, monitoringDestinationInfo, monitoringDestinationReady,
		instanceInfo, instancePhase, instanceCondition, instanceComponentReplicas,
		instanceResourceRequests, instanceResourceLimits, instancePITRAvailable, instanceLastSuccessfulBackup,
		backupsCount, restoresCount,
	}

	conditionStatuses = []string{"true", "false", "unknown"}

	instancePhases = []string{
		string(corev1alpha1.InstancePhasePending),
		string(corev1alpha1.InstancePhaseProvisioned),
		string(corev1alpha1.InstancePhaseTerminating),
	}

	extensionPhases = []string{
		string(extensionsv1alpha1.InstalledExtensionPhasePending),
		string(extensionsv1alpha1.InstalledExtensionPhaseInstalled),
		string(extensionsv1alpha1.InstalledExtensionPhaseUpgrading),
		string(extensionsv1alpha1.InstalledExtensionPhaseFailed),
		string(extensionsv1alpha1.InstalledExtensionPhaseUninstalling),
	}

	// instanceConditions are the summary conditions every client reads; the
	// checks and operations behind them would multiply the series per
	// Instance without adding an alertable signal.
	instanceConditions = map[string]bool{
		corev1alpha1.ConditionReady:          true,
		corev1alpha1.ConditionAvailable:      true,
		corev1alpha1.ConditionStalled:        true,
		corev1alpha1.ConditionNeedsAttention: true,
		corev1alpha1.ConditionUnattended:     true,
	}
)
