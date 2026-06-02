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

import { CrdsGen } from '@generated/api-types';

/** The full CRD type returned by the v2 cluster-scoped API. */
export type BackupStorageCRD = CrdsGen.components['schemas']['BackupStorage'];
export type BackupStorageListCRD =
  CrdsGen.components['schemas']['BackupStorageList'];

/**
 * Flat form-values representation of a backup storage.
 * Used only in create/edit forms and as the input to mutation hooks.
 */
export interface BackupStorageFormValues {
  name: string;
  namespace: string;
  type: StorageType;
  bucketName: string;
  url: string;
  region: string;
  accessKey: string;
  secretKey: string;
  verifyTLS: boolean;
  forcePathStyle: boolean;
}

export enum StorageType {
  S3 = 's3',
  AZURE = 'azure',
  GCS = 'gcs',
}
