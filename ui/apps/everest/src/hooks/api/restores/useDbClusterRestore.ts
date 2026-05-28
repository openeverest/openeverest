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
import { PerconaQueryOptions } from 'shared-types/query.types';
import {
  CreateRestorePayload,
  GetRestorePayload,
  Restore,
} from 'shared-types/restores.types';

export const RESTORES_QUERY_KEY = 'restores';

export const getRestoresListQueryKey = (
  clusterName: string,
  namespace: string,
  instanceName: string
) => [RESTORES_QUERY_KEY, clusterName, namespace, instanceName] as const;

export const useCreateRestoreFromBackup = (
  clusterName: string,
  namespace: string,
  options?: UseMutationOptions<
    unknown,
    unknown,
    { instanceName: string; backupName: string },
    unknown
  >
) =>
  useMutation({
    mutationFn: ({
      instanceName,
      backupName,
    }: {
      instanceName: string;
      backupName: string;
    }) => {
      const payload: CreateRestorePayload = {
        metadata: {
          name: `restore-${generateShortUID()}`,
        },
        spec: {
          instanceName,
          dataSource: {
            type: 'Backup',
            backup: {
              backupName,
            },
          },
        },
      };
      return createInstanceRestoreFn(clusterName, namespace, payload);
    },
    ...options,
  });

// TODO: Re-enable when PITR restore flow is implemented.
// export const useCreateRestoreFromPointInTime = (
//   clusterName: string,
//   namespace: string,
//   options?: UseMutationOptions<
//     unknown,
//     unknown,
//     { instanceName: string; backupName: string; pointInTimeDate: string },
//     unknown
//   >
// ) =>
//   useMutation({
//     mutationFn: ({
//       instanceName,
//       backupName,
//       pointInTimeDate,
//     }: {
//       instanceName: string;
//       backupName: string;
//       pointInTimeDate: string;
//     }) => {
//       const payload: CreateRestorePayload = {
//         metadata: { name: `restore-${generateShortUID()}` },
//         spec: {
//           instanceName,
//           dataSource: {
//             backupName,
//             pitr: { type: 'date', date: pointInTimeDate },
//           },
//         },
//       };
//       return createInstanceRestoreFn(clusterName, namespace, payload);
//     },
//     ...options,
//   });

export const useInstanceRestores = (
  clusterName: string,
  namespace: string,
  instanceName: string,
  options?: PerconaQueryOptions<GetRestorePayload, unknown, Restore[]>
) => {
  // TODO: Re-enable RBAC check when RBAC is implemented for v2 restores
  // const { canRead } = useRBACPermissions('restores', `${namespace}/${instanceName}`);

  return useQuery<GetRestorePayload, unknown, Restore[]>({
    queryKey: getRestoresListQueryKey(clusterName, namespace, instanceName),
    queryFn: () => getInstanceRestoresFn(clusterName, namespace, instanceName),
    refetchInterval: 5 * 1000,
    enabled: options?.enabled ?? true,
    // TODO: Re-enable RBAC-gated select when RBAC is implemented.
    // select: canRead ? (data) => data.items.map(...) : () => [],
    select: (data) =>
      data.items.map((item) => ({
        name: item.metadata.name,
        startTime: item.status?.startedAt || item.metadata.creationTimestamp,
        endTime: item.status?.completedAt,
        state: item.status?.state || 'unknown',
        type: item.spec.dataSource.backup?.pitr ? 'pitr' : 'full',
        backupSource: item.spec.dataSource.backup?.backupName || '',
      })),
    ...options,
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
