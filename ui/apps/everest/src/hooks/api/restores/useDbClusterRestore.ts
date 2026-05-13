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
import { deleteRestore, getInstanceRestores } from 'api/restores';
import { PerconaQueryOptions } from 'shared-types/query.types';
import { Restore, RestoreList } from 'shared-types/restores.types';
// import { useRBACPermissions } from 'hooks/rbac';

export const RESTORES_QUERY_KEY = 'restores';

export const getRestoreListQueryKey = (
  clusterName: string,
  namespace: string,
  instanceName: string
) => [RESTORES_QUERY_KEY, clusterName, namespace, instanceName] as const;

// TODO: Restore when v2 restore-from-backup API is implemented
// The v1 version used createDbClusterRestore with v1alpha1 DatabaseClusterRestore CRD.
// v2 will use the new Restore CRD — see Backup/Restore architecture doc.
// export const useRestoreFromBackup = (
//   instanceName: string,
//   options?: UseMutationOptions<unknown, unknown, unknown, unknown>
// ) =>
//   useMutation({
//     mutationFn: ({
//       backupName,
//       namespace,
//       clusterName,
//     }: {
//       backupName: string;
//       namespace: string;
//       clusterName: string;
//     }) =>
//       createRestore(clusterName, namespace, {
//         metadata: { name: `restore-${generateShortUID()}` },
//         spec: {
//           instanceName,
//           dataSource: { backupName },
//         },
//       }),
//     ...options,
//   });

// TODO: Restore when v2 PITR restore API is implemented
// export const useRestoreFromPointInTime = (
//   instanceName: string,
//   options?: UseMutationOptions<unknown, unknown, unknown, unknown>
// ) =>
//   useMutation({
//     mutationFn: ({
//       pointInTimeDate,
//       backupName,
//       namespace,
//       clusterName,
//     }: {
//       pointInTimeDate: string;
//       backupName: string;
//       namespace: string;
//       clusterName: string;
//     }) =>
//       createRestore(clusterName, namespace, {
//         metadata: { name: `restore-${generateShortUID()}` },
//         spec: {
//           instanceName,
//           dataSource: {
//             backupName,
//             pitr: { date: pointInTimeDate },
//           },
//         },
//       }),
//     ...options,
//   });

export const useDbClusterRestores = (
  clusterName: string,
  namespace: string,
  instanceName: string,
  options?: PerconaQueryOptions<RestoreList, unknown, Restore[]>
) => {
  // TODO: Restore RBAC check when v2 RBAC resource names are finalized
  // const { canRead } = useRBACPermissions(
  //   'restores',
  //   `${namespace}/${instanceName}`
  // );
  return useQuery<RestoreList, unknown, Restore[]>({
    queryKey: getRestoreListQueryKey(clusterName, namespace, instanceName),
    queryFn: () => getInstanceRestores(clusterName, namespace, instanceName),
    select: ({ items = [] }) => items,
    refetchInterval: 5 * 1000,
    ...options,
    enabled: options?.enabled ?? true,
  });
};

export const useDeleteRestore = (
  clusterName: string,
  namespace: string,
  instanceName: string,
  options?: UseMutationOptions<unknown, unknown, string, unknown>
) =>
  useMutation({
    mutationFn: (restoreName: string) =>
      deleteRestore(clusterName, namespace, instanceName, restoreName),
    ...options,
  });
