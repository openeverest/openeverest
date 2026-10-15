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
import { getResolvedPresetsFn } from 'api/instancePresetApi';
import { PerconaQueryOptions } from 'shared-types/query.types';
import { ResolvedPreset, ResolvedPresetList } from 'shared-types/api.types';

export const RESOLVED_PRESETS_QUERY_KEY = 'resolved-presets';

// Items are already resolved for the namespace, so the picked preset is read
// straight from this list. Keyed on namespace so switching it refetches.
export const useInstancePresets = (
  clusterName: string,
  namespace: string | undefined,
  provider?: string,
  options?: PerconaQueryOptions<ResolvedPresetList, unknown, ResolvedPreset[]>
) =>
  useQuery<ResolvedPresetList, unknown, ResolvedPreset[]>({
    queryKey: [RESOLVED_PRESETS_QUERY_KEY, clusterName, namespace, provider],
    queryFn: () => getResolvedPresetsFn(clusterName, namespace ?? '', provider),
    enabled: Boolean(clusterName) && Boolean(namespace) && Boolean(provider),
    ...options,
    select: (data) =>
      (data.items ?? []).filter((preset): preset is ResolvedPreset =>
        Boolean(preset)
      ),
  });
