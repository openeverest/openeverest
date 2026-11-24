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
import { ResolvedExtensionCatalog } from 'shared-types/extension-catalog.types';
import { useClusterName } from 'hooks/api/useClusterName';
import { buildExtensionCatalog } from './useExtensionCatalog.utils';

const EMPTY_CATALOG: ResolvedExtensionCatalog = {
  providerMeta: new Map(),
  providers: [],
};

// Fetches the extension catalog from the plugin-hub plugin and exposes a helper
// to resolve human-readable metadata for an installed provider, plus the list of
// every provider the catalog offers. When plugin-hub is not installed (or the
// catalog is unreachable), both are empty and callers fall back to the raw
// resource name.
export const useExtensionCatalog = () => {
  const clusterName = useClusterName();
  const query = useQuery({
    queryKey: ['extension-catalog', clusterName],
    queryFn: () => getExtensionCatalogFn(clusterName),
    retry: false,
    staleTime: 5 * 60 * 1000,
    refetchOnWindowFocus: false,
    select: (data) => buildExtensionCatalog(data, clusterName),
  });

  const { providerMeta, providers: catalogProviders } =
    query.data ?? EMPTY_CATALOG;

  return {
    ...query,
    providerMeta,
    catalogProviders,
    getProviderMeta: (providerName?: string | null) =>
      providerName ? providerMeta.get(providerName) : undefined,
  };
};
