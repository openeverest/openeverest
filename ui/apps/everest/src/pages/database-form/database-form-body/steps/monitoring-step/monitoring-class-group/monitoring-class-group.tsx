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

import { Box, Stack, Typography } from '@mui/material';
import { SelectableCard } from '@percona/ui-lib';
import { MonitoringSelection } from '../hooks/use-monitoring-selection';
import { Messages } from '../monitoring-step.messages';
import { MonitoringClassGroup as MonitoringClassGroupData } from '../monitoring-step.types';
import {
  getDestinationName,
  getDestinationReadiness,
  getDestinationSummary,
  getMonitoringClassName,
  getMonitoringClassTitle,
} from '../monitoring-step.utils';

interface MonitoringClassGroupProps {
  group: MonitoringClassGroupData;
  selection: MonitoringSelection;
}

const NO_HIGHLIGHT: string[] = [];

export const MonitoringClassGroup = ({
  group: { monitoringClass, destinations },
  selection,
}: MonitoringClassGroupProps) => (
  <Stack
    gap={1.5}
    data-testid={`monitoring-class-${getMonitoringClassName(monitoringClass)}`}
  >
    <Box>
      <Typography variant="sectionHeading">
        {getMonitoringClassTitle(monitoringClass)}
      </Typography>
      {monitoringClass.spec.description && (
        <Typography variant="body2" color="text.secondary">
          {monitoringClass.spec.description}
        </Typography>
      )}
    </Box>
    {destinations.length === 0 ? (
      <Typography variant="helperText" color="text.secondary">
        {Messages.noDestinations(selection.namespace)}
      </Typography>
    ) : (
      <Box
        sx={{
          display: 'grid',
          gridTemplateColumns: 'repeat(auto-fill, minmax(240px, 1fr))',
          gap: 2,
        }}
      >
        {destinations.map((destination) => {
          const name = getDestinationName(destination);
          const disabledReason = selection.getDisabledReason(destination);
          return (
            <SelectableCard
              key={name}
              item={{
                id: name,
                title: name,
                subtitle: getDestinationSummary(destination),
                detail: disabledReason
                  ? Messages.disabledReason[disabledReason]
                  : Messages.readiness[getDestinationReadiness(destination)],
              }}
              tokens={NO_HIGHLIGHT}
              selected={selection.isSelected(destination)}
              disabled={!!disabledReason}
              onSelect={() => selection.toggle(destination)}
            />
          );
        })}
      </Box>
    )}
  </Stack>
);
