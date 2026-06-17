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

import { z } from 'zod';
import {
  BackupStorageCRD,
  StorageType,
} from 'shared-types/backupStorages.types';
import { rfc_123_schema } from 'utils/common-validation';

export type BackupStoragesMutationContext = {
  queryKey: readonly [string, string, string];
  previousStorages?: BackupStorageCRD[];
};

export type DeleteBackupStorageArgs = {
  backupStorageId: string;
  namespace: string;
};

export enum StorageLocationsFields {
  name = 'name',
  type = 'type',
  bucketName = 'bucketName',
  region = 'region',
  url = 'url',
  accessKey = 'accessKey',
  secretKey = 'secretKey',
  namespace = 'namespace',
  verifyTLS = 'verifyTLS',
  forcePathStyle = 'forcePathStyle',
}

export const storageLocationDefaultValues = (namespace: string) => ({
  [StorageLocationsFields.name]: '',
  [StorageLocationsFields.type]: StorageType.S3,
  [StorageLocationsFields.url]: '',
  [StorageLocationsFields.region]: '',
  [StorageLocationsFields.accessKey]: '',
  [StorageLocationsFields.secretKey]: '',
  [StorageLocationsFields.bucketName]: '',
  [StorageLocationsFields.namespace]: namespace,
  [StorageLocationsFields.verifyTLS]: true,
  [StorageLocationsFields.forcePathStyle]: false,
});

export const storageLocationEditValues = (crd: BackupStorageCRD) => ({
  [StorageLocationsFields.name]: crd.metadata?.name ?? '',
  [StorageLocationsFields.type]:
    (crd.spec?.type as StorageType) ?? StorageType.S3,
  [StorageLocationsFields.url]: crd.spec?.s3?.endpointURL ?? '',
  [StorageLocationsFields.region]: crd.spec?.s3?.region ?? '',
  [StorageLocationsFields.accessKey]: crd.spec?.s3?.accessKeyId ?? '',
  [StorageLocationsFields.secretKey]: crd.spec?.s3?.secretAccessKey ?? '',
  [StorageLocationsFields.bucketName]: crd.spec?.s3?.bucket ?? '',
  [StorageLocationsFields.namespace]: crd.metadata?.namespace ?? '',
  [StorageLocationsFields.verifyTLS]: crd.spec?.s3?.verifyTLS ?? true,
  [StorageLocationsFields.forcePathStyle]:
    crd.spec?.s3?.forcePathStyle ?? false,
});

export const storageLocationsSchema = z.object({
  [StorageLocationsFields.name]: rfc_123_schema({
    fieldName: 'storage name',
    maxLength: 22,
  }),
  [StorageLocationsFields.type]: z.nativeEnum(StorageType),
  [StorageLocationsFields.bucketName]: z.string().nonempty(),
  [StorageLocationsFields.url]: z.string().nonempty().url(),
  [StorageLocationsFields.region]: z.string().nonempty(),
  [StorageLocationsFields.accessKey]: z.string().nonempty(),
  [StorageLocationsFields.secretKey]: z.string().nonempty(),
  [StorageLocationsFields.namespace]: z.string().nonempty(),
  [StorageLocationsFields.verifyTLS]: z.boolean(),
  [StorageLocationsFields.forcePathStyle]: z.boolean(),
});

export const storageLocationsEditSchema = storageLocationsSchema.extend({
  [StorageLocationsFields.accessKey]: z.string(),
  [StorageLocationsFields.secretKey]: z.string(),
});

export type BackupStorageType = z.infer<typeof storageLocationsSchema>;

export interface BackupStorageTableElement {
  name: string;
  type: StorageType;
  bucketName: string;
  url: string;
  namespace: string;
  raw: BackupStorageCRD;
}
