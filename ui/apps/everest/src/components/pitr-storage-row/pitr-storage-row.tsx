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

import { ReactNode } from 'react';
import { Box, Switch, Typography } from '@mui/material';
import EditableItem from 'components/editable-item/editable-item';
import { BlockedTooltip } from 'components/blocked-tooltip';
import { Messages } from './pitr-storage-row.messages';

interface PitrStorageRowProps {
  storageName: string;
  checked: boolean;
  onToggle: (checked: boolean) => void;
  toggleDisabled: boolean;
  // When set, the toggle is wrapped in a tooltip explaining why it's blocked.
  toggleReason?: string;
  // Show the configure (pencil) control for provider-specific PITR parameters.
  showConfig: boolean;
  configDisabled: boolean;
  onConfigClick: () => void;
  // Optional secondary line under the storage name (e.g. the recovery window).
  caption?: ReactNode;
}

// Presentational per-storage PITR row shared by the creation wizard and the
// instance Backups tab: storage name (+ optional caption), a configure pencil,
// and a toggle at the row's edge. All behaviour (enable/disable, confirmation,
// config modal, data) lives in the consumers.
export const PitrStorageRow = ({
  storageName,
  checked,
  onToggle,
  toggleDisabled,
  toggleReason,
  showConfig,
  configDisabled,
  onConfigClick,
  caption,
}: PitrStorageRowProps) => (
  <EditableItem
    dataTestId={storageName}
    editButtonProps={
      showConfig
        ? { onClick: onConfigClick, disabled: configDisabled }
        : undefined
    }
    controls={
      <BlockedTooltip reason={toggleReason}>
        <Switch
          size="small"
          checked={checked}
          disabled={toggleDisabled}
          onChange={(_, next) => onToggle(next)}
          data-testid={`pitr-toggle-${storageName}`}
        />
      </BlockedTooltip>
    }
  >
    <Box>
      <Typography variant="body1">
        {Messages.storageLabel(storageName)}
      </Typography>
      {caption && (
        <Typography
          variant="caption"
          color="text.secondary"
          sx={{ display: 'block' }}
        >
          {caption}
        </Typography>
      )}
    </Box>
  </EditableItem>
);
