// Legacy v1alpha1 backup hooks — used by old pages (BackupStoragesInput,
// db-actions-modals, LastBackup, database-form).
// These call the v1alpha1 REST API and will be removed once all consumers
// are migrated to the v2 Instance API.

import { useQuery } from '@tanstack/react-query';
import {
  legacyGetBackupsFn,
  getPitrFn,
} from 'api/backups';
import {
  DatabaseClusterPitr,
  DatabaseClusterPitrPayload,
  LegacyBackup,
  LegacyBackupStatus,
  LegacyGetBackupsPayload,
} from 'shared-types/backups.types';
import { PerconaQueryOptions } from 'shared-types/query.types';
import { useRBACPermissions } from 'hooks/rbac';
import { mapBackupState } from 'utils/backups';

const LEGACY_BACKUPS_QUERY_KEY = 'backups-old';

export const useDbBackups = (
  dbClusterName: string,
  namespace: string,
  options?: PerconaQueryOptions<LegacyGetBackupsPayload, unknown, LegacyBackup[]>
) => {
  const { canRead } = useRBACPermissions(
    'database-cluster-backups',
    `${namespace}/${dbClusterName}`
  );
  return useQuery<LegacyGetBackupsPayload, unknown, LegacyBackup[]>({
    queryKey: [LEGACY_BACKUPS_QUERY_KEY, namespace, dbClusterName],
    queryFn: () => legacyGetBackupsFn(dbClusterName, namespace),
    select: canRead
      ? ({ items = [] }) =>
          items.map(
            ({ metadata: { name }, status, spec: { backupStorageName } }) => ({
              name,
              created: status?.created,
              completed: status?.completed,
              state: status
                ? mapBackupState(status?.state)
                : LegacyBackupStatus.UNKNOWN,
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
