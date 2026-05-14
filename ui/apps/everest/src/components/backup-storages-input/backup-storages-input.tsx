import { AutoCompleteAutoFill } from 'components/auto-complete-auto-fill/auto-complete-auto-fill';
import { AutoCompleteAutoFillProps } from 'components/auto-complete-auto-fill/auto-complete-auto-fill.types';
import { useBackupStoragesByNamespace } from 'hooks/api/backup-storages/useBackupStorages';
import { BackupStorage } from 'shared-types/backupStorages.types';
import { Schedule } from 'shared-types/dbCluster.types';
import { Messages } from './backup-storages-input.messages';
import { getAvailableStorages } from './backup-storages-input.utils';

type Props = {
  name?: string;
  namespace: string;
  schedules: Schedule[];
  autoFillProps?: Partial<AutoCompleteAutoFillProps<BackupStorage>>;
  maxStorages?: number;
  maxSchedulesPerStorage?: number;
};

const BackupStoragesInput = ({
  name = 'storageLocation',
  namespace,
  schedules,
  autoFillProps,
  maxStorages,
  maxSchedulesPerStorage,
}: Props) => {
  const { data: backupStorages = [], isFetching } =
    useBackupStoragesByNamespace(namespace);

  const { storagesToShow, uniqueStoragesInUse } = getAvailableStorages({
    backupStorages,
    schedules,
    maxStorages,
    maxSchedulesPerStorage,
  });

  const helperText =
    maxStorages !== undefined && !autoFillProps?.disabled
      ? Messages.storageLimitHelperText(uniqueStoragesInUse, maxStorages)
      : undefined;

  return (
    <AutoCompleteAutoFill<BackupStorage>
      name={name}
      textFieldProps={{
        label: 'Backup storage',
        helperText,
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
