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
} from 'shared-types/backupStorages.types';
import { PerconaQueryOptions } from 'shared-types/query.types';
import { useNamespaces } from '../namespaces';
import { useClusterName } from '../useClusterName';

export const BACKUP_STORAGES_QUERY_KEY = 'backupStorages';

export const useBackupStorages = () => {
  const cluster = useClusterName();
  const { data: namespaces = [] } = useNamespaces({
    refetchInterval: 5 * 1000,
  });
  const queries = namespaces.map((namespace) => {
    return {
      queryKey: [BACKUP_STORAGES_QUERY_KEY, cluster, namespace],
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
  options?: PerconaQueryOptions<BackupStorageCRD[], unknown, BackupStorageCRD[]>
) => {
  const cluster = useClusterName();
  return useQuery<BackupStorageCRD[], unknown, BackupStorageCRD[]>({
    queryKey: [BACKUP_STORAGES_QUERY_KEY, cluster, namespace],
    queryFn: () => getBackupStoragesFn(cluster, namespace),
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
    ...options,
  });
};

export const useEditBackupStorage = (
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
      editBackupStorageFn(cluster, formData),
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
