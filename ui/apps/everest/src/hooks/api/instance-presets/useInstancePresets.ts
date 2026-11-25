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
import { getInstancePresetsFn } from 'api/instancePresetApi';
import { InstancePreset, InstancePresetList } from 'shared-types/api.types';

export const INSTANCE_PRESETS_QUERY_KEY = 'instance-presets';

export const useInstancePresets = (
  clusterName: string,
  namespace: string,
  provider?: string
) =>
  useQuery<InstancePresetList, unknown, InstancePreset[]>({
    queryKey: [INSTANCE_PRESETS_QUERY_KEY, clusterName, namespace, provider],
    queryFn: () => getInstancePresetsFn(clusterName, namespace, provider),
    enabled: Boolean(clusterName) && Boolean(namespace) && Boolean(provider),
    select: (data) => data.items ?? [],
  });
