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
  BackupStorageCRD,
  BackupStorageListCRD,
  BackupStorageListFilter,
  BackupStoragePayload,
} from 'shared-types/backupStorages.types';
import { api } from './api';

const storagesPath = (cluster: string, namespace: string) =>
  `clusters/${cluster}/namespaces/${namespace}/backup-storages`;

export const getBackupStoragesFn = async (
  cluster: string,
  namespace: string,
  filter?: BackupStorageListFilter
): Promise<BackupStorageCRD[]> => {
  const response = await api.get<BackupStorageListCRD>(
    storagesPath(cluster, namespace),
    { params: filter }
  );
  return response.data?.items ?? [];
};

export const getBackupStorageFn = async (
  cluster: string,
  namespace: string,
  name: string
) => {
  const response = await api.get<BackupStorageCRD>(
    `${storagesPath(cluster, namespace)}/${name}`
  );
  return response.data;
};

export const createBackupStorageFn = async (
  cluster: string,
  namespace: string,
  storage: BackupStoragePayload
) => {
  const response = await api.post<BackupStorageCRD>(
    storagesPath(cluster, namespace),
    storage
  );
  return response.data;
};

export const updateBackupStorageFn = async (
  cluster: string,
  namespace: string,
  storage: BackupStoragePayload
) => {
  const response = await api.put<BackupStorageCRD>(
    `${storagesPath(cluster, namespace)}/${storage.metadata?.name ?? ''}`,
    storage,
    // A storage changed meanwhile is reported on the form.
    { disableNotifications: (e) => e.status === 409 }
  );
  return response.data;
};

// A storage something still uses stays, Deleting, until nothing does.
export const deleteBackupStorageFn = async (
  cluster: string,
  namespace: string,
  name: string
) => {
  await api.delete(`${storagesPath(cluster, namespace)}/${name}`);
};
