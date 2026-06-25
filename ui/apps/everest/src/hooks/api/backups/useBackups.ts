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
  createBackupOnDemandFn,
  deleteBackupFn,
  getBackupFn,
  getBackupsListFn,
} from 'api/backups';
import {
  Backup,
  BackupList,
  BackupStatus,
  CreateBackupPayload,
  CreateBackupResponse,
  DeleteBackupPayload,
  GetBackupPayload,
} from 'shared-types/backups.types';
import { PerconaQueryOptions } from 'shared-types/query.types';
import { useRBACPermissions } from 'hooks/rbac';

export const BACKUPS_QUERY_KEY = 'backups';

export const getBackupQueryKey = (
  clusterName: string,
  namespace: string,
  backupName: string
) => [BACKUPS_QUERY_KEY, clusterName, namespace, backupName] as const;

export const getBackupListQueryKey = (
  clusterName: string,
  namespace: string,
  instanceName: string
) => [BACKUPS_QUERY_KEY, clusterName, namespace, instanceName, 'list'] as const;

type DeleteBackupArgType = {
  backupName: string;
  deletionPolicy?: 'Delete' | 'Retain';
};

export const useBackupsList = (
  clusterName: string,
  namespace: string,
  instanceName: string,
  options?: PerconaQueryOptions<BackupList, unknown, Backup[]>
) => {
  const { canRead } = useRBACPermissions(
    'backups',
    `${namespace}/${instanceName}`
  );

  return useQuery<BackupList, unknown, Backup[]>({
    queryKey: getBackupListQueryKey(clusterName, namespace, instanceName),
    queryFn: () => getBackupsListFn(clusterName, namespace, instanceName),
    select: canRead
      ? ({ items = [] }) =>
          items
            .filter((backup) => backup.spec.instanceName === instanceName)
            .map((backup) => ({
              ...backup,
              status: {
                ...backup.status,
                state: backup.status?.state ?? BackupStatus.UNKNOWN,
              },
            }))
      : () => [],
    enabled: (options?.enabled ?? true) && canRead,
    ...options,
  });
};

export const useCreateBackupOnDemand = (
  clusterName: string,
  namespace: string,
  options?: UseMutationOptions<
    CreateBackupResponse,
    unknown,
    CreateBackupPayload,
    unknown
  >
) => {
  // TODO RBAC: resource name 'backups' needs additional testing with real v2 RBAC policies
  const { canCreate } = useRBACPermissions('backups', `${namespace}/*`);

  return useMutation({
    mutationFn: (payload: CreateBackupPayload) => {
      if (!canCreate) {
        throw new Error('Not enough permissions to create backups');
      }
      return createBackupOnDemandFn(clusterName, namespace, payload);
    },
    ...options,
    meta: {
      ...(options?.meta ?? {}),
    },
  });
};

export const useDeleteBackup = (
  clusterName: string,
  namespace: string,
  instanceName: string,
  options?: UseMutationOptions<
    DeleteBackupPayload,
    unknown,
    DeleteBackupArgType,
    unknown
  >
) => {
  const { canDelete } = useRBACPermissions(
    'backups',
    `${namespace}/${instanceName}`
  );

  return useMutation({
    mutationFn: ({ backupName, deletionPolicy }: DeleteBackupArgType) => {
      if (!canDelete) {
        throw new Error('Not enough permissions to delete backups');
      }
      return deleteBackupFn(clusterName, namespace, backupName, deletionPolicy);
    },
    ...options,
  });
};

export const useGetBackup = (
  clusterName: string,
  namespace: string,
  backupName: string,
  options?: PerconaQueryOptions<GetBackupPayload, unknown, Backup>
) => {
  const { canRead } = useRBACPermissions('backups', `${namespace}/*`);

  return useQuery<GetBackupPayload, unknown, Backup>({
    queryKey: getBackupQueryKey(clusterName, namespace, backupName),
    queryFn: () => getBackupFn(clusterName, namespace, backupName),
    enabled: (options?.enabled ?? true) && canRead,
    ...options,
  });
};

// TODO: PITR is not part of this task — uncomment when working on PITR feature.
// The v2 API endpoint for PITR may change; update getPitrFn in api/backups.ts accordingly.
//
// import { DatabaseClusterPitrPayload, DatabaseClusterPitr } from 'shared-types/backups.types';
// import { getPitrFn } from 'api/backups';
//
// export const useDbClusterPitr = (
//   dbClusterName: string,
//   namespace: string,
//   options?: PerconaQueryOptions<
//     DatabaseClusterPitrPayload,
//     unknown,
//     DatabaseClusterPitr | undefined
//   >
// ) => {
//   const { canRead } = useRBACPermissions(
//     'database-clusters',
//     `${namespace}/${dbClusterName}`
//   );
//
//   return useQuery<
//     DatabaseClusterPitrPayload,
//     unknown,
//     DatabaseClusterPitr | undefined
//   >({
//     queryKey: [dbClusterName, 'pitr'],
//     queryFn: () => getPitrFn(dbClusterName, namespace),
//     select: (pitrData) => {
//       const { earliestDate, latestDate, latestBackupName, gaps } = pitrData;
//       if (
//         !Object.keys(pitrData).length ||
//         !earliestDate ||
//         !latestDate ||
//         !latestBackupName
//       ) {
//         return undefined;
//       }
//
//       return {
//         earliestDate: new Date(earliestDate),
//         latestDate: new Date(latestDate),
//         latestBackupName,
//         gaps,
//       };
//     },
//     ...options,
//     enabled: (options?.enabled ?? true) && canRead,
//   });
// };
