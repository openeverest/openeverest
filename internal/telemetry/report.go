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

package telemetry

import (
	"strconv"
	"strings"

	"github.com/openeverest/openeverest/v2/internal/inventory"
)

// ReportSchemaVersion is the version of the report's keys. Keys are only ever
// added, so it changes only on a breaking change. Version 1 was the Percona
// Everest report.
const ReportSchemaVersion = "2"

// Installation identifies the installation sending the report.
type Installation struct {
	// ID is the UID of the namespace OpenEverest is installed in.
	ID           string
	BuildChannel string
	Version      string
	Contract     string
}

// Report is one heartbeat: a flat set of string properties, the shape every
// event collector accepts.
type Report map[string]string

// NewReport builds the report of an installation managing fleet.
func NewReport(inst Installation, fleet inventory.Fleet) Report {
	r := Report{
		"event":               "heartbeat",
		"schema_version":      ReportSchemaVersion,
		"instance_id":         inst.ID,
		"build_channel":       inst.BuildChannel,
		"openeverest_version": inst.Version,
		"contract_version":    inst.Contract,

		"clusters_total": itoa(fleet.Clusters),
		"clusters_stale": itoa(fleet.StaleClusters),

		"providers_distinct":     itoa(len(fleet.Providers)),
		"providers":              strings.Join(fleet.Providers, ","),
		"provider_installations": itoa(fleet.ProviderInstallations),
		"plugins_total":          itoa(fleet.PluginInstallations),

		"instances_total":           itoa(fleet.Instances),
		"instances_ready":           itoa(fleet.InstancesReady),
		"instances_stalled":         itoa(fleet.InstancesStalled),
		"namespaces_with_instances": itoa(fleet.NamespacesWithInstances),

		"cpu_requested_cores":     strconv.FormatFloat(fleet.CPURequestedCores, 'f', 3, 64),
		"memory_requested_bytes":  strconv.FormatFloat(fleet.MemoryRequestedBytes, 'f', 0, 64),
		"storage_requested_bytes": strconv.FormatFloat(fleet.StorageRequestedBytes, 'f', 0, 64),

		"backups_succeeded":             itoa(fleet.BackupsSucceeded),
		"backups_failed":                itoa(fleet.BackupsFailed),
		"restores_total":                itoa(fleet.Restores),
		"instances_with_backup_storage": itoa(fleet.InstancesWithBackupStorage),
		"instances_with_pitr":           itoa(fleet.InstancesWithPITR),

		"monitoring_classes":      itoa(fleet.MonitoringClasses),
		"monitoring_destinations": itoa(fleet.MonitoringDestinations),
		"monitoring_bindings":     itoa(fleet.MonitoringBindings),
		"backup_classes":          itoa(fleet.BackupClasses),
		"backup_storage_classes":  itoa(fleet.BackupStorageClasses),
		"backup_storages":         itoa(fleet.BackupStorages),
	}
	for provider, n := range fleet.InstancesByProvider {
		r["provider."+provider+".instances"] = itoa(n)
	}
	for _, plugin := range fleet.Plugins {
		r["plugin."+plugin] = "1"
	}
	return r
}

func itoa(n int) string {
	return strconv.Itoa(n)
}
