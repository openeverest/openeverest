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
import { ScheduleRetention } from 'shared-types/backups.types';
import {
  DEFAULT_RETENTION_DURATION_UNIT,
  DEFAULT_RETENTION_DURATION_VALUE,
  RETENTION_DURATION_PATTERN,
  RetentionDurationUnit,
  RetentionType,
} from './schedule-form.constants';
import { ScheduleFormFields } from './schedule-form.types';

/** Known static field keys in ScheduleFormData (everything else is dynamic config). */
const STATIC_KEYS = new Set([
  'scheduleName',
  'backupClassName',
  'storageLocation',
  'retentionType',
  'retentionCopies',
  'retentionDurationValue',
  'retentionDurationUnit',
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

export const parseRetentionDuration = (
  duration: string
): {
  value: string;
  unit: RetentionDurationUnit;
} => {
  const match = duration.match(RETENTION_DURATION_PATTERN);
  if (!match) {
    return {
      value: DEFAULT_RETENTION_DURATION_VALUE,
      unit: DEFAULT_RETENTION_DURATION_UNIT,
    };
  }
  return {
    value: match[1],
    unit: match[2] as RetentionDurationUnit,
  };
};

export const retentionFieldsFromApi = (
  retention: ScheduleRetention | undefined
) => {
  if (retention?.type === 'count' && retention.count != null) {
    return {
      [ScheduleFormFields.retentionType]: RetentionType.count,
      [ScheduleFormFields.retentionCopies]: String(retention.count),
      [ScheduleFormFields.retentionDurationValue]:
        DEFAULT_RETENTION_DURATION_VALUE,
      [ScheduleFormFields.retentionDurationUnit]:
        DEFAULT_RETENTION_DURATION_UNIT,
    };
  }
  if (retention?.type === 'time' && retention.duration) {
    const { value, unit } = parseRetentionDuration(retention.duration);
    return {
      [ScheduleFormFields.retentionType]: RetentionType.time,
      [ScheduleFormFields.retentionCopies]: '1',
      [ScheduleFormFields.retentionDurationValue]: value,
      [ScheduleFormFields.retentionDurationUnit]: unit,
    };
  }
  return {
    [ScheduleFormFields.retentionType]: RetentionType.keepAll,
    [ScheduleFormFields.retentionCopies]: '1',
    [ScheduleFormFields.retentionDurationValue]:
      DEFAULT_RETENTION_DURATION_VALUE,
    [ScheduleFormFields.retentionDurationUnit]:
      DEFAULT_RETENTION_DURATION_UNIT,
  };
};

export const retentionToApi = (
  formData: Pick<
    ScheduleFormData,
    | 'retentionType'
    | 'retentionCopies'
    | 'retentionDurationValue'
    | 'retentionDurationUnit'
  >
): ScheduleRetention | undefined => {
  const {
    retentionType,
    retentionCopies,
    retentionDurationValue,
    retentionDurationUnit,
  } = formData;

  if (retentionType === RetentionType.count) {
    return {
      type: 'count',
      count: parseInt(retentionCopies, 10),
    };
  }
  if (retentionType === RetentionType.time) {
    return {
      type: 'time',
      duration: `${retentionDurationValue}${retentionDurationUnit}`,
    };
  }
  return undefined;
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

  // Extract dynamic parameters fields (UIGenerator backup parameters) from
  // form data. UIGenerator registers fields with sectionKey prefix
  // ("parameters.X"), so in form data they appear as a nested object:
  // { parameters: { compressionType: ... } }.
  // Unwrap one level to produce the flat parameters the API expects.
  const dynamicFields = Object.fromEntries(
    Object.entries(formData).filter(([key]) => !STATIC_KEYS.has(key))
  );
  const rawParameters =
    'parameters' in dynamicFields &&
    typeof dynamicFields.parameters === 'object' &&
    dynamicFields.parameters !== null
      ? (dynamicFields.parameters as Record<string, unknown>)
      : dynamicFields;
  const cleanedParameters =
    Object.keys(rawParameters).length > 0
      ? removeEmptyFieldValues(rawParameters)
      : undefined;

  const retention = retentionToApi(formData);

  const newSchedule: FlattenedSchedule = {
    enabled: true,
    name: scheduleName,
    storageName,
    cron,
    ...(retention ? { retention } : {}),
    ...(cleanedParameters && Object.keys(cleanedParameters).length > 0
      ? { parameters: cleanedParameters }
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
