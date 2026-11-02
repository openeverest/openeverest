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

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { createBackupImportFn, getBackupImportsFn } from 'api/backup-imports';
import { BackupImport, BackupImportList } from 'shared-types/backups.types';
import { PerconaQueryOptions } from 'shared-types/query.types';

export const BACKUP_IMPORTS_QUERY_KEY = 'backup-imports';

export const getBackupImportsQueryKey = (
  clusterName: string,
  namespace: string
) => [BACKUP_IMPORTS_QUERY_KEY, clusterName, namespace] as const;

export const useBackupImports = (
  clusterName: string,
  namespace: string,
  options?: PerconaQueryOptions<BackupImportList, unknown, BackupImport[]>
) =>
  useQuery<BackupImportList, unknown, BackupImport[]>({
    queryKey: getBackupImportsQueryKey(clusterName, namespace),
    queryFn: () => getBackupImportsFn(clusterName, namespace),
    select: ({ items = [] }) => items,
    ...options,
  });

export const useCreateBackupImport = (
  clusterName: string,
  namespace: string
) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (spec: BackupImport['spec']) =>
      createBackupImportFn(clusterName, namespace, {
        // The spec is immutable, so every scan is a new object.
        metadata: { generateName: 'import-' },
        spec,
      }),
    // Stays pending until the list shows the new import.
    onSuccess: () =>
      queryClient.invalidateQueries({
        queryKey: getBackupImportsQueryKey(clusterName, namespace),
      }),
  });
};
