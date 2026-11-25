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

// Keys core and the provider runtime both set or read on the objects they
// share.
const (
	// ProviderLabel names the Provider that manages an Instance. The
	// provider runtime sets it when it picks the Instance up.
	ProviderLabel = "core.openeverest.io/provider"

	// ProviderFinalizer is the finalizer the provider runtime adds to an
	// Instance when it picks the Instance up, and removes once cleanup is
	// done.
	ProviderFinalizer = "everest.percona.com/provider-finalizer"

	// DefinitionLabel names the definition a Secret or ConfigMap follows
	// within its source: a key of Provider.spec.secrets or
	// Provider.spec.configMaps, or an authentication mode of a
	// MonitoringClass or BackupStorageClass.
	DefinitionLabel = "openeverest.io/definition"
)

// ProviderStatusFieldOwner returns the server-side-apply field manager under
// which the runtime of the named provider writes the status it shares with
// other writers: the fields it owns on Instance.status and the Accepted
// condition of the BackupClasses the provider claims.
func ProviderStatusFieldOwner(provider string) string {
	return "openeverest-provider/" + provider
}
