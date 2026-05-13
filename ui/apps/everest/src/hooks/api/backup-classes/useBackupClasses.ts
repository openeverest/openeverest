import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { getBackupClassFn, getBackupClassesListFn } from "api/backups";
import { BackupClass, GetBackupClassPayload, ListBackupClassesPayload } from "shared-types/backups.types";
import { PerconaQueryOptions } from "shared-types/query.types";
import { useRBACPermissions } from 'hooks/rbac';
import type { Section } from "components/ui-generator/ui-generator.types";
import type { BackupClassUiSchemaSections } from "./backupClassFixtures";
import { mockUiSchemaByProvider } from "./backupClassFixtures";

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
  const { canRead } = useRBACPermissions('backup-classes');

  return useQuery<ListBackupClassesPayload, unknown, BackupClass[]>({
    queryKey: getBackupClassesQueryKey(clusterName),
    queryFn: () => getBackupClassesListFn(clusterName),
    select: canRead ? ({ items = [] }) => items : () => [],
    enabled: (options?.enabled ?? true) && canRead,
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
  const { canRead } = useRBACPermissions('backup-classes');

  return useQuery<GetBackupClassPayload, unknown, BackupClass>({
    queryKey: getBackupClassQueryKey(clusterName, backupClassName),
    queryFn: () => getBackupClassFn(clusterName, backupClassName),
    enabled: (options?.enabled ?? true) && canRead,
    ...options,
  });
};

/**
 * Returns parsed uiSchema sections for a given BackupClass.
 *
 * TODO: Once PR #2226 is merged and `backupClass.spec.uiSchema` is available
 * from the API, replace the mock lookup with:
 *   `backupClass.spec.uiSchema as unknown as BackupClassUiSchemaSections`
 */
export const useBackupClassUiSchema = (
  clusterName: string,
  backupClassName: string | undefined
): {
  sections: Record<string, Section> | undefined;
  uiSchema: BackupClassUiSchemaSections | undefined;
  isLoading: boolean;
} => {
  const { data: backupClass, isLoading } = useGetBackupClass(
    clusterName,
    backupClassName ?? '',
    { enabled: !!backupClassName }
  );

  const result = useMemo(() => {
    if (!backupClass) return { sections: undefined, uiSchema: undefined };

    // TODO: Switch to real data when PR #2226 lands:
    // const uiSchema = backupClass.spec?.uiSchema as unknown as BackupClassUiSchemaSections;
    const providerName = backupClass.spec?.supportedProviders?.[0];
    const uiSchema = providerName
      ? mockUiSchemaByProvider[providerName]
      : undefined;

    if (!uiSchema?.backup) return { sections: undefined, uiSchema };

    // Build sections map keyed as `config` so UIGenerator field names
    // become `config.type`, `config.compressionType`, etc.
    const sections: Record<string, Section> = {
      config: uiSchema.backup,
    };

    return { sections, uiSchema };
  }, [backupClass]);

  return { ...result, isLoading };
};

