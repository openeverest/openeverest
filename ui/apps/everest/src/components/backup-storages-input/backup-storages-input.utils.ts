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
  } else if (maxStorages === activeStoragesCount && maxStorages > 1) {
    // Limit reached, multiple storages: show only instance storages
    storagesToShow = backupStorages.filter((s) => inUseNames.has(s.name));
  } else {
    // Limit reached, single storage (limit == active == 1): show the single storage
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
