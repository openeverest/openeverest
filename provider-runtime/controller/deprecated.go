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

package controller

import (
	"github.com/openeverest/openeverest/v2/contract"
)

// The identifiers below moved to the contract packages, which core and the
// provider runtime both import, so that core no longer imports the provider
// runtime. They stay here for one release so that providers can move at
// their own pace.

// ProviderLabel is the label key used to identify which provider manages an
// Instance.
//
// Deprecated: use contract.ProviderLabel; removed in the next release.
const ProviderLabel = contract.ProviderLabel

// ProviderFinalizer is the finalizer the provider reconciler adds to an
// Instance when it picks the Instance up.
//
// Deprecated: use contract.ProviderFinalizer; removed in the next release.
const ProviderFinalizer = contract.ProviderFinalizer

// StatusFieldOwner returns the server-side-apply field manager under which
// the runtime of the named provider writes the status it shares with other
// writers.
//
// Deprecated: use contract.ProviderStatusFieldOwner; removed in the next
// release.
func StatusFieldOwner(provider string) string {
	return contract.ProviderStatusFieldOwner(provider)
}
