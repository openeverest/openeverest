// Copyright (C) 2023 Percona LLC
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
  useQueries,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query';
import { deleteBackupStorageFn, getBackupStoragesFn } from 'api/backupStorage';
import {
  BackupStorageCRD,
  BackupStorageListFilter,
} from 'shared-types/backupStorages.types';
import { PerconaQueryOptions } from 'shared-types/query.types';
import { useNamespaces } from '../namespaces';
import { useClusterName } from '../useClusterName';

export const BACKUP_STORAGES_QUERY_KEY = 'backupStorages';

// A filtered list is cached apart from the namespace's full one; both start
// with the namespace's key, which a write to the namespace invalidates.
export const getBackupStoragesQueryKey = (
  cluster: string,
  namespace: string,
  filter?: BackupStorageListFilter
) =>
  filter
    ? ([BACKUP_STORAGES_QUERY_KEY, cluster, namespace, filter] as const)
    : ([BACKUP_STORAGES_QUERY_KEY, cluster, namespace] as const);

export const useBackupStorages = () => {
  const cluster = useClusterName();
  const { data: namespaces = [] } = useNamespaces({
    refetchInterval: 5 * 1000,
  });
  return useQueries({
    queries: namespaces.map((namespace) => ({
      queryKey: getBackupStoragesQueryKey(cluster, namespace),
      queryFn: () => getBackupStoragesFn(cluster, namespace),
      refetchInterval: 5 * 1000,
    })),
  });
};

export const useBackupStoragesByNamespace = (
  namespace: string,
  filter?: BackupStorageListFilter,
  options?: PerconaQueryOptions<BackupStorageCRD[], unknown, BackupStorageCRD[]>
) => {
  const cluster = useClusterName();
  return useQuery<BackupStorageCRD[], unknown, BackupStorageCRD[]>({
    queryKey: getBackupStoragesQueryKey(cluster, namespace, filter),
    queryFn: () => getBackupStoragesFn(cluster, namespace, filter),
    ...options,
  });
};

interface DeleteBackupStorageVariables {
  namespace: string;
  name: string;
}

// Settled, not only successful: the list shows the storage as Deleting, or
// without it.
export const useDeleteBackupStorage = () => {
  const cluster = useClusterName();
  const queryClient = useQueryClient();
  return useMutation<void, unknown, DeleteBackupStorageVariables>({
    mutationFn: ({ namespace, name }) =>
      deleteBackupStorageFn(cluster, namespace, name),
    onSettled: (_data, _error, { namespace }) =>
      queryClient.invalidateQueries({
        queryKey: getBackupStoragesQueryKey(cluster, namespace),
      }),
  });
};
