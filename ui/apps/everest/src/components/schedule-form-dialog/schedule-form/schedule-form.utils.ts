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

import { ScheduleFormData } from './schedule-form-schema';
import { FlattenedSchedule } from '../schedule-form-dialog-context/schedule-form-dialog-context.types';
import { getCronExpressionFromFormValues } from '../../time-selection/time-selection.utils';
import { ScheduleWizardMode, WizardMode } from 'shared-types/wizard.types';
import { removeEmptyFieldValues } from 'components/ui-generator/utils/postprocess/postprocess-schema';

/** Known static field keys in ScheduleFormData (everything else is dynamic config). */
const STATIC_KEYS = new Set([
  'scheduleName',
  'backupClassName',
  'storageLocation',
  'retentionCopies',
  'selectedTime',
  'minute',
  'hour',
  'amPm',
  'onDay',
  'weekDay',
]);

type UpdateScheduleArrayProps = {
  formData: ScheduleFormData;
  mode: ScheduleWizardMode;
  schedules: FlattenedSchedule[];
};

export const getSchedulesPayload = ({
  formData,
  mode,
  schedules,
}: UpdateScheduleArrayProps): FlattenedSchedule[] => {
  const {
    selectedTime,
    minute,
    hour,
    amPm,
    onDay,
    weekDay,
    scheduleName,
    storageLocation,
    retentionCopies,
  } = formData;
  const cron = getCronExpressionFromFormValues({
    selectedTime,
    minute,
    hour,
    amPm,
    onDay,
    weekDay,
  });

  const storageName =
    typeof storageLocation === 'string'
      ? storageLocation
      : (storageLocation!.metadata?.name ?? '');

  // Extract dynamic config fields (UIGenerator backup config) from form data.
  // UIGenerator registers fields with sectionKey prefix ("config.X"), so in form
  // data they appear as a nested object: { config: { compressionType: ... } }.
  // Unwrap one level to produce the flat config the API expects.
  const dynamicFields = Object.fromEntries(
    Object.entries(formData).filter(([key]) => !STATIC_KEYS.has(key))
  );
  const rawConfig =
    'config' in dynamicFields &&
    typeof dynamicFields.config === 'object' &&
    dynamicFields.config !== null
      ? (dynamicFields.config as Record<string, unknown>)
      : dynamicFields;
  const cleanedConfig =
    Object.keys(rawConfig).length > 0
      ? removeEmptyFieldValues(rawConfig)
      : undefined;

  const newSchedule: FlattenedSchedule = {
    enabled: true,
    name: scheduleName,
    storageName,
    cron,
    retentionCopies: parseInt(retentionCopies, 10),
    ...(cleanedConfig && Object.keys(cleanedConfig).length > 0
      ? { config: cleanedConfig }
      : {}),
  };

  if (mode === WizardMode.New) {
    return [...(schedules ?? []), newSchedule];
  }

  if (mode === WizardMode.Edit) {
    const newSchedulesArray = [...(schedules || [])];
    const editedScheduleIndex = newSchedulesArray.findIndex(
      (item) => item.name === scheduleName
    );
    if (editedScheduleIndex !== -1) {
      newSchedulesArray[editedScheduleIndex] = newSchedule;
    }
    return newSchedulesArray;
  }

  return schedules;
};

export const removeScheduleFromArray = (
  name: string,
  schedules: FlattenedSchedule[]
) => {
  return schedules.filter((item) => item.name !== name);
};
