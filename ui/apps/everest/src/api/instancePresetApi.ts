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

import { InstancePreset, InstancePresetList } from 'shared-types/api.types';
import { api } from './api';

export const getInstancePresetsFn = async (
  clusterName: string,
  provider?: string
) => {
  const query = provider ? `?provider=${encodeURIComponent(provider)}` : '';
  const response = await api.get<InstancePresetList>(
    `clusters/${clusterName}/instance-presets${query}`
  );
  return response.data;
};

// Resolve fills the preset spec with namespace defaults (e.g. default
// StorageClass) so it can be used verbatim as an Instance spec.
export const resolveInstancePresetFn = async (
  clusterName: string,
  name: string,
  namespace: string
) => {
  const response = await api.get<InstancePreset>(
    `clusters/${clusterName}/instance-presets/${name}/resolve?namespace=${encodeURIComponent(
      namespace
    )}`
  );
  return response.data;
};
