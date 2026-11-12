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
import { DESTINATION_CARDS_GRID_SX } from '../monitoring-destinations-picker.constants';
import { Messages } from '../monitoring-destinations-picker.messages';
import { MonitoringSelection } from '../monitoring-destinations-picker.types';

interface UnavailableEntriesProps {
  selection: MonitoringSelection;
}

export const UnavailableEntries = ({ selection }: UnavailableEntriesProps) => (
  <Stack gap={2} data-testid="monitoring-unavailable-entries">
    <Box>
      <Typography variant="sectionHeading">
        {Messages.unavailableEntries}
      </Typography>
      <Typography variant="body2" color="text.secondary">
        {Messages.unavailableEntriesCaption}
      </Typography>
    </Box>
    <Box sx={DESTINATION_CARDS_GRID_SX}>
      {selection.unavailableEntries.map(
        ({ entry, reason, selected, disabledReason }) => (
          <SelectableCard
            key={entry.name}
            item={{
              id: entry.name,
              title: entry.destinationRef.name,
              subtitle: Messages.unavailableEntryReason[reason],
              detail: disabledReason
                ? Messages.disabledReason[disabledReason]
                : selected
                  ? undefined
                  : Messages.removedOnSave,
            }}
            selected={selected}
            disabled={!!disabledReason}
            onSelect={() => selection.toggleEntry(entry)}
          />
        )
      )}
    </Box>
  </Stack>
);
