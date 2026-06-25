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

import { useQuery } from '@tanstack/react-query';
import { getExtensionCatalogFn } from 'api/extension-catalog';
import {
  ExtensionCatalogEntry,
  ExtensionIndex,
  ResolvedExtensionMeta,
} from 'shared-types/extension-catalog.types';

// Resolves the install version of a catalog entry from its default chart channel.
const resolveVersion = (entry: ExtensionCatalogEntry): string | undefined => {
  const chart = entry.artifacts?.chart;
  if (!chart?.channels) {
    return undefined;
  }
  const channelName = chart.defaultChannel ?? Object.keys(chart.channels)[0];
  return channelName ? chart.channels[channelName]?.version : undefined;
};

const toResolvedMeta = (
  entry: ExtensionCatalogEntry
): ResolvedExtensionMeta => ({
  name: entry.name,
  displayName: entry.displayName || entry.name,
  description: entry.description,
  icon: entry.icon || undefined,
  version: resolveVersion(entry),
  categories: entry.categories,
  maturity: entry.maturity,
});

// Builds a lookup keyed by the identifiers a provider/plugin can be matched by.
// Providers are matched by `provider.providerName` (which equals the Provider
// CR metadata.name), with the entry `name` as a fallback key.
const buildProviderMetaMap = (
  index: ExtensionIndex | null
): Map<string, ResolvedExtensionMeta> => {
  const map = new Map<string, ResolvedExtensionMeta>();
  if (!index?.extensions) {
    return map;
  }
  for (const entry of index.extensions) {
    if (entry.type !== 'provider') {
      continue;
    }
    const meta = toResolvedMeta(entry);
    if (entry.provider?.providerName) {
      map.set(entry.provider.providerName, meta);
    }
    if (entry.name) {
      map.set(entry.name, meta);
    }
  }
  return map;
};

// Fetches the extension catalog from the plugin-hub plugin and exposes a helper
// to resolve human-readable metadata for an installed provider. When plugin-hub
// is not installed (or the catalog is unreachable), the map is empty and callers
// fall back to the raw resource name.
export const useExtensionCatalog = () => {
  const query = useQuery({
    queryKey: ['extension-catalog'],
    queryFn: getExtensionCatalogFn,
    retry: false,
    staleTime: 5 * 60 * 1000,
    refetchOnWindowFocus: false,
    select: buildProviderMetaMap,
  });

  const providerMeta = query.data ?? new Map<string, ResolvedExtensionMeta>();

  return {
    ...query,
    providerMeta,
    getProviderMeta: (providerName?: string | null) =>
      providerName ? providerMeta.get(providerName) : undefined,
  };
};
