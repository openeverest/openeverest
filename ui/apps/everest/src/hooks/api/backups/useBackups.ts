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
  createBackupOnDemand,
  deleteBackupFn,
  getBackupsFn,
  getPitrFn,
} from 'api/backups';
import {
  Backup,
  BackupStatus,
  DatabaseClusterPitr,
  DatabaseClusterPitrPayload,
  GetBackupsPayload,
  SingleBackupPayload,
} from 'shared-types/backups.types';
import { mapBackupState } from 'utils/backups';
import { BackupFormData } from 'pages/db-cluster-details/backups/on-demand-backup-modal/on-demand-backup-modal.types';
import { PerconaQueryOptions } from 'shared-types/query.types';
import { useRBACPermissions } from 'hooks/rbac';

export const BACKUPS_QUERY_KEY = 'backups';

type DeleteBackupArgType = {
  backupName: string;
  cleanupBackupStorage: boolean;
};

export const useDbBackups = (
  dbClusterName: string,
  namespace: string,
  options?: PerconaQueryOptions<GetBackupsPayload, unknown, Backup[]>
) => {
  const { canRead } = useRBACPermissions(
    'database-cluster-backups',
    `${namespace}/${dbClusterName}`
  );
  return useQuery<GetBackupsPayload, unknown, Backup[]>({
    queryKey: [BACKUPS_QUERY_KEY, namespace, dbClusterName],
    queryFn: () => getBackupsFn(dbClusterName, namespace),
    select: canRead
      ? ({ items = [] }) =>
          items.map(
            ({ metadata: { name }, status, spec: { backupStorageName } }) => ({
              name,
              created: status?.created,
              completed: status?.completed,
              state: status
                ? mapBackupState(status?.state)
                : BackupStatus.UNKNOWN,
              size: status?.size,
              dbClusterName,
              backupStorageName,
            })
          )
      : () => [],
    ...options,
    enabled: (options?.enabled ?? true) && canRead,
  });
};

export const useCreateBackupOnDemand = (
  dbClusterName: string,
  namespace: string,
  options?: UseMutationOptions<
    SingleBackupPayload,
    unknown,
    BackupFormData,
    unknown
  >
) =>
  useMutation({
    mutationFn: (formData: BackupFormData) =>
      createBackupOnDemand(
        {
          apiVersion: 'everest.percona.com/v1alpha1',
          kind: 'DatabaseClusterBackup',
          metadata: {
            name: formData.name,
          },
          spec: {
            dbClusterName,
            backupStorageName:
              typeof formData.storageLocation === 'string'
                ? formData.storageLocation
                : formData.storageLocation!.name,
          },
        },
        namespace
      ),
    ...options,
  });

export const useDeleteBackup = (
  namespace: string,
  options?: UseMutationOptions<unknown, unknown, DeleteBackupArgType, unknown>
) =>
  useMutation({
    mutationFn: ({ backupName, cleanupBackupStorage }: DeleteBackupArgType) =>
      deleteBackupFn(backupName, namespace, cleanupBackupStorage),
    ...options,
  });

export const useDbClusterPitr = (
  dbClusterName: string,
  namespace: string,
  options?: PerconaQueryOptions<
    DatabaseClusterPitrPayload,
    unknown,
    DatabaseClusterPitr | undefined
  >
) => {
  const { canRead } = useRBACPermissions(
    'database-clusters',
    `${namespace}/${dbClusterName}`
  );

  return useQuery<
    DatabaseClusterPitrPayload,
    unknown,
    DatabaseClusterPitr | undefined
  >({
    queryKey: [dbClusterName, 'pitr'],
    queryFn: () => getPitrFn(dbClusterName, namespace),
    select: (pitrData) => {
      const { earliestDate, latestDate, latestBackupName, gaps } = pitrData;
      if (
        !Object.keys(pitrData).length ||
        !earliestDate ||
        !latestDate ||
        !latestBackupName
      ) {
        return undefined;
      }

      return {
        earliestDate: new Date(earliestDate),
        latestDate: new Date(latestDate),
        latestBackupName,
        gaps,
      };
    },
    ...options,
    enabled: (options?.enabled ?? true) && canRead,
  });
};
