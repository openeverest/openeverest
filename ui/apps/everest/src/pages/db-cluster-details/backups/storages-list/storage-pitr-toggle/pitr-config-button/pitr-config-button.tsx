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

import { IconButton } from '@mui/material';
import EditOutlinedIcon from '@mui/icons-material/EditOutlined';
import { BlockedTooltip } from '../blocked-tooltip/blocked-tooltip';
import { Messages } from '../storage-pitr-toggle.messages';

interface PitrConfigButtonProps {
  storageName: string;
  disabled: boolean;
  reason?: string;
  onClick: () => void;
}

export const PitrConfigButton = ({
  storageName,
  disabled,
  reason,
  onClick,
}: PitrConfigButtonProps) => (
  <BlockedTooltip reason={reason}>
    <IconButton
      size="small"
      disabled={disabled}
      onClick={onClick}
      aria-label={Messages.configure}
      data-testid={`pitr-configure-${storageName}`}
    >
      <EditOutlinedIcon fontSize="small" />
    </IconButton>
  </BlockedTooltip>
);
