import { useQuery } from '@tanstack/react-query';
import { getDbInstanceFn } from 'api/instanceApi';
import { PerconaQueryOptions } from 'shared-types/query.types';
import { GetDbInstancePayload, Instance } from 'types/api';

export const DB_INSTANCE_QUERY_KEY = 'instance';

export const useDbInstance = (
  namespace: string,
  instanceName: string,
  options?: PerconaQueryOptions<GetDbInstancePayload, unknown, Instance>
) => {
  // TODO implement RBAC
  // const { canRead } = useRBACPermissions(
  //     'database-clusters',
  //     `${namespace}/${dbClusterName}`
  //   );
  // TODO change to global use of cluster name during implementing multicluster feature
  const clusterName = 'main';

  return useQuery<GetDbInstancePayload, unknown, Instance>({
    queryKey: [DB_INSTANCE_QUERY_KEY, namespace, instanceName],
    queryFn: () => getDbInstanceFn(clusterName, namespace, instanceName),
    enabled: !!namespace && !!instanceName,
    refetchInterval: 5 * 1000,
    ...options,
  });
};
