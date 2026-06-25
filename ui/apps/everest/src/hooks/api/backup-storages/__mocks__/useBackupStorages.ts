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

import { BackupStorageCRD } from 'shared-types/backupStorages.types';

const storageDataObject = {
  metadata: {
    name: 'backup-storage-1',
    namespace: 'the-dark-side',
  },
  spec: {
    type: 's3',
    s3: {
      bucket: 'bucket-001',
      region: 'Us',
      credentialsSecretName: 'secret-1',
      endpointURL: 'http://localhost',
    },
  },
} as unknown as BackupStorageCRD;

// was moved as separate object to avoid recreation since the original use of useQuery caches the data
const backupStorageMockData = {
  data: [storageDataObject],
};

export const useBackupStorages = () => backupStorageMockData;

export const useCreateBackupStorage = () => storageDataObject;

export const useBackupStoragesByNamespace = (namespace: string) => {
  return {
    data: backupStorageMockData.data.filter(
      (item) => item.metadata?.namespace === namespace
    ),
  };
};
