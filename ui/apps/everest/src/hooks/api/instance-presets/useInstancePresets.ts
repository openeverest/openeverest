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
import {
  getInstancePresetsFn,
  resolveInstancePresetFn,
} from 'api/instancePresetApi';
import { PerconaQueryOptions } from 'shared-types/query.types';
import { InstancePreset, InstancePresetList } from 'shared-types/api.types';

export const INSTANCE_PRESETS_QUERY_KEY = 'instance-presets';
export const RESOLVE_INSTANCE_PRESET_QUERY_KEY = 'resolve-instance-preset';

export const useInstancePresets = (
  clusterName: string,
  provider?: string,
  options?: PerconaQueryOptions<
    InstancePresetList,
    unknown,
    InstancePreset[]
  >
) =>
  useQuery<InstancePresetList, unknown, InstancePreset[]>({
    queryKey: [INSTANCE_PRESETS_QUERY_KEY, clusterName, provider],
    queryFn: () => getInstancePresetsFn(clusterName, provider),
    enabled: Boolean(clusterName) && Boolean(provider),
    ...options,
    select: (data) =>
      (data.items ?? []).filter((preset): preset is InstancePreset =>
        Boolean(preset)
      ),
  });

// Resolve is a GET modelled as a query keyed on cluster+name+namespace so the
// spec re-resolves automatically when the namespace changes and react-query
// dedupes/cancels stale fetches (no manual race handling needed).
export const useResolveInstancePreset = (
  clusterName: string,
  name: string,
  namespace: string | undefined,
  options?: PerconaQueryOptions<InstancePreset, unknown, InstancePreset>
) =>
  useQuery<InstancePreset, unknown, InstancePreset>({
    queryKey: [RESOLVE_INSTANCE_PRESET_QUERY_KEY, clusterName, name, namespace],
    queryFn: () => resolveInstancePresetFn(clusterName, name, namespace ?? ''),
    enabled: Boolean(clusterName) && Boolean(name) && Boolean(namespace),
    ...options,
  });
