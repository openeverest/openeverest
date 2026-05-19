import { Typography } from '@mui/material';
import { useEffect } from 'react';
import { useFormContext } from 'react-hook-form';
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
  enableFillFirst?: boolean;
  autoFillProps?: Omit<
    Partial<AutoCompleteAutoFillProps<BackupStorage>>,
    'enableFillFirst'
  >;
  maxStorages?: number;
  maxSchedulesPerStorage?: number;
  /** Storage names currently active on the instance (instance.spec.backup.storages[].storageRef.name). */
  instanceStorageNames?: string[];
};

const BackupStoragesInput = ({
  name = 'storageLocation',
  namespace,
  schedules,
  enableFillFirst = true,
  autoFillProps,
  maxStorages,
  maxSchedulesPerStorage,
  instanceStorageNames,
}: Props) => {
  const { data: backupStorages = [], isFetching, isLoading } =
    useBackupStoragesByNamespace(namespace);
  const { setValue, getValues } = useFormContext();

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

  useEffect(() => {
    if (!enableFillFirst || isLoading || !storagesToShow.length) return;
    const current = getValues(name);
    if (current == null) {
      setValue(name, storagesToShow[0], { shouldValidate: true });
    }
  }, [storagesToShow, enableFillFirst, isLoading, name, setValue, getValues]);

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
      {...(isDisabled && { disabled: true })}
    />
  );
};

export default BackupStoragesInput;
