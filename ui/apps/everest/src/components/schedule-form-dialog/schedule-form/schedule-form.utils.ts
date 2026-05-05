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

import { ScheduleFormData } from './schedule-form-schema';
import { DbCluster, Schedule } from 'shared-types/dbCluster.types';
import { getCronExpressionFromFormValues } from '../../time-selection/time-selection.utils';
import { ScheduleWizardMode, WizardMode } from 'shared-types/wizard.types';
import { DbEngineType } from 'shared-types/dbEngines.types';

type UpdateScheduleArrayProps = {
  formData: ScheduleFormData;
  mode: ScheduleWizardMode;
  schedules: Schedule[];
};

export const getSchedulesPayload = ({
  formData,
  mode,
  schedules,
}: UpdateScheduleArrayProps): Schedule[] => {
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
  const backupSchedule = getCronExpressionFromFormValues({
    selectedTime,
    minute,
    hour,
    amPm,
    onDay,
    weekDay,
  });
  let schedulesPayload: Schedule[] = [];

  if (mode === WizardMode.New) {
    schedulesPayload = [
      ...(schedules ?? []),
      {
        enabled: true,
        name: scheduleName,
        backupStorageName:
          typeof storageLocation === 'string'
            ? storageLocation
            : storageLocation!.name,
        schedule: backupSchedule,
        retentionCopies: parseInt(retentionCopies, 10),
      },
    ];
  }

  if (mode === WizardMode.Edit) {
    const newSchedulesArray = schedules && [...(schedules || [])];
    const editedScheduleIndex = newSchedulesArray?.findIndex(
      (item) => item.name === scheduleName
    );
    if (newSchedulesArray && editedScheduleIndex !== undefined) {
      newSchedulesArray[editedScheduleIndex] = {
        enabled: true,
        name: scheduleName,
        backupStorageName:
          typeof storageLocation === 'string'
            ? storageLocation
            : storageLocation!.name,
        schedule: backupSchedule,
        retentionCopies: parseInt(retentionCopies, 10),
      };
      schedulesPayload = newSchedulesArray;
    }
  }
  return schedulesPayload;
};

export const removeScheduleFromArray = (
  name: string,
  schedules: Schedule[]
) => {
  return schedules.filter((item) => item.name !== name);
};

export const backupScheduleFormValuesToDbClusterPayload = (
  dbPayload: ScheduleFormData,
  dbCluster: DbCluster,
  mode: WizardMode
): DbCluster => {
  const {
    selectedTime,
    minute,
    hour,
    amPm,
    onDay,
    weekDay,
    scheduleName,
    retentionCopies,
  } = dbPayload;
  const schedule = getCronExpressionFromFormValues({
    selectedTime,
    minute,
    hour,
    amPm,
    onDay,
    weekDay,
  });

  let schedulesPayload: Schedule[] = [];
  if (mode === WizardMode.New) {
    schedulesPayload = [
      ...(dbCluster.spec.backup?.schedules || []).map((schedule) => ({
        ...schedule,
      })),
      {
        enabled: true,
        retentionCopies: parseInt(retentionCopies, 10),
        name: scheduleName,
        backupStorageName:
          typeof dbPayload.storageLocation === 'string'
            ? dbPayload.storageLocation
            : dbPayload.storageLocation!.name,
        schedule,
      },
    ];
  }

  if (mode === WizardMode.Edit) {
    const schedulesArray = dbCluster?.spec?.backup?.schedules || [];
    const editedScheduleIndex = schedulesArray?.findIndex(
      (item) => item.name === scheduleName
    );
    if (schedulesArray && editedScheduleIndex !== undefined) {
      schedulesArray[editedScheduleIndex] = {
        enabled: true,
        name: scheduleName,
        retentionCopies: parseInt(retentionCopies, 10),
        backupStorageName:
          typeof dbPayload.storageLocation === 'string'
            ? dbPayload.storageLocation
            : dbPayload.storageLocation!.name,
        schedule,
      };
      schedulesPayload = schedulesArray;
    }
  }

  return {
    apiVersion: 'everest.percona.com/v1alpha1',
    kind: 'DatabaseCluster',
    metadata: dbCluster.metadata,
    spec: {
      ...dbCluster?.spec,
      backup: {
        ...dbCluster.spec.backup,
        pitr: {
          ...dbCluster.spec.backup?.pitr,
          backupStorageName:
            dbCluster.spec.backup?.pitr?.backupStorageName || '',
          enabled:
            dbCluster.spec.engine.type === DbEngineType.POSTGRESQL
              ? schedulesPayload.length > 0
              : !!dbCluster.spec.backup?.pitr?.enabled,
        },
        schedules: schedulesPayload.length > 0 ? schedulesPayload : undefined,
      },
    },
  };
};
