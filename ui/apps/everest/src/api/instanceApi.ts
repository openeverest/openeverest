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
  CreateDbInstancePayload,
  GetDbInstancePayload,
  GetInstances,
  Instance,
  InstanceConnectionDetails,
} from 'shared-types/api.types';
import { api } from './api';

export const createDbInstanceFn = async (
  clusterName: string,
  instanceName: string,
  namespace: string,
  data: CreateDbInstancePayload['spec']
) => {
  const payload: CreateDbInstancePayload = {
    apiVersion: 'core.openeverest.io/v1alpha1',
    kind: 'Instance',
    // TODO this TS error should gone after BE types updates
    // @ts-ignore
    metadata: { name: instanceName },
    spec: {
      ...data,
    },
  };
  const response = await api.post(
    `clusters/${clusterName}/namespaces/${namespace}/instances`,
    payload
  );

  return response.data;
};

export const getDbInstancesFn = async (
  clusterName: string,
  namespace: string
) => {
  const response = await api.get<GetInstances>(
    `clusters/${clusterName}/namespaces/${namespace}/instances`
  );
  return response.data;
};

export const deleteDbInstanceFn = async (
  clusterName: string,
  dbInstanceName: string,
  namespace: string
  // cleanupBackupStorage: boolean
) => {
  const response = await api.delete<Instance>(
    `clusters/${clusterName}/namespaces/${namespace}/instances/${dbInstanceName}`
  );
  return response.data;
};

export const getDbInstanceFn = async (
  clusterName: string,
  namespace: string,
  instanceName: string
) => {
  const response = await api.get<GetDbInstancePayload>(
    `clusters/${clusterName}/namespaces/${namespace}/instances/${instanceName}`
  );
  return response.data;
};

export const updateDbInstanceFn = async (
  clusterName: string,
  namespace: string,
  instanceName: string,
  data: Instance
) => {
  const response = await api.put<Instance>(
    `clusters/${clusterName}/namespaces/${namespace}/instances/${instanceName}`,
    data
  );
  return response.data;
};

export const getDbInstanceConnectionFn = async (
  clusterName: string,
  namespace: string,
  instanceName: string
) => {
  const response = await api.get<InstanceConnectionDetails>(
    `clusters/${clusterName}/namespaces/${namespace}/instances/${instanceName}/connection`
  );
  return response.data;
};
