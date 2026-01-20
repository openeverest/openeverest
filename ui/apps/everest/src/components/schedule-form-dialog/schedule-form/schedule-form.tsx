// everest
// Copyright (C) 2023 Percona LLC
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

import { AutoCompleteInput, LabeledContent, TextInput } from '@percona/ui-lib';
import LogicalPhysicalRadioGroup from 'components/logical-physical-radio-group';
import { TimeSelection } from '../../time-selection/time-selection';
import { Messages } from './schedule-form.messages.ts';
import {
  ScheduleFormFields,
  ScheduleFormProps,
} from './schedule-form.types.ts';
import { Alert } from '@mui/material';
import { useFormContext } from 'react-hook-form';
import { useContext } from 'react';
import { ScheduleFormDialogContext } from '../schedule-form-dialog-context/schedule-form-dialog.context';
import { DbEngineType } from '@percona/types';
import LinkedAlert from '../../linked-alert';
import BackupStoragesInput from 'components/backup-storages-input';
import { dbEngineToDbType } from '@percona/utils';

export const ScheduleForm = ({
  allowScheduleSelection,
  disableStorageSelection = false,
  disableNameInput,
  autoFillLocation,
  schedules,
  showTypeRadio,
  disableNameEdit = false,
}: ScheduleFormProps) => {
  const {
    formState: { errors },
  } = useFormContext();
  const schedulesNamesList =
    (schedules && schedules.map((item) => item?.name)) || [];
  const {
    dbClusterInfo: { dbEngine, namespace, dbClusterName },
  } = useContext(ScheduleFormDialogContext);

  const errorInfoAlert = errors?.root ? (
    <Alert data-testid="same-schedule-warning" severity="error">
      {errors?.root?.message}
    </Alert>
  ) : null;

  return (
    <>
      {showTypeRadio && <LogicalPhysicalRadioGroup />}
      <LabeledContent label={Messages.backupDetails}>
        {dbEngine === DbEngineType.POSTGRESQL && disableStorageSelection && (
          <LinkedAlert
            severity="warning"
            message={Messages.pgStorageEditRestriction}
            linkProps={{
              linkContent: 'Learn More',
              href: 'https://openeverest.io/documentation/current/reference/known_limitations.html',
            }}
          />
        )}
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
      </LabeledContent>
      <BackupStoragesInput
        namespace={namespace}
        dbClusterName={dbClusterName}
        dbType={dbEngineToDbType(dbEngine)}
        schedules={schedules}
        autoFillProps={{
          isRequired: true,
          enableFillFirst: autoFillLocation,
          disabled: disableStorageSelection,
        }}
        hideUsedStoragesInSchedules={dbEngine === DbEngineType.POSTGRESQL}
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
        <TimeSelection
          showInfoAlert
          errorInfoAlert={errorInfoAlert}
          shouldRestrictSelectableHours={dbEngine === DbEngineType.PSMDB}
        />
      </LabeledContent>
    </>
  );
};
