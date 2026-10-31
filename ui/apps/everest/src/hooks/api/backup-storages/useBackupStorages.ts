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
  UseMutationOptions,
  useQueries,
  useQuery,
} from '@tanstack/react-query';
import {
  createBackupStorageFn,
  deleteBackupStorageFn,
  editBackupStorageFn,
  getBackupStoragesFn,
} from 'api/backupStorage';
import {
  BackupStorageCRD,
  BackupStorageFormValues,
  BackupStorageListFilter,
} from 'shared-types/backupStorages.types';
import { PerconaQueryOptions } from 'shared-types/query.types';
import { useNamespaces } from '../namespaces';
import { useClusterName } from '../useClusterName';

export const BACKUP_STORAGES_QUERY_KEY = 'backupStorages';

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
  const queries = namespaces.map((namespace) => {
    return {
      queryKey: getBackupStoragesQueryKey(cluster, namespace),
      queryFn: () => getBackupStoragesFn(cluster, namespace),
      refetchInterval: 5 * 1000,
    };
  });

  const queryResults = useQueries({
    queries,
  });

  return queryResults;
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

export const useCreateBackupStorage = (
  options?: UseMutationOptions<
    unknown,
    unknown,
    BackupStorageFormValues,
    unknown
  >
) => {
  const cluster = useClusterName();
  return useMutation({
    mutationFn: (formData: BackupStorageFormValues) =>
      createBackupStorageFn(cluster, formData),
    // Don't keep the submitted credentials in the mutation cache.
    gcTime: 0,
    ...options,
  });
};

export type EditBackupStorageArgs = {
  formData: BackupStorageFormValues;
  previous: BackupStorageCRD;
};

export const useEditBackupStorage = (
  options?: UseMutationOptions<unknown, unknown, EditBackupStorageArgs, unknown>
) => {
  const cluster = useClusterName();
  return useMutation({
    mutationFn: ({ formData, previous }: EditBackupStorageArgs) =>
      editBackupStorageFn(cluster, formData, previous),
    gcTime: 0,
    ...options,
  });
};

type DeleteBackupStorageArgType = {
  backupStorageId: string;
  namespace: string;
};

export const useDeleteBackupStorage = (
  options?: UseMutationOptions<
    unknown,
    unknown,
    DeleteBackupStorageArgType,
    unknown
  >
) => {
  const cluster = useClusterName();
  return useMutation({
    mutationFn: ({ backupStorageId, namespace }: DeleteBackupStorageArgType) =>
      deleteBackupStorageFn(cluster, backupStorageId, namespace),
    ...options,
  });
};
