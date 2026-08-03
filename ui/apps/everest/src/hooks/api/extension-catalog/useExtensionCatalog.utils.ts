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

import { PLUGIN_HUB_NAME } from 'api/extension-catalog';
import {
  RawExtensionCatalogEntry,
  RawExtensionIndex,
  ResolvedExtensionMeta,
} from 'shared-types/extension-catalog.types';

// Resolves the install version of a catalog entry from its default chart
// channel. Every access is guarded because the raw entry is untrusted.
const resolveVersion = (
  entry: RawExtensionCatalogEntry
): string | undefined => {
  const chart = entry.artifacts?.chart;
  const channels = chart?.channels;
  if (!channels) {
    return undefined;
  }
  const channelName = chart?.defaultChannel ?? Object.keys(channels)[0];
  return channelName ? channels[channelName]?.version : undefined;
};

// The plugin-hub backend rewrites cross-origin icon URLs to a path relative to
// its own mount (e.g. `api/icon/<sha256>`) to avoid CSP failures, so we should
// never see http(s):// values here — reject them defensively so a
// misconfigured or malicious catalog can't inject external image loads that
// CSP would block anyway.
const resolveIconSrc = (rawIcon?: string): string | undefined => {
  if (!rawIcon) {
    return undefined;
  }
  if (rawIcon.startsWith('http://') || rawIcon.startsWith('https://')) {
    return undefined;
  }
  if (rawIcon.startsWith('data:') || rawIcon.startsWith('/v1/plugins/')) {
    return rawIcon;
  }
  const stripped = rawIcon.startsWith('/') ? rawIcon.slice(1) : rawIcon;
  return `/v1/plugins/${PLUGIN_HUB_NAME}/${stripped}`;
};

// Single narrowing point for catalog entries. Rejects anything that is not a
// well-formed provider entry — unknown `type` values (including new ones the
// hub might introduce) are tolerated by returning null rather than crashing.
// Downstream consumers work with `ResolvedExtensionMeta`, which is strict.
const normalizeEntry = (
  raw: RawExtensionCatalogEntry
): {
  meta: ResolvedExtensionMeta;
  providerName?: string;
} | null => {
  if (!raw?.name || raw.type !== 'provider') {
    return null;
  }
  const meta: ResolvedExtensionMeta = {
    name: raw.name,
    displayName: raw.displayName || raw.name,
    description: raw.description,
    icon: resolveIconSrc(raw.icon),
    version: resolveVersion(raw),
    categories: raw.categories,
    maturity: raw.maturity,
  };
  return { meta, providerName: raw.provider?.providerName };
};

// Builds a lookup keyed by every identifier a provider can be matched by
// (provider CR name and the catalog entry name). Used as the react-query
// `select` for the extension catalog so components consume resolved metadata.
export const buildProviderMetaMap = (
  index: RawExtensionIndex | null
): Map<string, ResolvedExtensionMeta> => {
  const map = new Map<string, ResolvedExtensionMeta>();
  const entries = index?.extensions;
  if (!entries) {
    return map;
  }
  for (const raw of entries) {
    const normalized = normalizeEntry(raw);
    if (!normalized) {
      continue;
    }
    if (normalized.providerName) {
      map.set(normalized.providerName, normalized.meta);
    }
    map.set(normalized.meta.name, normalized.meta);
  }
  return map;
};
