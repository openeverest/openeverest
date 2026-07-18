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

import { api } from 'api/api';
import {
  GetInstancePresetsPayload,
  InstancePreset,
} from 'shared-types/api.types';

export const getInstancePresetsFn = async (
  clusterName: string,
  provider?: string
): Promise<GetInstancePresetsPayload> => {
  const params = provider ? { provider } : {};
  const { data } = await api.get<GetInstancePresetsPayload>(
    `clusters/${clusterName}/instance-presets`,
    { params }
  );
  return data;
};

export const getInstancePresetFn = async (
  clusterName: string,
  name: string
): Promise<InstancePreset> => {
  const { data } = await api.get<InstancePreset>(
    `clusters/${clusterName}/instance-presets/${name}`
  );
  return data;
};

export const resolveInstancePresetFn = async (
  clusterName: string,
  name: string,
  namespace: string
): Promise<InstancePreset> => {
  const { data } = await api.get<InstancePreset>(
    `clusters/${clusterName}/instance-presets/${name}/resolve`,
    { params: { namespace } }
  );
  return data;
};
