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

import {
  CreateRestorePayload,
  GetRestorePayload,
} from 'shared-types/restores.types';
import { api } from './api';

export const createInstanceRestoreFn = async (
  clusterName: string,
  namespace: string,
  payload: CreateRestorePayload
) => {
  const response = await api.post(
    `clusters/${clusterName}/namespaces/${namespace}/restores`,
    payload
  );

  return response.data;
};

export const getInstanceRestoresFn = async (
  clusterName: string,
  namespace: string,
  instanceName: string
) => {
  const response = await api.get<GetRestorePayload>(
    `clusters/${clusterName}/namespaces/${namespace}/instances/${instanceName}/restores`
  );

  return response.data;
};

export const deleteRestoreFn = async (
  clusterName: string,
  namespace: string,
  restoreName: string
) => {
  const response = await api.delete(
    `clusters/${clusterName}/namespaces/${namespace}/restores/${restoreName}`
  );

  return response.data;
};
