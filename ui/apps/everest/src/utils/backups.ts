import { DbType } from '@percona/types';
import { BACKUP_STATE_TO_STATUS, PG_SLOTS_LIMIT } from 'consts';
import { Backup, BackupStatus } from 'shared-types/backupsOld.types';
import { BackupStorage } from 'shared-types/backupStorages.types';
import { Schedule } from 'shared-types/dbCluster.types';

export const mapBackupState = (backupState: string) =>
  BACKUP_STATE_TO_STATUS[backupState] || BackupStatus.UNKNOWN;

export const getAvailableBackupStoragesForBackups = (
  backups: Backup[],
  schedules: Schedule[],
  backupStorages: BackupStorage[],
  dbType: DbType,
  hideUsedStoragesInSchedules = false
): {
  uniqueStoragesInUse: string[];
  storagesToShow: BackupStorage[];
} => {
  const storagesInDemandBackups = backups.map(
    (backup) => backup.backupStorageName
  );
  const storagesInSchedules = schedules.map(
    (schedule) => schedule.backupStorageName
  );

  const storagesInUse = [...storagesInDemandBackups, ...storagesInSchedules];
  const uniqueStoragesInUse = [...new Set(storagesInUse)];
  const pgLimitAchieved =
    dbType === DbType.Postresql && uniqueStoragesInUse.length >= PG_SLOTS_LIMIT;

  let storagesToShow: BackupStorage[] = pgLimitAchieved
    ? backupStorages.filter((storage) =>
        uniqueStoragesInUse.includes(storage.name)
      )
    : backupStorages;

  if (hideUsedStoragesInSchedules) {
    storagesToShow = storagesToShow.filter(
      (storage) => !storagesInSchedules.includes(storage.name)
    );
  }

  return {
    uniqueStoragesInUse,
    storagesToShow,
  };
};
