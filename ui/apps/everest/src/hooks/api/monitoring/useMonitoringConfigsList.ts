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
  UseMutationOptions,
  useMutation,
  useQueries,
  useQuery,
  UseQueryResult,
} from '@tanstack/react-query';
import {
  createMonitoringConfigFn,
  deleteMonitoringConfigFn,
  getMonitoringConfigsFn,
  updateMonitoringConfigFn,
} from 'api/monitoring';
import {
  MonitoringConfig,
  MonitoringConfigCreateParams,
  MonitoringConfigUpdateParams,
} from 'shared-types/api.types';
import { useNamespaces } from '../namespaces';

export const MONITORING_CONFIGS_QUERY_KEY = 'monitoringConfigs';

const CLUSTER_NAME = 'main';

export const useMonitoringConfigsList = (
  cluster: string,
  namespace: string,
  options?: { refetchInterval?: number; enabled?: boolean }
) => {
  return useQuery({
    queryKey: [MONITORING_CONFIGS_QUERY_KEY, cluster, namespace],
    queryFn: () => getMonitoringConfigsFn(cluster, namespace),
    refetchInterval: options?.refetchInterval ?? 5_000,
    enabled: options?.enabled ?? (!!namespace && !!cluster),
  });
};

// Multi-namespace variant for the Settings page (fetches all namespaces in parallel)
export interface MonitoringConfigForNamespaceResult {
  namespace: string;
  queryResult: UseQueryResult<MonitoringConfig[], unknown>;
}

export const useMonitoringConfigsListMultiNs = (options?: {
  refetchInterval?: number;
}) => {
  const { data: namespaces = [], isLoading: isNamespacesLoading } =
    useNamespaces({
      refetchInterval: options?.refetchInterval ?? 10_000,
    });
  const queries = namespaces.map((namespace) => ({
    queryKey: [MONITORING_CONFIGS_QUERY_KEY, CLUSTER_NAME, namespace],
    queryFn: () => getMonitoringConfigsFn(CLUSTER_NAME, namespace),
    refetchInterval: options?.refetchInterval ?? 5_000,
  }));

  const queryResults = useQueries({ queries });

  const configs = queryResults.map((item, i) => ({
    namespace: namespaces[i],
    queryResult: item,
  }));

  return { configs, isNamespacesLoading };
};

export const useCreateMonitoringConfig = (
  options?: UseMutationOptions<
    MonitoringConfig,
    unknown,
    { namespace: string; payload: MonitoringConfigCreateParams },
    unknown
  >
) =>
  useMutation({
    mutationFn: ({
      namespace,
      payload,
    }: {
      namespace: string;
      payload: MonitoringConfigCreateParams;
    }) => createMonitoringConfigFn(CLUSTER_NAME, namespace, payload),
    ...options,
  });

export const useUpdateMonitoringConfig = (
  options?: UseMutationOptions<
    MonitoringConfig,
    unknown,
    { namespace: string; name: string; payload: MonitoringConfigUpdateParams },
    unknown
  >
) =>
  useMutation({
    mutationFn: ({
      namespace,
      name,
      payload,
    }: {
      namespace: string;
      name: string;
      payload: MonitoringConfigUpdateParams;
    }) => updateMonitoringConfigFn(CLUSTER_NAME, namespace, name, payload),
    ...options,
  });

export const useDeleteMonitoringConfig = (
  options?: UseMutationOptions<
    void,
    unknown,
    { namespace: string; name: string },
    unknown
  >
) =>
  useMutation({
    mutationFn: ({ namespace, name }: { namespace: string; name: string }) =>
      deleteMonitoringConfigFn(CLUSTER_NAME, namespace, name),
    ...options,
  });
