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

import { useEffect, useMemo, useRef } from 'react';
import { useFormContext } from 'react-hook-form';
import { autocompleteClasses, Box, Typography } from '@mui/material';
import { AutoCompleteAutoFill } from 'components/auto-complete-auto-fill/auto-complete-auto-fill';
import { useBackupStoragesByNamespace } from 'hooks/api/backup-storages/useBackupStorages';
import { BackupStorageCRD } from 'shared-types/backupStorages.types';
import { Messages } from './backup-storages-input.messages';
import {
  getAvailableStorages,
  getBackupStorageUnavailableReason,
  getStoragesPollInterval,
  withoutDeletedStorages,
} from './backup-storages-input.utils';
import {
  BackupStoragesInputProps,
  ScheduleWithStorage,
} from './backup-storages-input.types';

const BackupStoragesInput = ({
  name = 'storageLocation',
  namespace,
  autoFillProps,
  backupClass,
  maxStorages,
  maxSchedulesPerStorage,
  ...rest
}: BackupStoragesInputProps) => {
  // Derive flat schedules and instanceStorageNames from nested storages when provided
  const { schedules, instanceStorageNames } = useMemo(() => {
    if ('instanceStorages' in rest && rest.instanceStorages) {
      const storages = rest.instanceStorages;
      const derivedSchedules: ScheduleWithStorage[] = storages.flatMap((s) =>
        (s.schedules ?? []).map(() => ({
          backupStorageName: s.storageRef?.name ?? '',
        }))
      );
      const derivedNames = storages
        .map((s) => s.storageRef?.name)
        .filter((n): n is string => Boolean(n));
      return {
        schedules: derivedSchedules,
        instanceStorageNames: derivedNames,
      };
    }
    return {
      schedules: rest.schedules ?? [],
      instanceStorageNames: rest.instanceStorageNames,
    };
  }, [rest]);

  const { getValues, setValue, setError, trigger } = useFormContext();
  const {
    data: namespaceStorages = [],
    isFetching,
    isSuccess,
  } = useBackupStoragesByNamespace(
    namespace,
    backupClass ? { backupClass } : undefined,
    { refetchInterval: (query) => getStoragesPollInterval(query.state.data) }
  );
  const backupStorages = withoutDeletedStorages(
    namespaceStorages,
    new Set(instanceStorageNames)
  );

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

  const classHasNoStorages =
    !!backupClass && isSuccess && namespaceStorages.length === 0;
  const helperText = classHasNoStorages
    ? Messages.noCompatibleStorages
    : maxStorages !== undefined
      ? Messages.storageLimitHelperText(activeStoragesCount, maxStorages)
      : undefined;

  // Show "(in use)" label only when displaying the full namespace list (limit not reached)
  const showInUseHighlight = !limitReached && inUseNames.size > 0;

  const fillsFirst = autoFillProps?.enableFillFirst ?? true;
  const isPickFixedByCaller = !!autoFillProps?.disabled;

  const isMounted = useRef(false);
  useEffect(() => {
    isMounted.current = true;
    return () => {
      isMounted.current = false;
    };
  }, []);

  // A storage picked for another backup class may not work with this one:
  // once the class's list lacks it, the first valid storage takes its place,
  // as on a first fill. A pick the caller fixed, such as an existing
  // schedule's, stays.
  useEffect(() => {
    const value: BackupStorageCRD | null | undefined = getValues(name);
    const pickedName = value?.metadata?.name;
    if (
      isPickFixedByCaller ||
      !isSuccess ||
      !pickedName ||
      backupStorages.some((storage) => storage.metadata?.name === pickedName)
    ) {
      return;
    }
    const replacement = fillsFirst
      ? storagesToShow.find(
          (storage) => !getBackupStorageUnavailableReason(storage)
        )
      : undefined;
    if (replacement) {
      setValue(name, replacement, { shouldValidate: true });
      return;
    }
    setValue(name, null);
    // A validation the backup class change started still sees the old pick
    // and, settling after this, would leave the form valid with no storage.
    // It has settled by the next task.
    setTimeout(() => {
      if (!isMounted.current || getValues(name)) {
        return;
      }
      if (classHasNoStorages) {
        // Says why the field is empty, where a required error would only
        // say that it is.
        setError(name, {
          type: 'manual',
          message: Messages.noCompatibleStorages,
        });
      } else {
        trigger(name);
      }
    });
  }, [
    classHasNoStorages,
    fillsFirst,
    isPickFixedByCaller,
    isSuccess,
    backupStorages,
    storagesToShow,
    name,
    getValues,
    setValue,
    setError,
    trigger,
  ]);

  return (
    <AutoCompleteAutoFill<BackupStorageCRD>
      name={name}
      textFieldProps={{
        label: 'Backup storage',
        helperText,
      }}
      enableFillFirst
      loading={isFetching}
      options={storagesToShow}
      {...autoFillProps}
      controllerProps={{ name, defaultValue: null }}
      autoCompleteProps={{
        isOptionEqualToValue: (option, value) =>
          option.metadata?.name === value.metadata?.name,
        getOptionLabel: (option) =>
          typeof option === 'string' ? option : (option.metadata?.name ?? ''),
        getOptionDisabled: (option) =>
          !!getBackupStorageUnavailableReason(option),
        ...(backupClass && { noOptionsText: Messages.noCompatibleStorages }),
        slotProps: {
          listbox: {
            // The reason stays readable: MUI fades a disabled option.
            sx: {
              [`& .${autocompleteClasses.option}[aria-disabled="true"]`]: {
                opacity: 1,
              },
            },
          },
        },
        renderOption: (props, option) => {
          const storageName = option.metadata?.name ?? '';
          const unavailableReason = getBackupStorageUnavailableReason(option);
          return (
            <Box component="li" {...props} key={storageName}>
              <Box>
                <Typography
                  component="span"
                  variant="body2"
                  color={unavailableReason ? 'text.disabled' : undefined}
                >
                  {storageName}
                </Typography>
                {showInUseHighlight && inUseNames.has(storageName) && (
                  <Typography
                    component="span"
                    variant="body2"
                    sx={{
                      color: 'text.secondary',
                      ml: 1,
                    }}
                  >
                    {Messages.inUseLabel}
                  </Typography>
                )}
                {unavailableReason && (
                  <Typography
                    variant="caption"
                    color="text.secondary"
                    sx={{ display: 'block' }}
                  >
                    {unavailableReason}
                  </Typography>
                )}
              </Box>
            </Box>
          );
        },
      }}
      disabled={isDisabled}
    />
  );
};

export default BackupStoragesInput;
