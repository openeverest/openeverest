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

import { Box, Skeleton, Stack } from '@mui/material';
import { getMonitoringClassName } from 'utils/monitoring-classes';
import { StepHeader } from 'pages/database-form/database-form-body/steps-old/step-header/step-header';
import { StepProps } from 'pages/database-form/database-form.types';
import { useMonitoringSelection } from './hooks/use-monitoring-selection';
import { MonitoringClassGroup } from './monitoring-class-group/monitoring-class-group';
import { Messages } from './monitoring-step.messages';

export const MonitoringStep = (
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  _props: StepProps
) => {
  const selection = useMonitoringSelection();

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column' }}>
      <StepHeader
        pageTitle={Messages.title}
        pageDescription={Messages.caption}
      />
      {selection.isLoading ? (
        <>
          <Skeleton height="120px" />
          <Skeleton height="120px" />
        </>
      ) : (
        <Stack gap={4} sx={{ mt: 3 }}>
          {selection.groups.map((group) => (
            <MonitoringClassGroup
              key={getMonitoringClassName(group.monitoringClass)}
              group={group}
              selection={selection}
            />
          ))}
        </Stack>
      )}
    </Box>
  );
};
