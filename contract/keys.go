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
	// a user install it anyway. It names the core release on which the user
	// acknowledged the incompatibility, and the chart writes it on the
	// extension's objects as AcknowledgedAnnotation. Naming a release rather
	// than switching the check off lets AcknowledgementHolds expire it once
	// a core release changes the contract, so it never silently covers an
	// upgrade that changes what the user accepted.
	AcknowledgeValuePath = "compatibility.acknowledge.coreVersion"

	// AcknowledgedAnnotation carries, on the objects of an extension that
	// core interprets, the core release named by the extension chart's
	// AcknowledgeValuePath. Readers honour it while AcknowledgementHolds.
	AcknowledgedAnnotation = "openeverest.io/compatibility-acknowledged"

	// AcknowledgedExtensionsAnnotation carries, on every core CRD, the
	// acknowledgements the core chart publishes: comma-separated
	// <name>@<release> entries, where <name> is the name of the extension's
	// root object (its Provider, MonitoringClass or Plugin) and <release>
	// the core release it is acknowledged on. ParseAcknowledgedExtensions
	// reads it, and each entry is honoured while AcknowledgementHolds, like
	// AcknowledgedAnnotation. Publishing the list on the CRDs lets every
	// reader, an extension chart rendering included, see the same live list.
	AcknowledgedExtensionsAnnotation = "openeverest.io/acknowledged-extensions"

	// IncompatiblePluginsAnnotation carries, on the
	// plugins.extensions.openeverest.io CRD, the core chart's policy for a
	// Plugin that is ExtensionTooOld: IncompatiblePluginsBlock or
	// IncompatiblePluginsContinue.
	IncompatiblePluginsAnnotation = "openeverest.io/incompatible-plugins"
)

// Values of IncompatiblePluginsAnnotation. Only IncompatiblePluginsContinue
// changes what readers do: any other value, or none, leaves a Plugin that is
// ExtensionTooOld refused like every other extension object.
const (
	// IncompatiblePluginsBlock refuses a Plugin that is ExtensionTooOld
	// unless the incompatibility is acknowledged.
	IncompatiblePluginsBlock = "block"

	// IncompatiblePluginsContinue admits a Plugin that is ExtensionTooOld:
	// it stays installed but is not loaded, and the UI shows it as
	// incompatible. An admin can then upgrade core without waiting for every
	// plugin to catch up, and without uninstalling any.
	IncompatiblePluginsContinue = "continue"
)
