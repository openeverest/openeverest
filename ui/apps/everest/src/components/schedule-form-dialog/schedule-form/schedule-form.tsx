// everest
// Copyright (C) 2023 Percona LLC
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

import {
  AutoCompleteInput,
  LabeledContent,
  SelectInput,
  TextInput,
} from '@percona/ui-lib';
import { TimeSelection } from '../../time-selection/time-selection';
import { BackupConfigFields } from 'components/backup-config-fields';
import { FormMode } from 'components/ui-generator/ui-generator.types';
import { Messages } from './schedule-form.messages';
import { ScheduleFormFields, ScheduleFormProps } from './schedule-form.types';
import { Alert, MenuItem } from '@mui/material';
import { useFormContext } from 'react-hook-form';
import { useContext, useMemo } from 'react';
import { ScheduleFormDialogContext } from '../schedule-form-dialog-context/schedule-form-dialog.context';
import BackupStoragesInput from 'components/backup-storages-input';

export const ScheduleForm = ({
  allowScheduleSelection,
  disableStorageSelection = false,
  autoFillLocation = false,
  disableNameInput,
  schedules,
  disableNameEdit = false,
  maxStorages,
  maxSchedulesPerStorage,
  instanceStorageNames,
  availableClasses,
  disableClassSelection = false,
  backupClass,
}: ScheduleFormProps) => {
  const {
    formState: { errors },
  } = useFormContext();
  const schedulesNamesList =
    (schedules && schedules.map((item) => item?.name)) || [];
  const {
    dbInstanceInfo: { namespace },
  } = useContext(ScheduleFormDialogContext);

  // Map flattened schedules to the shape BackupStoragesInput expects
  const storageSchedules = useMemo(
    () =>
      schedules.map((s) => ({
        backupStorageName: s.storageName,
      })),
    [schedules]
  );

  const errorInfoAlert = errors?.root ? (
    <Alert data-testid="same-schedule-warning" severity="error">
      {errors?.root?.message}
    </Alert>
  ) : null;

  return (
    <>
      <LabeledContent label={Messages.backupDetails}>
        {allowScheduleSelection ? (
          <AutoCompleteInput
            name={ScheduleFormFields.scheduleName}
            textFieldProps={{
              label: Messages.scheduleName.label,
            }}
            options={schedulesNamesList}
            isRequired
            disabled={disableNameEdit}
          />
        ) : (
          <TextInput
            name={ScheduleFormFields.scheduleName}
            textFieldProps={{
              label: Messages.scheduleName.label,
              disabled: disableNameInput,
            }}
            isRequired
          />
        )}
        <SelectInput
          name={ScheduleFormFields.backupClassName}
          label={Messages.backupClass.label}
          helperText={
            disableClassSelection
              ? Messages.backupClass.disabledHelperText
              : undefined
          }
          formControlProps={{ disabled: disableClassSelection }}
          selectFieldProps={{
            label: Messages.backupClass.label,
            disabled: disableClassSelection,
          }}
        >
          {availableClasses.map((bc) => (
            <MenuItem key={bc.metadata?.name} value={bc.metadata?.name ?? ''}>
              {bc.spec?.displayName || bc.metadata?.name}
            </MenuItem>
          ))}
        </SelectInput>
      </LabeledContent>
      <BackupStoragesInput
        namespace={namespace}
        schedules={storageSchedules}
        maxStorages={maxStorages}
        maxSchedulesPerStorage={maxSchedulesPerStorage}
        instanceStorageNames={instanceStorageNames}
        autoFillProps={{
          isRequired: true,
          enableFillFirst: autoFillLocation,
          disabled: disableStorageSelection,
        }}
      />
      <TextInput
        name={ScheduleFormFields.retentionCopies}
        textFieldProps={{
          type: 'number',
          label: Messages.retentionCopies.label,
          helperText: Messages.retentionCopies.helperText,
        }}
        isRequired
      />
      <LabeledContent label={Messages.repeats}>
        <TimeSelection showInfoAlert errorInfoAlert={errorInfoAlert} />
      </LabeledContent>
      <BackupConfigFields
        backupClass={backupClass}
        formMode={FormMode.New}
        namespace={namespace}
      />
    </>
  );
};
