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

import { useCallback } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { getResolvedPresetsFn } from 'api/instancePresetApi';
import { useClusterName } from 'hooks/api/useClusterName';
import { useNamespaces } from 'hooks/api/namespaces';
import { Provider } from 'shared-types/api.types';
import { RESOLVED_PRESETS_QUERY_KEY } from './useInstancePresets';

// Warms the presets cache on navigation intent (e.g. hovering a provider tile)
// so the create-instance form can render its final layout without a reflow.
// Presets are namespace-resolved, so this warms the namespace the form
// preselects (the first one).
export const usePrefetchInstancePresets = () => {
  const queryClient = useQueryClient();
  const clusterName = useClusterName();
  const { data: namespaces = [] } = useNamespaces();
  const namespace = namespaces[0];

  return useCallback(
    (provider?: Provider) => {
      const name = provider?.metadata?.name;
      if (!clusterName || !namespace || !name) {
        return;
      }
      queryClient.prefetchQuery({
        queryKey: [RESOLVED_PRESETS_QUERY_KEY, clusterName, namespace, name],
        queryFn: () => getResolvedPresetsFn(clusterName, namespace, name),
      });
    },
    [queryClient, clusterName, namespace]
  );
};
