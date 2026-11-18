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
  BackupStorageClass,
  BackupStorageCRD,
} from 'shared-types/backupStorages.types';
import {
  formatBackupStorageLocation,
  getBackupStorageClassTitle,
  getBackupStorageStatus,
} from 'utils/backup-storages';
import { Messages } from './storage-locations.messages';
import { BackupStorageRow } from './storage-locations.types';

export const toBackupStorageRows = (
  storages: BackupStorageCRD[],
  classes: BackupStorageClass[]
): BackupStorageRow[] =>
  storages.map((storage) => ({
    name: storage.metadata?.name ?? '',
    namespace: storage.metadata?.namespace ?? '',
    type: getBackupStorageClassTitle(
      storage.spec.backupStorageClassRef.name,
      classes
    ),
    location: formatBackupStorageLocation(storage),
    status: getBackupStorageStatus(storage),
    raw: storage,
  }));

export const describeBackupStorageTls = (
  tls: BackupStorageCRD['spec']['tls']
) => {
  if (tls?.insecureSkipVerify) {
    return Messages.details.tlsSkipVerify;
  }
  const caRef = tls?.ca?.secretKeyRef;
  return caRef
    ? Messages.details.tlsCustomCa(caRef.name, caRef.key)
    : Messages.details.tlsSystem;
};
