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

import { FormDialog } from 'components/form-dialog';
import { useContext, useMemo } from 'react';
import { Typography } from '@mui/material';

import { ScheduleFormDialogContext } from './schedule-form-dialog-context/schedule-form-dialog.context';
import { Messages } from './schedule-form-dialog.messages';
import { schema } from 'components/schedule-form-dialog/schedule-form/schedule-form-schema';
import { scheduleModalDefaultValues } from './schedule-form-dialog.utils';
import { ScheduleFormWrapper } from './schedule-form-wrapper';
import { WizardMode } from 'shared-types/wizard.types';

export const ScheduleFormDialog = () => {
  const {
    mode = WizardMode.New,
    selectedScheduleName,
    openScheduleModal,
    dbInstanceInfo,
    handleClose,
    isPending,
    handleSubmit,
  } = useContext(ScheduleFormDialogContext);

  const { schedules = [] } = dbInstanceInfo;

  const scheduledBackupSchema = useMemo(
    () => schema(schedules, mode),
    [schedules, mode]
  );

  const selectedSchedule = useMemo(() => {
    if (mode === WizardMode.Edit) {
      return schedules.find((item) => item?.name === selectedScheduleName);
    }
  }, [mode, schedules, selectedScheduleName]);

  const values = useMemo(() => {
    return scheduleModalDefaultValues(mode, selectedSchedule);
  }, [mode, selectedSchedule]);

  return (
    <FormDialog
      isOpen={!!openScheduleModal}
      closeModal={handleClose}
      headerMessage={
        mode === WizardMode.New
          ? Messages.createSchedule.headerMessage
          : Messages.editSchedule.headerMessage
      }
      onSubmit={handleSubmit}
      submitting={isPending}
      submitMessage={
        mode === WizardMode.New
          ? Messages.createSchedule.submitMessage
          : Messages.editSchedule.submitMessage
      }
      schema={scheduledBackupSchema}
      {...(mode === WizardMode.Edit && { values })}
      defaultValues={values}
      size="XXL"
      dataTestId={`${mode}-scheduled-backup`}
    >
      {mode === WizardMode.New && (
        <Typography variant="body1">
          {Messages.createSchedule.subhead}
        </Typography>
      )}
      <ScheduleFormWrapper />
    </FormDialog>
  );
};
