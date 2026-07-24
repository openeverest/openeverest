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

import type {
  GetProviderMeta,
  ProviderDisplay,
  ResolveProviderDisplayInput,
} from './provider-identity.types';

// Resolves the identifier and human-readable label for a Provider, together
// with the optional catalog metadata used to render richer visuals when the
// plugin-hub is installed. Callers use { name, label, meta } directly to
// compose ProviderIdentity or their own container markup.
export const resolveProviderDisplay = (
  provider: ResolveProviderDisplayInput,
  getMeta: GetProviderMeta
): ProviderDisplay => {
  const name = provider?.metadata?.name ?? '';
  const meta = getMeta(name);
  const label = meta?.displayName || name;
  return { name, label, meta };
};
