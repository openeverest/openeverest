import { DbType } from '@percona/types';
import { AutoCompleteAutoFill } from 'components/auto-complete-auto-fill/auto-complete-auto-fill';
import { AutoCompleteAutoFillProps } from 'components/auto-complete-auto-fill/auto-complete-auto-fill.types';
import { useBackupStoragesByNamespace } from 'hooks/api/backup-storages/useBackupStorages';
import { useDbBackups } from 'hooks/api/backups/useBackups';
import { BackupStorage } from 'shared-types/backupStorages.types';
import { Schedule } from 'shared-types/dbCluster.types';
import { Messages } from './BackupStoragesInput.messages';
import { getAvailableBackupStoragesForBackups } from 'utils/backups';

type Props = {
  dbClusterName?: string;
  namespace: string;
  dbType: DbType;
  schedules: Schedule[];
  autoFillProps?: Partial<AutoCompleteAutoFillProps<BackupStorage>>;
  hideUsedStoragesInSchedules?: boolean;
};

const BackupStoragesInput = ({
  namespace,
  dbClusterName,
  dbType,
  schedules,
  autoFillProps,
  hideUsedStoragesInSchedules,
}: Props) => {
  const { data: backupStorages = [], isFetching: fetchingStorages } =
    useBackupStoragesByNamespace(namespace);
  const { data: backups = [], isFetching: fetchingBackups } = useDbBackups(
    dbClusterName!,
    namespace,
    {
      enabled: !!dbClusterName && dbType === DbType.Postresql,
    }
  );
  const isFetching = fetchingStorages || fetchingBackups;
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
