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

import { PreviewContentText } from '../preview-section';
import { SectionProps } from './section.types';
import {
  Messages,
  getTimeSelectionPreviewMessage,
} from '../database.preview.messages';
import { getFormValuesFromCronExpression } from 'components/time-selection/time-selection.utils';
import { FlattenedSchedule } from 'components/schedule-form-dialog/schedule-form-dialog-context/schedule-form-dialog-context.types';

export const PreviewBackupSection = (props: SectionProps) => {
  const backup = (props as Record<string, unknown>).backup as
    | {
        schedules?: FlattenedSchedule[];
        pitr?: Record<string, { enabled?: boolean } | undefined>;
      }
    | undefined;
  const schedules = backup?.schedules ?? [];
  const pitrStorages = Object.entries(backup?.pitr ?? {})
    .filter(([, config]) => config?.enabled)
    .map(([storageName]) => storageName);

  // The wizard attaches PITR only to storages that have a schedule, while a
  // preset is deployed as is: its PITR runs with no schedule at all.
  const hasPitrWithoutSchedules =
    Boolean(props.presetId) && pitrStorages.length > 0;

  if (schedules.length === 0 && !hasPitrWithoutSchedules) {
    return <PreviewContentText text={Messages.noBackupSchedules} />;
  }

  return (
    <>
      {schedules.length === 0 && (
        <PreviewContentText text={Messages.noBackupSchedules} />
      )}
      {schedules.map((schedule) => (
        <PreviewContentText
          key={schedule.name}
          text={`${getTimeSelectionPreviewMessage(getFormValuesFromCronExpression(schedule.cron))} → ${schedule.storageName}`}
        />
      ))}
      <PreviewContentText
        text={
          pitrStorages.length > 0
            ? `PITR: enabled on ${pitrStorages.join(', ')}`
            : 'PITR: disabled'
        }
      />
    </>
  );
};
