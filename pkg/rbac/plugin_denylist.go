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

package rbac

import "github.com/openeverest/openeverest/v2/pkg/plugintoken"

// spec001Resources are the resources managed by spec-001 Provider plugins.
// Daemon plugin tokens are unconditionally denied write access to these.
var spec001Resources = map[string]struct{}{
	ResourceDatabaseClusters:           {},
	ResourceDatabaseClusterBackups:     {},
	ResourceDatabaseClusterCredentials: {},
	ResourceDatabaseClusterRestores:    {},
	ResourceDatabaseEngines:            {},
	ResourceBackupStorages:             {},
	ResourceMonitoringInstances:        {},
}

// writeActions are the actions that constitute a write operation.
var writeActions = map[string]struct{}{
	ActionCreate: {},
	ActionUpdate: {},
	ActionDelete: {},
	ActionAll:    {},
}

// IsPluginWriteDenied returns true if the given subject is a plugin daemon
// token and the requested action+resource combination would write to a
// spec-001 managed resource.
func IsPluginWriteDenied(subject, resource, action string) bool {
	if !plugintoken.IsSubjectPluginToken(subject) {
		return false
	}
	if _, isWrite := writeActions[action]; !isWrite {
		return false
	}
	_, isSpec001 := spec001Resources[resource]
	return isSpec001
}
