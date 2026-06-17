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
import type { PluginDescriptor } from './usePluginsForNamespace';

async function fetchAllPlugins(): Promise<PluginDescriptor[]> {
  const token = localStorage.getItem('everestToken');
  const headers: Record<string, string> = {};
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }
  const resp = await window.fetch('/v1/plugins', { headers });
  if (!resp.ok) return [];
  return resp.json();
}

/**
 * Returns the list of all enabled plugins the current user can access.
 * Used by the Plugins overview page.
 */
export const useAllPlugins = () => {
  return useQuery<PluginDescriptor[]>({
    queryKey: ['plugins', 'all'],
    queryFn: fetchAllPlugins,
    staleTime: 30_000,
  });
};
