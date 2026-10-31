// everest
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

import { APIRequestContext, APIResponse, expect } from '@playwright/test';
import { EVEREST_CI_CLUSTER } from '@e2e/constants';

export type S3BackupStorage = {
  name: string;
  bucket: string;
  region: string;
  endpointURL: string;
  accessKeyId: string;
  secretAccessKey: string;
  verifyTLS: boolean;
  forcePathStyle: boolean;
};

const namespaceUrl = (namespace: string) =>
  `/v1/clusters/${EVEREST_CI_CLUSTER}/namespaces/${namespace}`;

export const backupStorageCredentialsSecretName = (storageName: string) =>
  `backup-storage-${storageName}-credentials`;

// Returns the first response that isn't a success, so callers can tell a
// missing namespace (404) from a storage that already exists (409).
export const createBackupStorage = async (
  request: APIRequestContext,
  namespace: string,
  storage: S3BackupStorage,
  token: string
): Promise<APIResponse> => {
  const headers = { Authorization: `Bearer ${token}` };
  const secretName = backupStorageCredentialsSecretName(storage.name);

  const secretResponse = await request.post(
    `${namespaceUrl(namespace)}/secrets`,
    {
      data: {
        metadata: { name: secretName },
        type: 'Opaque',
        stringData: {
          AWS_ACCESS_KEY_ID: storage.accessKeyId,
          AWS_SECRET_ACCESS_KEY: storage.secretAccessKey,
        },
      },
      headers,
    }
  );
  if (![201, 409].includes(secretResponse.status())) {
    return secretResponse;
  }

  return request.post(`${namespaceUrl(namespace)}/backup-storages`, {
    data: {
      metadata: { name: storage.name, namespace },
      spec: {
        backupStorageClassRef: { name: 's3' },
        location: { bucket: storage.bucket },
        parameters: {
          region: storage.region,
          endpointURL: storage.endpointURL,
          forcePathStyle: storage.forcePathStyle,
          implementation: 'other',
        },
        authentication: {
          mode: 'AccessKey',
          credentialsSecretRef: { name: secretName },
        },
        ...(!storage.verifyTLS && { tls: { insecureSkipVerify: true } }),
      },
    },
    headers,
  });
};

export const getBackupStorage = async (
  request: APIRequestContext,
  namespace: string,
  name: string,
  token: string
) => {
  const response = await request.get(
    `${namespaceUrl(namespace)}/backup-storages/${name}`,
    {
      headers: {
        Authorization: `Bearer ${token}`,
      },
    }
  );
  expect(response.status() === 200).toBeTruthy();
  return await response.json();
};

export const deleteBackupStorage = async (
  request: APIRequestContext,
  namespace: string,
  name: string,
  token: string
) => {
  const headers = {
    'Content-Type': 'application/json',
    Authorization: `Bearer ${token}`,
  };
  const response = await request.delete(
    `${namespaceUrl(namespace)}/backup-storages/${name}`,
    { headers }
  );
  expect([204, 404]).toContain(response.status());

  const secretResponse = await request.delete(
    `${namespaceUrl(namespace)}/secrets/${backupStorageCredentialsSecretName(name)}`,
    { headers }
  );
  expect([204, 404]).toContain(secretResponse.status());
};
