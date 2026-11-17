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

import { CrdsGen, HttpApi } from '@generated/api-types';

/** The full CRD type returned by the v2 cluster-scoped API. */
export type BackupStorageCRD = CrdsGen.components['schemas']['BackupStorage'];
export type BackupStorageListCRD =
  CrdsGen.components['schemas']['BackupStorageList'];
export type BackupStorageSpec = BackupStorageCRD['spec'];

// What a save sends: the generated type can't hold a class's parameters.
export type BackupStoragePayload = Omit<BackupStorageCRD, 'spec' | 'status'> & {
  spec: Omit<BackupStorageSpec, 'parameters'> & {
    parameters?: Record<string, unknown>;
  };
};

// backupClass: the storages that backup class can use;
// backupStorageClass: those of the class; secret: those that use the Secret.
export type BackupStorageListFilter = NonNullable<
  HttpApi.operations['listBackupStorages']['parameters']['query']
>;

export type BackupStorageCondition = NonNullable<
  NonNullable<BackupStorageCRD['status']>['conditions']
>[number];

export type BackupStorageClass =
  CrdsGen.components['schemas']['BackupStorageClass'];
export type BackupStorageClassList =
  CrdsGen.components['schemas']['BackupStorageClassList'];

export enum BackupStorageState {
  // Valid for its class; nothing has tested that the store is reachable.
  Valid = 'valid',
  Pending = 'pending',
  Invalid = 'invalid',
  // Deleted, waiting until nothing uses it.
  Deleting = 'deleting',
}

// The Accepted and Deleting conditions rolled into one state, with the
// condition that explains it.
export interface BackupStorageStatus {
  state: BackupStorageState;
  reason?: string;
  message?: string;
}
