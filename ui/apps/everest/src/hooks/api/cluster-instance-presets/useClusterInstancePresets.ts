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
import { getClusterInstancePresetsFn } from 'api/clusterInstancePresetApi';
import { PerconaQueryOptions } from 'shared-types/query.types';
import {
  ClusterInstancePreset,
  ClusterInstancePresetList,
} from 'shared-types/api.types';

export const CLUSTER_INSTANCE_PRESETS_QUERY_KEY = 'cluster-instance-presets';

export const useClusterInstancePresets = (
  clusterName: string,
  provider?: string,
  options?: PerconaQueryOptions<
    ClusterInstancePresetList,
    unknown,
    ClusterInstancePreset[]
  >
) =>
  useQuery<ClusterInstancePresetList, unknown, ClusterInstancePreset[]>({
    queryKey: [CLUSTER_INSTANCE_PRESETS_QUERY_KEY, clusterName, provider],
    queryFn: () => getClusterInstancePresetsFn(clusterName, provider),
    enabled: Boolean(clusterName) && Boolean(provider),
    ...options,
    select: (data) =>
      (data.items ?? []).filter((preset): preset is ClusterInstancePreset =>
        Boolean(preset)
      ),
  });
