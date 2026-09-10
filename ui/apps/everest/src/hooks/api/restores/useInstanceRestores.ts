// everest
// Copyright (C) 2023 Percona LLC
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
  useMutation,
  UseMutationOptions,
  useQuery,
} from '@tanstack/react-query';
import {
  createInstanceRestoreFn,
  deleteRestoreFn,
  getInstanceRestoresFn,
} from 'api/restores';
import { generateShortUID } from 'utils/generateShortUID';
import { useRBACPermissions } from 'hooks/rbac';
import { PerconaQueryOptions } from 'shared-types/query.types';
import {
  CreateRestorePayload,
  GetRestorePayload,
  Restore,
  RestoreCreateVariables,
} from 'shared-types/restores.types';
import { buildRestoreDataSource } from './restore-data-source';
import { mapRestore } from './map-restore';

export const RESTORES_QUERY_KEY = 'restores';

export const getRestoresListQueryKey = (
  clusterName: string,
  namespace: string,
  instanceName: string
) => [RESTORES_QUERY_KEY, clusterName, namespace, instanceName] as const;

export const useCreateInstanceRestore = (
  clusterName: string,
  namespace: string,
  options?: UseMutationOptions<
    unknown,
    unknown,
    RestoreCreateVariables,
    unknown
  >
) =>
  useMutation({
    mutationFn: (variables: RestoreCreateVariables) => {
      const payload: CreateRestorePayload = {
        metadata: {
          name: `restore-${generateShortUID()}`,
        },
        spec: {
          instanceRef: { name: variables.instanceName },
          dataSource: buildRestoreDataSource(variables),
        },
      };
      return createInstanceRestoreFn(clusterName, namespace, payload);
    },
    ...options,
  });

export const useInstanceRestores = (
  clusterName: string,
  namespace: string,
  instanceName: string,
  options?: PerconaQueryOptions<GetRestorePayload, unknown, Restore[]>
) => {
  const { canRead } = useRBACPermissions(
    'database-cluster-restores',
    `${namespace}/${instanceName}`
  );

  return useQuery<GetRestorePayload, unknown, Restore[]>({
    queryKey: getRestoresListQueryKey(clusterName, namespace, instanceName),
    queryFn: () => getInstanceRestoresFn(clusterName, namespace, instanceName),
    refetchInterval: 5 * 1000,
    // Spread caller options first so the hook's own RBAC gate and view-model
    // transform below always win — they are not meant to be overridable.
    ...options,
    enabled: (options?.enabled ?? true) && canRead,
    select: canRead ? (data) => (data.items ?? []).map(mapRestore) : () => [],
  });
};

export const useDeleteRestore = (
  clusterName: string,
  namespace: string,
  options?: UseMutationOptions<unknown, unknown, string, unknown>
) =>
  useMutation({
    mutationFn: (restoreName: string) =>
      deleteRestoreFn(clusterName, namespace, restoreName),
    ...options,
  });
