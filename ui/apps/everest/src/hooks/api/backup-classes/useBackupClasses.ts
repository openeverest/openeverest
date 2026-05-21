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

import { useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { getBackupClassFn, getBackupClassesListFn } from 'api/backups';
import {
  BackupClass,
  GetBackupClassPayload,
  ListBackupClassesPayload,
} from 'shared-types/backups.types';
import { PerconaQueryOptions } from 'shared-types/query.types';
import { BackupClassUiSchemaSections } from 'shared-types/backups.types';
import { useRBACPermissions } from 'hooks/rbac';
import type { Section } from 'components/ui-generator/ui-generator.types';

export const BACKUP_CLASSES_QUERY_KEY = 'backup-classes';

export const getBackupClassesQueryKey = (clusterName: string) =>
  [BACKUP_CLASSES_QUERY_KEY, clusterName] as const;

export const getBackupClassQueryKey = (
  clusterName: string,
  backupClassName: string
) => [BACKUP_CLASSES_QUERY_KEY, clusterName, backupClassName] as const;

export const useBackupClassesList = (
  clusterName: string,
  options?: PerconaQueryOptions<
    ListBackupClassesPayload,
    unknown,
    BackupClass[]
  >
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
  options?: PerconaQueryOptions<GetBackupClassPayload, unknown, BackupClass>
) => {
  const { canRead } = useRBACPermissions('backup-classes');

  return useQuery<GetBackupClassPayload, unknown, BackupClass>({
    queryKey: getBackupClassQueryKey(clusterName, backupClassName),
    queryFn: () => getBackupClassFn(clusterName, backupClassName),
    enabled: (options?.enabled ?? true) && canRead,
    ...options,
  });
};

export const useBackupClassUiSchema = (
  backupClass: BackupClass | undefined
): {
  sections: Record<string, Section> | undefined;
  uiSchema: BackupClassUiSchemaSections | undefined;
} => {
  return useMemo(() => {
    if (!backupClass) return { sections: undefined, uiSchema: undefined };

    // The generated CRD types define uiSchema as Record<string, never> (opaque).
    // We cast to our typed alias to access the known runtime shape.
    const uiSchema = backupClass.spec?.uiSchema as unknown as
      | BackupClassUiSchemaSections
      | undefined;

    if (!uiSchema?.backup) return { sections: undefined, uiSchema };

    const sections: Record<string, Section> = {
      config: uiSchema.backup,
    };

    return { sections, uiSchema };
  }, [backupClass]);
};
