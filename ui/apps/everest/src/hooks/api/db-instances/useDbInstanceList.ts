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
  useQueries,
  useQuery,
  UseQueryOptions,
  UseQueryResult,
} from '@tanstack/react-query';
import { getDbInstanceConnectionFn, getDbInstancesFn } from 'api/instanceApi';
import { PerconaQueryOptions } from 'shared-types/query.types';
import {
  GetInstances,
  Instance,
  InstanceConnectionDetails,
} from 'shared-types/api.types';

export const DB_INSTANCES_QUERY_KEY = 'instances';
export const DB_INSTANCE_CONNECTION_QUERY_KEY = 'instanceConnection';

export const getDbInstancesQueryKey = (
  namespace: string,
  clusterName: string
) => [DB_INSTANCES_QUERY_KEY, namespace, clusterName] as const;

export const getDbInstanceConnectionQueryKey = (
  instanceName: string,
  namespace: string,
  clusterName: string
) =>
  [
    DB_INSTANCE_CONNECTION_QUERY_KEY,
    namespace,
    clusterName,
    instanceName,
  ] as const;

export interface DbInstanceForNamespaceResult {
  namespace: string;
  queryResult: UseQueryResult<Instance[], unknown>;
}

export const useDbInstanceList = (
  namespace: string,
  options?: PerconaQueryOptions<GetInstances, unknown, Instance[]>
) => {
  // TODO change to global use of cluster name during implementing multicluster feature
  const clusterName = 'main';
  return useQuery<GetInstances, unknown, Instance[]>({
    queryKey: getDbInstancesQueryKey(namespace, clusterName),
    queryFn: () => getDbInstancesFn(clusterName, namespace),
    refetchInterval: 5 * 1000,
    ...options,
    select: (instances) => {
      const selectedInstances = options?.select
        ? options.select(instances)
        : instances.items;

      return (selectedInstances ?? []).filter(
        (instance): instance is Instance => Boolean(instance)
      );
    },
  });
};

// TODO during adding backups don't forget to check timezone and CRON converting
export const instancesQuerySelect = (data: GetInstances): Instance[] =>
  (data.items ?? [])
    .filter((instance): instance is Instance => Boolean(instance))
    .sort((a, b) =>
      (a.metadata?.name ?? '').localeCompare(b.metadata?.name ?? '')
    );

export const useInstancesForNamespaces = (
  queryParams: Array<{
    namespace: string;
    options?: PerconaQueryOptions<GetInstances, unknown, Instance[]>;
  }>
) => {
  // TODO change to global use of cluster name during implementing multicluster feature
  const clusterName = 'main';

  const queries = queryParams.map<
    UseQueryOptions<GetInstances, unknown, Instance[]>
  >(({ namespace, options }) => {
    return {
      queryKey: getDbInstancesQueryKey(namespace, clusterName),
      queryFn: () => getDbInstancesFn(clusterName, namespace),
      refetchInterval: 5 * 1000,
      select: instancesQuerySelect,
      ...options,
    };
  });

  const queryResults = useQueries({ queries });
  const results: DbInstanceForNamespaceResult[] = queryResults.map(
    (item, i) => ({
      namespace: queryParams[i].namespace,
      queryResult: item,
    })
  );

  return results;
};

export const useInstanceConnection = (
  instanceName: string,
  namespace: string,
  options?: PerconaQueryOptions<
    InstanceConnectionDetails,
    unknown,
    InstanceConnectionDetails
  >
) => {
  // TODO change to global use of cluster name during implementing multicluster feature
  const clusterName = 'main';

  return useQuery<
    InstanceConnectionDetails,
    unknown,
    InstanceConnectionDetails
  >({
    queryKey: getDbInstanceConnectionQueryKey(
      instanceName,
      namespace,
      clusterName
    ),
    queryFn: () =>
      getDbInstanceConnectionFn(clusterName, namespace, instanceName),
    ...options,
  });
};
