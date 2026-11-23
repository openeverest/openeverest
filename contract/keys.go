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

package contract

const (
	// VersionAnnotation carries a contract version. On the core CRDs and in
	// the core Chart.yaml it is the Version of that core, the top of its
	// window. On the objects of an extension that core interprets (Provider,
	// BackupClass, BackupStorageClass, MonitoringClass, Plugin) and in the
	// extension's Chart.yaml it is the stamp: the Version of the core the
	// extension was built against.
	VersionAnnotation = "openeverest.io/contract-version"

	// MinVersionAnnotation carries the MinVersion of a core, the bottom of its
	// window, on the core CRDs and in the core Chart.yaml.
	MinVersionAnnotation = "openeverest.io/min-contract-version"

	// UpgradableFromAnnotation lists, comma-separated, the core releases a
	// core can be upgraded from, on the core CRDs and in the core Chart.yaml.
	// It is empty when a release can only be installed fresh.
	UpgradableFromAnnotation = "openeverest.io/upgradable-from"

	// CoreVersionKey is the release version of core: a label on the core CRDs,
	// so it can be selected, and an annotation in the core Chart.yaml.
	CoreVersionKey = "openeverest.io/version"

	// AcknowledgeValuePath is the Helm value of an extension chart that lets
	// a user install it anyway on one core release, the one it is set to.
	// Pinning the acknowledgement to a release keeps it from silently
	// covering the next core upgrade.
	AcknowledgeValuePath = "compatibility.acknowledge.coreVersion"
)
