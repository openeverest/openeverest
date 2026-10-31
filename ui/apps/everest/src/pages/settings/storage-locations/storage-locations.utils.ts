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
import { BaseStatus } from 'components/status-field/status-field.types';
import {
  BackupStorageClass,
  BackupStorageCRD,
} from 'shared-types/backupStorages.types';
import {
  BACKUP_STORAGE_ACCEPTED_CONDITION,
  BACKUP_STORAGE_DELETING_CONDITION,
  formatBackupStorageLocation,
  getBackupStorageClassDisplayName,
  getBackupStorageCondition,
} from 'utils/backup-storages';
import { Messages } from './storage-locations.messages';
import {
  BackupStorageStatus,
  BackupStorageTableElement,
} from './storage-locations.types';

export const BACKUP_STORAGE_STATUS_TO_BASE_STATUS: Record<
  BackupStorageStatus,
  BaseStatus
> = {
  [BackupStorageStatus.Accepted]: 'success',
  [BackupStorageStatus.Rejected]: 'error',
  [BackupStorageStatus.Deleting]: 'deleting',
  [BackupStorageStatus.Pending]: 'pending',
};

export const getBackupStorageStatus = (
  storage: BackupStorageCRD
): Pick<
  BackupStorageTableElement,
  'status' | 'statusLabel' | 'statusMessage'
> => {
  const deleting = getBackupStorageCondition(
    storage,
    BACKUP_STORAGE_DELETING_CONDITION
  );
  if (deleting?.status === 'True') {
    return {
      status: BackupStorageStatus.Deleting,
      statusLabel: Messages.status.deleting,
      statusMessage: deleting.message,
    };
  }

  const accepted = getBackupStorageCondition(
    storage,
    BACKUP_STORAGE_ACCEPTED_CONDITION
  );
  switch (accepted?.status) {
    case 'True':
      return {
        status: BackupStorageStatus.Accepted,
        statusLabel: Messages.status.accepted,
      };
    case 'False':
      return {
        status: BackupStorageStatus.Rejected,
        statusLabel: accepted.reason,
        statusMessage: accepted.message,
      };
    default:
      return {
        status: BackupStorageStatus.Pending,
        statusLabel: Messages.status.pending,
        statusMessage: accepted?.message,
      };
  }
};

export const convertBackupStoragesPayloadToTableFormat = (
  data: UseQueryResult<BackupStorageCRD[], Error>[],
  backupStorageClasses: BackupStorageClass[]
): BackupStorageTableElement[] =>
  data.flatMap((item) =>
    item.isSuccess
      ? item.data.map((storage) => {
          const className = storage.spec.backupStorageClassRef.name;
          return {
            name: storage.metadata?.name ?? '',
            namespace: storage.metadata?.namespace ?? '',
            backupStorageClass: getBackupStorageClassDisplayName(
              backupStorageClasses.find(
                (backupStorageClass) =>
                  backupStorageClass.metadata?.name === className
              ),
              className
            ),
            location: formatBackupStorageLocation(storage),
            authMode: storage.spec.authentication.mode,
            ...getBackupStorageStatus(storage),
            raw: storage,
          };
        })
      : []
  );
