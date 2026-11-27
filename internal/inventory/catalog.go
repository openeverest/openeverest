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

package inventory

import "k8s.io/apimachinery/pkg/util/sets"

// The names of the extensions the openeverest-official catalog publishes
// (openeverest/hub, index/index.json): the Provider name each provider entry
// installs (provider.providerName), the Plugin name of each plugin entry and
// the MonitoringClasses of each monitoring integration
// (integration.monitoring.classes), plus the BackupStorageClasses core ships.
//
// Telemetry reports a provider, plugin or class by name only when it is listed
// here; every other one is counted without its name. A name missing here therefore
// costs detail, never privacy. Add the name when the catalog adds an entry.
//
//nolint:gochecknoglobals
var (
	officialProviders = sets.New(
		"provider-altinity-clickhouse",
		"provider-cassandra",
		"provider-cloudnative-pg",
		"provider-kserve",
		"mariadb",
		"milvus",
		"mssql",
		"neo4j",
		"opensearch",
		"provider-percona-postgresql",
		"percona-server-mongodb",
		"percona-server-mysql",
		"percona-xtradb-cluster",
		"provider-seaweedfs",
		"tidb",
		"valkey",
	)
	officialPlugins = sets.New(
		"plugin-audit",
		"plugin-bench",
		"plugin-branching",
		"plugin-hub",
		"plugin-inspector",
		"plugin-metrics",
		"plugin-migrations",
		"plugin-mongodb-explorer",
		"plugin-user-management",
	)
	officialMonitoringClasses = sets.New(
		"coroot",
		"opentelemetry",
		"pmm",
		"prometheus",
		"releem",
		"victoriametrics",
	)
	officialBackupStorageClasses = sets.New(
		"s3",
	)
)
