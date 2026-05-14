import { BackupStorage } from 'shared-types/backupStorages.types';
import { Schedule } from 'shared-types/dbCluster.types';

type GetAvailableStoragesParams = {
  backupStorages: BackupStorage[];
  schedules: Schedule[];
  /** Max distinct storage entries (from BackupClass limits.maxStorages). */
  maxStorages?: number;
  /**
   * Max schedules per storage entry (from BackupClass limits.maxSchedulesPerStorage).
   * When set, storages that already have this many schedules are hidden from the list.
   * e.g. 1 → behaves like old PG slot logic (one schedule per storage);
   *      2 → hide storages with 2+ schedules; undefined → no per-storage filtering.
   */
  maxSchedulesPerStorage?: number;
};

type GetAvailableStoragesResult = {
  storagesToShow: BackupStorage[];
  uniqueStoragesInUse: number;
  limitReached: boolean;
};

export const getAvailableStorages = ({
  backupStorages,
  schedules,
  maxStorages,
  maxSchedulesPerStorage,
}: GetAvailableStoragesParams): GetAvailableStoragesResult => {
  const storagesInSchedules = schedules
    .map((s) => s.backupStorageName)
    .filter(Boolean);
  const uniqueStoragesInUse = new Set(storagesInSchedules).size;

  const limitReached =
    maxStorages !== undefined && uniqueStoragesInUse >= maxStorages;

  let storagesToShow = limitReached
    ? backupStorages.filter((storage) =>
        storagesInSchedules.includes(storage.name)
      )
    : backupStorages;

  // When a per-storage schedule limit is set, hide storages that have already
  // reached that limit (i.e. can't accept another schedule).
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

  return { storagesToShow, uniqueStoragesInUse, limitReached };
};
