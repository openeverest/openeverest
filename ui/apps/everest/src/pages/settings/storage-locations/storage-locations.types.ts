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

export type DeleteBackupStorageArgs = {
  backupStorageId: string;
  namespace: string;
};

export enum StorageLocationsFields {
  name = 'name',
  namespace = 'namespace',
  backupStorageClass = 'backupStorageClass',
  spec = 'spec',
  authMode = 'authMode',
  credentialsSource = 'credentialsSource',
  credentialsSecretName = 'credentialsSecretName',
  credentials = 'credentials',
  tlsMode = 'tlsMode',
  caSource = 'caSource',
  caBundle = 'caBundle',
  caSecretName = 'caSecretName',
  caSecretKey = 'caSecretKey',
}

export enum BackupStorageStatus {
  Accepted = 'accepted',
  Rejected = 'rejected',
  Deleting = 'deleting',
  Pending = 'pending',
}

export interface BackupStorageTableElement {
  name: string;
  namespace: string;
  backupStorageClass: string;
  location: string;
  authMode: string;
  status: BackupStorageStatus;
  statusLabel: string;
  statusMessage?: string;
  raw: BackupStorageCRD;
}
