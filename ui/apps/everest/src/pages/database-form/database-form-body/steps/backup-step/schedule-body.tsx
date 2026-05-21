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

import { Stack, Typography } from '@mui/material';
import { FlattenedSchedule } from 'components/schedule-form-dialog/schedule-form-dialog-context/schedule-form-dialog-context.types';
import { getTimeSelectionPreviewMessage } from 'pages/database-form/database-preview/database.preview.messages';
import { getFormValuesFromCronExpression } from 'components/time-selection/time-selection.utils';

export const ScheduleContent = ({
  schedule,
  storageName,
}: {
  schedule: FlattenedSchedule;
  storageName: string;
}) => {
  return (
    <Stack
      direction="row"
      alignItems="center"
      sx={{
        width: '100%',
      }}
    >
      <Stack
        sx={{
          width: '50%',
        }}
      >
        <Typography variant="body1">{schedule.name}</Typography>
        <Typography variant="body2">
          {getTimeSelectionPreviewMessage(
            getFormValuesFromCronExpression(schedule.cron)
          )}
        </Typography>
      </Stack>
      <Typography
        sx={{
          width: '50%',
        }}
        variant="body2"
      >
        Storage: {storageName}
      </Typography>
    </Stack>
  );
};
