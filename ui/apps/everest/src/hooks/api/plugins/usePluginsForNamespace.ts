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
import { getAuthToken } from 'api/session-token';

export interface EnabledPluginDescriptor {
  name: string;
  displayName: string;
  description?: string;
  version?: string;
  vendor?: string;
  icon?: string;
  bundleUrl: string;
  extensionPoints?: {
    type: string;
    label?: string;
    path?: string;
    icon?: string;
    providers?: string[];
  }[];
}

async function fetchPluginsForNamespace(
  namespace: string
): Promise<EnabledPluginDescriptor[]> {
  const token = getAuthToken();
  const headers: Record<string, string> = {};
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }
  const resp = await window.fetch(
    `/v1/plugins?namespace=${encodeURIComponent(namespace)}`,
    { headers }
  );
  if (!resp.ok) return [];
  return resp.json();
}

/**
 * Returns the list of plugins enabled for the given namespace via an
 * `InstalledExtension` (covered by `spec.plugin.namespaces[]`, or installed
 * with `scope: Cluster` and `allowClusterScope`). Used to filter
 * extension-point contributions (cluster-detail tabs, form sections, etc.)
 * to only plugins the namespace operator has explicitly enabled.
 *
 * When `namespace` is empty/undefined, the query is disabled and an empty
 * array is returned (no filter applied).
 */
export const usePluginsForNamespace = (namespace: string | undefined) => {
  return useQuery<EnabledPluginDescriptor[]>({
    queryKey: ['plugins', 'namespace', namespace],
    queryFn: () => fetchPluginsForNamespace(namespace!),
    enabled: Boolean(namespace),
    staleTime: 30_000,
  });
};
