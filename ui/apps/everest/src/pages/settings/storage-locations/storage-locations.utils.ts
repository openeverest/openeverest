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

import { UseQueryResult } from '@tanstack/react-query';
import {
  BackupStorageCRD,
  StorageType,
} from 'shared-types/backupStorages.types';
import { Messages } from './storage-locations.messages';
import { BackupStorageTableElement } from './storage-locations.types';

export const convertStoragesType = (value: StorageType) =>
  ({
    [StorageType.S3]: Messages.s3,
  })[value] ?? value;

export const convertBackupStoragesPayloadToTableFormat = (
  data: UseQueryResult<BackupStorageCRD[], Error>[]
): BackupStorageTableElement[] => {
  return data.flatMap((item) =>
    item.isSuccess
      ? item.data.map((storage) => ({
          namespace: storage.metadata?.namespace ?? '',
          name: storage.metadata?.name ?? '',
          type: (storage.spec?.type as StorageType) ?? StorageType.S3,
          bucketName: storage.spec?.s3?.bucket ?? '',
          url: storage.spec?.s3?.endpointURL ?? '',
          raw: storage,
        }))
      : []
  );
};
