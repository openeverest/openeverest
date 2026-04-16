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

import { useQuery } from '@tanstack/react-query';
import { getDbInstanceFn } from 'api/instanceApi';
import { PerconaQueryOptions } from 'shared-types/query.types';
import { GetDbInstancePayload, Instance } from 'shared-types/api.types';

export const DB_INSTANCE_QUERY_KEY = 'instance';

export const getDbInstanceQueryKey = (
  namespace: string,
  instanceName: string,
  clusterName: string
) => [DB_INSTANCE_QUERY_KEY, namespace, clusterName, instanceName] as const;

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
    queryKey: getDbInstanceQueryKey(namespace, instanceName, clusterName),
    queryFn: () => getDbInstanceFn(clusterName, namespace, instanceName),
    enabled: options?.enabled ?? true /*&& canRead*/,
    ...options,
  });
};
