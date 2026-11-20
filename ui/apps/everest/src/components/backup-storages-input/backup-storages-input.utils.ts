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
  BackupStorageCRD,
  BackupStorageState,
} from 'shared-types/backupStorages.types';
import {
  describeBackupStorageStatus,
  getBackupStorageStatus,
} from 'utils/backup-storages';
import { BackupStorageStateMessages } from 'utils/backup-storages.messages';
import { ScheduleWithStorage } from './backup-storages-input.types';

const PENDING_STORAGES_POLL_INTERVAL_MS = 5000;

// Why backups can't be written to the storage now; none once it is valid.
export const getBackupStorageUnavailableReason = (
  storage: BackupStorageCRD
): string | undefined => {
  const status = getBackupStorageStatus(storage);
  return status.state === BackupStorageState.Valid
    ? undefined
    : (describeBackupStorageStatus(status) ??
        BackupStorageStateMessages[status.state]);
};

// Core checks a storage shortly after it is added; until then it is Pending
// and can't be picked.
export const getStoragesPollInterval = (
  storages: BackupStorageCRD[] | undefined
): number | false =>
  storages?.some(
    (storage) =>
      getBackupStorageStatus(storage).state === BackupStorageState.Pending
  )
    ? PENDING_STORAGES_POLL_INTERVAL_MS
    : false;

// A storage being deleted is offered only where the instance already uses it,
// so its current choice still shows.
export const withoutDeletedStorages = (
  storages: BackupStorageCRD[],
  inUseNames: Set<string>
) =>
  storages.filter(
    (storage) =>
      getBackupStorageStatus(storage).state !== BackupStorageState.Deleting ||
      inUseNames.has(storage.metadata?.name ?? '')
  );

export type GetAvailableStoragesParams = {
  backupStorages: BackupStorageCRD[];
  schedules: ScheduleWithStorage[];
  maxStorages?: number;
  // Applied as a cascading filter AFTER the maxStorages filter.
  maxSchedulesPerStorage?: number;
  // instance.spec.backup.storages[].storageRef.name
  instanceStorageNames?: string[];
};

export type GetAvailableStoragesResult = {
  storagesToShow: BackupStorageCRD[];
  activeStoragesCount: number;
  limitReached: boolean;
  shouldDisable: boolean;
  inUseNames: Set<string>;
};

export const getAvailableStorages = ({
  backupStorages,
  schedules,
  maxStorages,
  maxSchedulesPerStorage,
  instanceStorageNames,
}: GetAvailableStoragesParams): GetAvailableStoragesResult => {
  const inUseNames = new Set(instanceStorageNames ?? []);
  const activeStoragesCount = inUseNames.size;

  const limitReached =
    maxStorages !== undefined &&
    activeStoragesCount > 0 &&
    activeStoragesCount >= maxStorages;

  let storagesToShow: BackupStorageCRD[];

  if (
    activeStoragesCount === 0 ||
    maxStorages === undefined ||
    maxStorages > activeStoragesCount
  ) {
    // Limit not reached (or no limit / no active storages): show all namespace storages
    storagesToShow = backupStorages;
  } else {
    // Limit reached: show only instance storages
    storagesToShow = backupStorages.filter((s) =>
      inUseNames.has(s.metadata?.name ?? '')
    );
  }

  const shouldDisable =
    limitReached && maxStorages === 1 && storagesToShow.length <= 1;

  // Cascading filter: maxSchedulesPerStorage removes storages that can't accept more schedules
  if (maxSchedulesPerStorage !== undefined) {
    const schedulesPerStorage = schedules.reduce<Record<string, number>>(
      (acc, s) => {
        if (s.backupStorageName) {
          acc[s.backupStorageName] = (acc[s.backupStorageName] ?? 0) + 1;
        }
        return acc;
      },
      {}
    );
    storagesToShow = storagesToShow.filter(
      (storage) =>
        (schedulesPerStorage[storage.metadata?.name ?? ''] ?? 0) <
        maxSchedulesPerStorage
    );
  }

  return {
    storagesToShow,
    activeStoragesCount,
    limitReached,
    shouldDisable,
    inUseNames,
  };
};
