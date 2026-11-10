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
  MonitoringDestination,
  MonitoringDestinationList,
  MonitoringDestinationPayload,
} from 'shared-types/monitoring-destinations.types';

const destinationsPath = (cluster: string, namespace: string) =>
  `clusters/${cluster}/namespaces/${namespace}/monitoring-destinations`;

export const getMonitoringDestinationsFn = async (
  cluster: string,
  namespace: string
) => {
  const response = await api.get<MonitoringDestinationList>(
    destinationsPath(cluster, namespace)
  );
  return response.data;
};

export const getMonitoringDestinationFn = async (
  cluster: string,
  namespace: string,
  name: string
) => {
  const response = await api.get<MonitoringDestination>(
    `${destinationsPath(cluster, namespace)}/${name}`
  );
  return response.data;
};

export const createMonitoringDestinationFn = async (
  cluster: string,
  namespace: string,
  destination: MonitoringDestinationPayload
) => {
  const response = await api.post<MonitoringDestination>(
    destinationsPath(cluster, namespace),
    destination
  );
  return response.data;
};

export const updateMonitoringDestinationFn = async (
  cluster: string,
  namespace: string,
  destination: MonitoringDestinationPayload
) => {
  const response = await api.put<MonitoringDestination>(
    `${destinationsPath(cluster, namespace)}/${destination.metadata?.name ?? ''}`,
    destination,
    // A destination changed meanwhile is reported on the form.
    { disableNotifications: (e) => e.status === 409 }
  );
  return response.data;
};

export const deleteMonitoringDestinationFn = async (
  cluster: string,
  namespace: string,
  name: string
) => {
  await api.delete(`${destinationsPath(cluster, namespace)}/${name}`);
};
