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
  BackupStorage,
  BackupStorageListCRD,
  GetBackupStoragesPayload,
} from 'shared-types/backupStorages.types';
import { api } from './api';
import {
  crdToFlat,
  flatToCrdCreate,
  flatToCrdEdit,
} from './backupStorage.mappers';

export const getBackupStoragesFn = async (
  cluster: string,
  namespace: string
): Promise<GetBackupStoragesPayload> => {
  const response = await api.get<BackupStorageListCRD>(
    `clusters/${cluster}/namespaces/${namespace}/backup-storages`
  );
  return response.data?.items?.map(crdToFlat) ?? [];
};

export const createBackupStorageFn = async (
  cluster: string,
  formData: BackupStorage
) => {
  const { namespace } = formData;
  const response = await api.post(
    `clusters/${cluster}/namespaces/${namespace}/backup-storages`,
    flatToCrdCreate(formData)
  );
  return response.data;
};

export const editBackupStorageFn = async (
  cluster: string,
  formData: BackupStorage
) => {
  const { name, namespace } = formData;
  const response = await api.patch(
    `clusters/${cluster}/namespaces/${namespace}/backup-storages/${name}`,
    flatToCrdEdit(formData)
  );
  return response.data;
};

export const deleteBackupStorageFn = async (
  cluster: string,
  backupStorageId: string,
  namespace: string
) => {
  const response = await api.delete(
    `clusters/${cluster}/namespaces/${namespace}/backup-storages/${backupStorageId}`
  );
  return response.data;
};
