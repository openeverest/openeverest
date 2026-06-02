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
  BackupStorageFormValues,
  BackupStorageCRD,
  BackupStorageListCRD,
} from 'shared-types/backupStorages.types';
import { api } from './api';

const formValuesToCrdCreate = (formValues: BackupStorageFormValues) => ({
  metadata: {
    name: formValues.name,
    namespace: formValues.namespace,
  },
  spec: {
    type: formValues.type,
    s3: {
      bucket: formValues.bucketName,
      endpointURL: formValues.url,
      region: formValues.region,
      credentialsSecretName: `backup-storage-${formValues.name}-credentials`,
      accessKeyId: formValues.accessKey,
      secretAccessKey: formValues.secretKey,
      verifyTLS: formValues.verifyTLS,
      forcePathStyle: formValues.forcePathStyle,
    },
  },
});

const formValuesToCrdEdit = (formValues: BackupStorageFormValues) => ({
  spec: {
    type: formValues.type,
    s3: {
      bucket: formValues.bucketName,
      endpointURL: formValues.url,
      region: formValues.region,
      credentialsSecretName: `backup-storage-${formValues.name}-credentials`,
      ...(formValues.accessKey && { accessKeyId: formValues.accessKey }),
      ...(formValues.secretKey && { secretAccessKey: formValues.secretKey }),
      verifyTLS: formValues.verifyTLS,
      forcePathStyle: formValues.forcePathStyle,
    },
  },
});

export const getBackupStoragesFn = async (
  cluster: string,
  namespace: string
): Promise<BackupStorageCRD[]> => {
  const response = await api.get<BackupStorageListCRD>(
    `clusters/${cluster}/namespaces/${namespace}/backup-storages`
  );
  return response.data?.items ?? [];
};

export const createBackupStorageFn = async (
  cluster: string,
  formData: BackupStorageFormValues
) => {
  const { namespace } = formData;
  const response = await api.post(
    `clusters/${cluster}/namespaces/${namespace}/backup-storages`,
    formValuesToCrdCreate(formData)
  );
  return response.data;
};

export const editBackupStorageFn = async (
  cluster: string,
  formData: BackupStorageFormValues
) => {
  const { name, namespace } = formData;
  const response = await api.patch(
    `clusters/${cluster}/namespaces/${namespace}/backup-storages/${name}`,
    formValuesToCrdEdit(formData)
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
