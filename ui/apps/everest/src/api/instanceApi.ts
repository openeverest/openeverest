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
  CreateInstanceArgType,
  GetInstances,
  InstanceConnectionDetails,
} from 'types/api';
import { api } from './api';

export const createInstanceFn = async (
  clusterName: string,
  instanceName: string,
  providerName: string,
  namespace: string,
  data: CreateInstanceArgType['spec']
) => {
  const payload: CreateInstanceArgType = {
    apiVersion: 'core.openeverest.io/v1alpha1',
    kind: 'Instance',
    // TODO this TS error should gone after BE types updates
    // @ts-ignore
    metadata: { name: instanceName },
    spec: { provider: providerName, ...data },
  };
  const response = await api.post(
    `clusters/${clusterName}/namespaces/${namespace}/instances`,
    payload
  );

  return response.data;
};

export const getInstancesFn = async (
  clusterName: string,
  namespace: string
) => {
  const response = await api.get<GetInstances>(
    `/clusters/${clusterName}/namespaces/${namespace}/instances`
  );
  return response.data;
};

export const getInstanceConnectionFn = async (
  clusterName: string,
  namespace: string,
  instanceName: string
) => {
  const response = await api.get<InstanceConnectionDetails>(
    `/clusters/${clusterName}/namespaces/${namespace}/instances/${instanceName}/connection`
  );
  return response.data;
};
