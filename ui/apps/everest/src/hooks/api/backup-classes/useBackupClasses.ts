import { useQuery } from "@tanstack/react-query";
import { getBackupClassFn, listBackupClassesFn } from "api/backups";
import { BackupClass, GetBackupClassPayload, ListBackupClassesPayload } from "shared-types/backups.types";
import { PerconaQueryOptions } from "shared-types/query.types";

export const BACKUP_CLASSES_QUERY_KEY = 'backup-classes';

export const getBackupClassesQueryKey = (clusterName: string) =>
  [BACKUP_CLASSES_QUERY_KEY, clusterName] as const;

export const getBackupClassQueryKey = (
  clusterName: string,
  backupClassName: string
) => [BACKUP_CLASSES_QUERY_KEY, clusterName, backupClassName] as const;


export const useBackupClassesList = (
  clusterName: string,
  options?: PerconaQueryOptions<ListBackupClassesPayload, unknown, BackupClass[]>
) => {
//   const { canRead } = useRBACPermissions('backup-classes');

  return useQuery<ListBackupClassesPayload, unknown, BackupClass[]>({
    queryKey: getBackupClassesQueryKey(clusterName),
    queryFn: () => listBackupClassesFn(clusterName),
    // select: canRead ? ({ items = [] }) => items : () => [],
    select: ({ items = [] }) => items,
    // enabled: (options?.enabled ?? true) && canRead,
    enabled: (options?.enabled ?? true),
    ...options,
  });
};

export const useGetBackupClass = (
  clusterName: string,
  backupClassName: string,
  options?: PerconaQueryOptions<
    GetBackupClassPayload,
    unknown,
    BackupClass
  >
) => {
//   const { canRead } = useRBACPermissions('backup-classes');

  return useQuery<GetBackupClassPayload, unknown, BackupClass>({
    queryKey: getBackupClassQueryKey(clusterName, backupClassName),
    queryFn: () => getBackupClassFn(clusterName, backupClassName),
    // enabled: (options?.enabled ?? true) && canRead,
    enabled: (options?.enabled ?? true),
    ...options,
  });
};

