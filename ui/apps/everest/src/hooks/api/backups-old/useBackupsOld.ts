import { useQuery } from '@tanstack/react-query';
import {
  getBackupsFn,
  getPitrFn,
} from 'api/backups-old';
import {
  Backup,
  BackupStatus,
  DatabaseClusterPitr,
  DatabaseClusterPitrPayload,
  GetBackupsPayload,
} from 'shared-types/backupsOld.types';
import { mapBackupState } from 'utils/backups';
import { PerconaQueryOptions } from 'shared-types/query.types';
import { useRBACPermissions } from 'hooks/rbac';

export const BACKUPS_QUERY_KEY = 'backups-old';

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
              dbClusterName,
              backupStorageName,
            })
          )
      : () => [],
    ...options,
    enabled: (options?.enabled ?? true) && canRead,
  });
};

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
