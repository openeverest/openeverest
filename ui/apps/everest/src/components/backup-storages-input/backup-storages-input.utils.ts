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

import { BackupStorage } from 'shared-types/backupStorages.types';
import { Schedule } from 'shared-types/dbCluster.types';

export type GetAvailableStoragesParams = {
  backupStorages: BackupStorage[];
  schedules: Schedule[];
  maxStorages?: number;
  // Applied as a cascading filter AFTER the maxStorages filter.
  maxSchedulesPerStorage?: number;
  // instance.spec.backup.storages[].storageRef.name
  instanceStorageNames?: string[];
};

export type GetAvailableStoragesResult = {
  storagesToShow: BackupStorage[];
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

  let storagesToShow: BackupStorage[];

  if (
    activeStoragesCount === 0 ||
    maxStorages === undefined ||
    maxStorages > activeStoragesCount
  ) {
    // Limit not reached (or no limit / no active storages): show all namespace storages
    storagesToShow = backupStorages;
  } else {
    // Limit reached: show only instance storages
    storagesToShow = backupStorages.filter((s) => inUseNames.has(s.name));
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
        (schedulesPerStorage[storage.name] ?? 0) < maxSchedulesPerStorage
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
