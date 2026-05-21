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

import { FlattenedSchedule } from './schedule-form-dialog-context/schedule-form-dialog-context.types';
import { getFormValuesFromCronExpression } from 'components/time-selection/time-selection.utils.ts';
import { TIME_SELECTION_DEFAULTS } from '../time-selection/time-selection.constants';
import { ScheduleFormData } from './schedule-form/schedule-form-schema';
import { ScheduleFormFields } from './schedule-form/schedule-form.types';
import { generateShortUID } from 'utils/generateShortUID';
import { WizardMode } from 'shared-types/wizard.types';

export const scheduleModalDefaultValues = (
  mode: WizardMode,
  selectedSchedule?: FlattenedSchedule
): ScheduleFormData => {
  if (mode === WizardMode.Edit && selectedSchedule) {
    const { name, storageName, cron, retentionCopies } = selectedSchedule;
    const formValues = getFormValuesFromCronExpression(cron);
    return {
      [ScheduleFormFields.scheduleName]: name || '',
      [ScheduleFormFields.storageLocation]: { name: storageName },
      [ScheduleFormFields.retentionCopies]: retentionCopies?.toString() || '0',
      ...formValues,
    };
  }
  return {
    [ScheduleFormFields.scheduleName]: `backup-${generateShortUID()}`,
    [ScheduleFormFields.storageLocation]: null,
    [ScheduleFormFields.retentionCopies]: '0',
    ...TIME_SELECTION_DEFAULTS,
  };
};

export const sameScheduleFunc = (
  schedules: FlattenedSchedule[],
  mode: WizardMode,
  currentSchedule: string,
  scheduleName: string
) => {
  if (mode === WizardMode.Edit) {
    return schedules.find(
      (item) => item.cron === currentSchedule && item.name !== scheduleName
    );
  } else {
    return schedules.find((item) => item.cron === currentSchedule);
  }
};

export const sameStorageLocationFunc = (
  schedules: FlattenedSchedule[],
  mode: WizardMode,
  currentBackupStorage: string | { name: string } | undefined | null,
  scheduleName: string
) => {
  const currentStorage =
    typeof currentBackupStorage === 'object'
      ? currentBackupStorage?.name
      : currentBackupStorage;
  if (mode === WizardMode.Edit) {
    return schedules.find(
      (item) =>
        item.storageName === currentStorage && item.name !== scheduleName
    );
  } else {
    return schedules.find((item) => item.storageName === currentStorage);
  }
};
