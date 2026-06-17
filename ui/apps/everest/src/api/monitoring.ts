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
  MonitoringConfig,
  MonitoringConfigCreateParams,
  MonitoringConfigList,
  MonitoringConfigUpdateParams,
} from 'shared-types/api.types';

export const getMonitoringConfigsFn = async (
  cluster: string,
  namespace: string
): Promise<MonitoringConfig[]> => {
  const response = await api.get<MonitoringConfigList>(
    `clusters/${cluster}/namespaces/${namespace}/monitoring-configs`
  );
  return response.data?.items ?? [];
};

export const createMonitoringConfigFn = async (
  cluster: string,
  namespace: string,
  payload: MonitoringConfigCreateParams
): Promise<MonitoringConfig> => {
  const response = await api.post<MonitoringConfig>(
    `clusters/${cluster}/namespaces/${namespace}/monitoring-configs`,
    payload
  );
  return response.data;
};

export const deleteMonitoringConfigFn = async (
  cluster: string,
  namespace: string,
  name: string
): Promise<void> => {
  await api.delete(
    `clusters/${cluster}/namespaces/${namespace}/monitoring-configs/${name}`
  );
};

export const updateMonitoringConfigFn = async (
  cluster: string,
  namespace: string,
  name: string,
  payload: MonitoringConfigUpdateParams
): Promise<MonitoringConfig> => {
  const response = await api.patch<MonitoringConfig>(
    `clusters/${cluster}/namespaces/${namespace}/monitoring-configs/${name}`,
    payload
  );
  return response.data;
};
