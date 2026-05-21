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

import { useContext, useEffect, useMemo } from 'react';
import { useFormContext } from 'react-hook-form';
import { ScheduleFormDialogContext } from '../schedule-form-dialog-context/schedule-form-dialog.context';
import { ScheduleFormFields } from '../schedule-form/schedule-form.types';
import { ScheduleForm } from '../schedule-form/schedule-form';
import { WizardMode } from 'shared-types/wizard.types';

export const ScheduleFormWrapper = () => {
  const { watch, trigger } = useFormContext();
  const {
    mode = WizardMode.New,
    setSelectedScheduleName,
    dbInstanceInfo,
  } = useContext(ScheduleFormDialogContext);
  const { schedules = [], defaultSchedules = [], backupClass } = dbInstanceInfo;

  const [scheduleName] = watch([ScheduleFormFields.scheduleName]);

  const isJustAddedSchedule = !defaultSchedules.find(
    (item) => item?.name === scheduleName
  );
  const disableStorageSelection =
    mode === WizardMode.Edit && !isJustAddedSchedule;

  // Extract limits from the backup class
  const maxStorages =
    backupClass?.spec?.providerManaged?.limits?.maxStorages;
  const maxSchedulesPerStorage =
    backupClass?.spec?.providerManaged?.limits?.maxSchedulesPerStorage;

  // Compute active storage names from existing schedules
  // (in create/wizard mode these are the storages chosen so far in the form)
  const instanceStorageNames = useMemo(
    () => [...new Set(schedules.map((s) => s.storageName).filter(Boolean))],
    [schedules]
  );

  const [amPm, hour, minute, onDay, weekDay, selectedTime] = watch([
    ScheduleFormFields.amPm,
    ScheduleFormFields.hour,
    ScheduleFormFields.minute,
    ScheduleFormFields.onDay,
    ScheduleFormFields.weekDay,
    ScheduleFormFields.selectedTime,
  ]);

  useEffect(() => {
    trigger();
  }, [amPm, hour, minute, onDay, weekDay, selectedTime, trigger]);

  useEffect(() => {
    if (mode === WizardMode.Edit && setSelectedScheduleName) {
      setSelectedScheduleName(scheduleName);
    }
  }, [scheduleName, mode, setSelectedScheduleName]);

  return (
    <ScheduleForm
      allowScheduleSelection={mode === WizardMode.Edit}
      disableStorageSelection={disableStorageSelection}
      autoFillLocation={mode === WizardMode.New}
      disableNameEdit={mode === WizardMode.Edit}
      schedules={schedules}
      maxStorages={maxStorages}
      maxSchedulesPerStorage={maxSchedulesPerStorage}
      instanceStorageNames={instanceStorageNames}
    />
  );
};
