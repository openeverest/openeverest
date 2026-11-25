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
import { getInstancePresetsFn } from 'api/instancePresetApi';
import { usePrefetchClusterInstancePresets } from 'hooks/api/cluster-instance-presets';
import { useNamespaces } from 'hooks/api/namespaces';
import { useClusterName } from 'hooks/api/useClusterName';
import { Provider } from 'shared-types/api.types';
import { INSTANCE_PRESETS_QUERY_KEY } from './useInstancePresets';

// Warms the presets the create wizard opens with, on navigation intent (e.g.
// hovering a provider tile): the provider's cluster presets and those of the
// namespace the wizard defaults to. The form then renders its final layout
// without a reflow.
export const usePrefetchPresets = () => {
  const queryClient = useQueryClient();
  const clusterName = useClusterName();
  const prefetchClusterPresets = usePrefetchClusterInstancePresets();
  const { data: namespaces = [] } = useNamespaces();
  // A new instance starts in the first namespace (useDatabaseFormSync).
  const defaultNamespace = namespaces[0];

  return useCallback(
    (provider?: Provider) => {
      prefetchClusterPresets(provider);

      const name = provider?.metadata?.name;
      if (!clusterName || !name || !defaultNamespace) {
        return;
      }
      queryClient.prefetchQuery({
        queryKey: [
          INSTANCE_PRESETS_QUERY_KEY,
          clusterName,
          defaultNamespace,
          name,
        ],
        queryFn: () =>
          getInstancePresetsFn(clusterName, defaultNamespace, name),
      });
    },
    [queryClient, clusterName, defaultNamespace, prefetchClusterPresets]
  );
};
