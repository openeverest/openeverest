import { Typography } from '@mui/material';
import { AutoCompleteAutoFill } from 'components/auto-complete-auto-fill/auto-complete-auto-fill';
import { useBackupStoragesByNamespace } from 'hooks/api/backup-storages/useBackupStorages';
import { BackupStorage } from 'shared-types/backupStorages.types';
import { Messages } from './backup-storages-input.messages';
import { getAvailableStorages } from './backup-storages-input.utils';
import { BackupStoragesInputProps } from './backup-storages-input.types';

const BackupStoragesInput = ({
  name = 'storageLocation',
  namespace,
  schedules,
  autoFillProps,
  maxStorages,
  maxSchedulesPerStorage,
  instanceStorageNames,
}: BackupStoragesInputProps) => {
  const { data: backupStorages = [], isFetching } =
    useBackupStoragesByNamespace(namespace);

  const {
    storagesToShow,
    activeStoragesCount,
    shouldDisable,
    inUseNames,
    limitReached,
  } = getAvailableStorages({
    backupStorages,
    schedules,
    maxStorages,
    maxSchedulesPerStorage,
    instanceStorageNames,
  });

  const isDisabled = shouldDisable || autoFillProps?.disabled;

  const helperText =
    maxStorages !== undefined
      ? Messages.storageLimitHelperText(activeStoragesCount, maxStorages)
      : undefined;

  // Show "(in use)" label only when displaying the full namespace list (limit not reached)
  const showInUseHighlight = !limitReached && inUseNames.size > 0;

  return (
    <AutoCompleteAutoFill<BackupStorage>
      name={name}
      textFieldProps={{
        label: 'Backup storage',
        helperText,
      }}
      enableFillFirst
      loading={isFetching}
      options={storagesToShow}
      controllerProps={{ name, defaultValue: null }}
      autoCompleteProps={{
        isOptionEqualToValue: (option, value) => option.name === value.name,
        getOptionLabel: (option) =>
          typeof option === 'string' ? option : option.name,
        ...(showInUseHighlight && {
          renderOption: (props, option) => (
            <li {...props} key={option.name}>
              {option.name}
              {inUseNames.has(option.name) && (
                <Typography
                  component="span"
                  variant="body2"
                  color="text.secondary"
                  sx={{ ml: 1 }}
                >
                  {Messages.inUseLabel}
                </Typography>
              )}
            </li>
          ),
        }),
      }}
      disabled={isDisabled}
      {...autoFillProps}
    />
  );
};

export default BackupStoragesInput;
