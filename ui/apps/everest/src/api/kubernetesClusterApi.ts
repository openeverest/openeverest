// everest
// Copyright (C) 2023 Percona LLC
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
import { api } from './api';
import {
  GetKubernetesClusterInfoPayload,
  GetKubernetesClusterResourcesInfoPayload,
} from 'shared-types/kubernetes.types';

export const getKubernetesClusterInfoFn = async (clusterName: string) => {
  const response = await api.get<GetKubernetesClusterInfoPayload>(
    `clusters/${clusterName}/cluster-info`
  );

  return response.data;
};

export const getKubernetesClusterResourcesInfo = async (
  clusterName: string
) => {
  const response = await api.get<GetKubernetesClusterResourcesInfoPayload>(
    `clusters/${clusterName}/resources`
  );
  return response.data;
};
