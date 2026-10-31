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

import { useQuery } from '@tanstack/react-query';
import {
  getBackupStorageClassFn,
  getBackupStorageClassesFn,
} from 'api/backupStorageClasses';
import { useRBACPermissions } from 'hooks/rbac';
import { BackupStorageClass } from 'shared-types/backupStorages.types';
import { PerconaQueryOptions } from 'shared-types/query.types';
import { useClusterName } from '../useClusterName';

export const BACKUP_STORAGE_CLASSES_QUERY_KEY = 'backup-storage-classes';

export const useBackupStorageClasses = (
  options?: PerconaQueryOptions<BackupStorageClass[]>
) => {
  const cluster = useClusterName();
  const { canRead } = useRBACPermissions('backup-storage-classes');

  return useQuery<BackupStorageClass[]>({
    queryKey: [BACKUP_STORAGE_CLASSES_QUERY_KEY, cluster],
    queryFn: () => getBackupStorageClassesFn(cluster),
    ...options,
    enabled: (options?.enabled ?? true) && canRead,
  });
};

export const useBackupStorageClass = (
  name: string,
  options?: PerconaQueryOptions<BackupStorageClass>
) => {
  const cluster = useClusterName();
  const { canRead } = useRBACPermissions('backup-storage-classes');

  return useQuery<BackupStorageClass>({
    queryKey: [BACKUP_STORAGE_CLASSES_QUERY_KEY, cluster, name],
    queryFn: () => getBackupStorageClassFn(cluster, name),
    ...options,
    enabled: (options?.enabled ?? true) && canRead && !!name,
  });
};
