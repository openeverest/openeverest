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

import { Typography } from '@mui/material';
import { AutoCompleteAutoFill } from 'components/auto-complete-auto-fill/auto-complete-auto-fill';
import { useBackupStoragesByNamespace } from 'hooks/api/backup-storages/useBackupStorages';
import { useMemo } from 'react';
import { BackupStorageCRD } from 'shared-types/backupStorages.types';
import { Messages } from './backup-storages-input.messages';
import { getAvailableStorages } from './backup-storages-input.utils';
import {
  BackupStoragesInputProps,
  ScheduleWithStorage,
} from './backup-storages-input.types';

const BackupStoragesInput = ({
  name = 'storageLocation',
  namespace,
  autoFillProps,
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
        ...(showInUseHighlight && {
          renderOption: (props, option) => (
            <li {...props} key={option.metadata?.name}>
              {option.metadata?.name}
              {inUseNames.has(option.metadata?.name ?? '') && (
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
    />
  );
};

export default BackupStoragesInput;
