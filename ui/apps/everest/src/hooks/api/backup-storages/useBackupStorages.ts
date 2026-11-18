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
import {
  createBackupStorageFn,
  deleteBackupStorageFn,
  getBackupStorageFn,
  getBackupStoragesFn,
  updateBackupStorageFn,
} from 'api/backupStorage';
import {
  CredentialsSecretWrite,
  getCredentialsSecretRequests,
  saveWithCredentialsSecret,
  SECRETS_QUERY_KEY,
} from 'hooks/api/secrets';
import { isAxiosError } from 'axios';
import {
  BackupStorageCRD,
  BackupStorageListFilter,
  BackupStoragePayload,
} from 'shared-types/backupStorages.types';
import { PerconaQueryOptions } from 'shared-types/query.types';
import { isSameJsonDocument } from 'utils/json-document';
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

export const useBackupStorage = (
  namespace: string,
  name: string,
  options?: PerconaQueryOptions<BackupStorageCRD>
) => {
  const cluster = useClusterName();
  return useQuery<BackupStorageCRD>({
    // Under the namespace's key, so a write to the namespace refreshes it.
    queryKey: [BACKUP_STORAGES_QUERY_KEY, cluster, namespace, name],
    queryFn: () => getBackupStorageFn(cluster, namespace, name),
    ...options,
  });
};

export interface SaveBackupStorageRequest {
  namespace: string;
  storage: BackupStoragePayload;
  // The spec the edit started from; a new storage is added without it.
  savedSpec?: BackupStoragePayload['spec'];
  credentialsSecret?: CredentialsSecretWrite;
}

const STATUS_ONLY_CONFLICT_RETRIES = 3;

// Core writes a storage's status as it checks it, e.g. once its Secret is
// back, and that changes the storage's version too. Only a spec changed since
// the edit started is a conflict the user has to look at: over any other
// change the update goes again, at the latest version.
const updateOverStatusChanges = async (
  cluster: string,
  namespace: string,
  storage: BackupStoragePayload,
  savedSpec: BackupStoragePayload['spec'],
  retriesLeft = STATUS_ONLY_CONFLICT_RETRIES
): Promise<BackupStorageCRD> => {
  try {
    return await updateBackupStorageFn(cluster, namespace, storage);
  } catch (error) {
    const isConflict = isAxiosError(error) && error.response?.status === 409;
    if (!isConflict || retriesLeft === 0) {
      throw error;
    }
    const latest = await getBackupStorageFn(
      cluster,
      namespace,
      storage.metadata?.name ?? ''
    );
    if (!isSameJsonDocument(latest.spec, savedSpec)) {
      throw error;
    }
    return updateOverStatusChanges(
      cluster,
      namespace,
      { ...storage, metadata: latest.metadata },
      savedSpec,
      retriesLeft - 1
    );
  }
};

export const useSaveBackupStorage = () => {
  const cluster = useClusterName();
  const queryClient = useQueryClient();
  return useMutation<BackupStorageCRD, unknown, SaveBackupStorageRequest>({
    mutationFn: ({ namespace, storage, savedSpec, credentialsSecret }) =>
      saveWithCredentialsSecret(
        {
          credentialsSecret,
          save: () =>
            savedSpec
              ? updateOverStatusChanges(cluster, namespace, storage, savedSpec)
              : createBackupStorageFn(cluster, namespace, storage),
        },
        getCredentialsSecretRequests(cluster, namespace)
      ),
    // Don't keep the submitted credentials in the mutation cache.
    gcTime: 0,
    // Settled, not only successful: a refused save may still have written
    // the Secret.
    onSettled: (_data, _error, { namespace }) =>
      Promise.all([
        queryClient.invalidateQueries({
          queryKey: getBackupStoragesQueryKey(cluster, namespace),
        }),
        queryClient.invalidateQueries({
          queryKey: [SECRETS_QUERY_KEY, cluster, namespace],
        }),
      ]),
  });
};
