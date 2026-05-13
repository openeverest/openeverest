import { DbType } from '@percona/types';
import { AutoCompleteAutoFill } from 'components/auto-complete-auto-fill/auto-complete-auto-fill';
import { AutoCompleteAutoFillProps } from 'components/auto-complete-auto-fill/auto-complete-auto-fill.types';
import { useBackupStoragesByNamespace } from 'hooks/api/backup-storages/useBackupStorages';
import { BackupStorage } from 'shared-types/backupStorages.types';
import { Schedule } from 'shared-types/dbCluster.types';
import { Messages } from './BackupStoragesInput.messages';
import { getAvailableBackupStoragesForBackups } from 'utils/backups';

type Props = {
  dbInstanceName?: string;
  namespace: string;
  dbType: DbType;
  schedules: Schedule[];
  autoFillProps?: Partial<AutoCompleteAutoFillProps<BackupStorage>>;
  hideUsedStoragesInSchedules?: boolean;
};

const BackupStoragesInput = ({
  namespace,
  dbInstanceName: _dbInstanceName,
  dbType,
  schedules,
  autoFillProps,
  hideUsedStoragesInSchedules,
}: Props) => {
  const { data: backupStorages = [], isFetching: fetchingStorages } =
    useBackupStoragesByNamespace(namespace);
  // In v1, useDbBackups was called here only for PG to enforce PG_SLOTS_LIMIT (3 storage repos).
  // In v2, storage limits are provider-driven (via BackupClass), so this client-side check is obsolete.
  // TODO: re-evaluate if backup-based storage filtering is needed with v2 BackupClass limits
  const backups: never[] = [];
  const isFetching = fetchingStorages;
  const { storagesToShow, uniqueStoragesInUse } =
    getAvailableBackupStoragesForBackups(
      backups,
      schedules,
      backupStorages,
      dbType,
      hideUsedStoragesInSchedules
    );

  return (
    <AutoCompleteAutoFill<BackupStorage>
      name="storageLocation"
      textFieldProps={{
        label: 'Backup storage',
        helperText:
          dbType === DbType.Postresql && !autoFillProps?.disabled
            ? Messages.pgHelperText(uniqueStoragesInUse.length)
            : undefined,
      }}
      loading={isFetching}
      options={storagesToShow}
      enableFillFirst
      autoCompleteProps={{
        isOptionEqualToValue: (option, value) => option.name === value.name,
        getOptionLabel: (option) =>
          typeof option === 'string' ? option : option.name,
      }}
      {...autoFillProps}
    />
  );
};

export default BackupStoragesInput;
