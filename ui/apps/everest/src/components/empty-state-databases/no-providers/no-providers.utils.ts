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
import type {
  ExtensionMaturity,
  ResolvedExtensionMeta,
} from 'shared-types/extension-catalog.types';
import type { TileLink } from 'components/empty-state-databases/provider-tiles/tile-action-area';
import { EXTENSION_HUB_WEBSITE_URL } from './no-providers.constants';

const MATURITY_RANK: Record<ExtensionMaturity, number> = {
  stable: 0,
  beta: 1,
  alpha: 2,
  deprecated: 3,
};

const maturityRank = (provider: ResolvedExtensionMeta) =>
  MATURITY_RANK[provider.maturity ?? 'alpha'];

// Picks a few catalog providers to suggest on an empty install. Curated
// `preferredNames` come first, in the given order, when the catalog offers
// them. Remaining slots are ranked by trust, since the hub has no popularity
// signal: verified first, then maturity, then catalog order. One provider per
// primary category is taken first so the suggestions show the breadth of the
// platform (database, storage, AI, ...) rather than several flavours of the
// same thing.
export const pickSpotlightProviders = (
  providers: ResolvedExtensionMeta[],
  limit: number,
  preferredNames: string[] = []
): ResolvedExtensionMeta[] => {
  const eligible = providers.filter(
    (provider) => provider.maturity !== 'deprecated'
  );
  const preferred = preferredNames.flatMap(
    (name) => eligible.find((provider) => provider.name === name) ?? []
  );
  const ranked = eligible
    .filter((provider) => !preferred.includes(provider))
    .sort(
      (a, b) =>
        Number(Boolean(b.verified)) - Number(Boolean(a.verified)) ||
        maturityRank(a) - maturityRank(b)
    );

  const picked: ResolvedExtensionMeta[] = [...preferred];
  const coveredCategories = new Set(
    preferred.flatMap((provider) => provider.categories?.[0] ?? [])
  );
  for (const provider of ranked) {
    const category = provider.categories?.[0];
    if (category && coveredCategories.has(category)) {
      continue;
    }
    if (category) {
      coveredCategories.add(category);
    }
    picked.push(provider);
  }
  const rest = ranked.filter((provider) => !picked.includes(provider));
  return [...picked, ...rest].slice(0, limit);
};

// Links into the plugin-hub UI when it is registered in this host; otherwise
// (hub UI not loaded, e.g. incompatible version) to the public Extension Hub.
export const buildHubLink = (
  hubUiAvailable: boolean,
  subPath: string
): TileLink =>
  hubUiAvailable
    ? { to: `/plugins/${PLUGIN_HUB_NAME}/${subPath}` }
    : { href: EXTENSION_HUB_WEBSITE_URL };
