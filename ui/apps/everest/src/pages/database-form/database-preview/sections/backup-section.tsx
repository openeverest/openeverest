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
import { getTimeSelectionPreviewMessage } from '../database.preview.messages';
import { getFormValuesFromCronExpression } from 'components/time-selection/time-selection.utils';
import { FlattenedSchedule } from 'components/schedule-form-dialog/schedule-form-dialog-context/schedule-form-dialog-context.types';

interface PreviewBackupFormState {
  schedules?: FlattenedSchedule[];
  pitr?: Record<string, { enabled?: boolean } | undefined>;
}

// The wizard keeps backup schedules/PITR under a flat `backup` form field that
// is not part of the static DbWizardType (SectionProps), so narrow to read it.
const hasBackupFormState = (
  value: unknown
): value is { backup?: PreviewBackupFormState } =>
  typeof value === 'object' && value !== null && 'backup' in value;

export const PreviewBackupSection = (props: SectionProps) => {
  const backup = hasBackupFormState(props) ? props.backup : undefined;
  const schedules = backup?.schedules ?? [];
  const pitrStorages = Object.entries(backup?.pitr ?? {})
    .filter(([, config]) => config?.enabled)
    .map(([storageName]) => storageName);

  if (schedules.length === 0) {
    return <PreviewContentText text="No backup schedules" />;
  }

  return (
    <>
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
