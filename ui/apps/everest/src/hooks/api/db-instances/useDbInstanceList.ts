import {
  useQueries,
  useQuery,
  UseQueryOptions,
  UseQueryResult,
} from '@tanstack/react-query';
import { getInstanceConnectionFn, getInstancesFn } from 'api/instanceApi';
import { PerconaQueryOptions } from 'shared-types/query.types';
import { GetInstances, Instance, InstanceConnectionDetails } from 'types/api';

export const INSTANCES_QUERY_KEY = 'instances';
export const INSTANCE_CONNECTION_QUERY_KEY = 'instanceConnection';

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
    queryKey: [INSTANCES_QUERY_KEY, `${namespace}-${clusterName}`],
    queryFn: () => getInstancesFn(clusterName, namespace),
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
      queryKey: [INSTANCES_QUERY_KEY, `${namespace}-${clusterName}`],
      queryFn: () => getInstancesFn(clusterName, namespace),
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
    queryKey: [INSTANCE_CONNECTION_QUERY_KEY, instanceName],
    queryFn: () =>
      getInstanceConnectionFn(clusterName, namespace, instanceName),
    ...options,
  });
};
