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

import { Chip } from '@mui/material';
import WarningAmberOutlinedIcon from '@mui/icons-material/WarningAmberOutlined';
import { Messages } from './needs-attention-chip.messages';

interface NeedsAttentionChipProps {
  count: number;
  active: boolean;
  onToggle: () => void;
}

export const NeedsAttentionChip = ({
  count,
  active,
  onToggle,
}: NeedsAttentionChipProps) =>
  count > 0 ? (
    <Chip
      clickable
      color="warning"
      variant={active ? 'filled' : 'outlined'}
      icon={<WarningAmberOutlinedIcon />}
      label={Messages.label(count)}
      onClick={onToggle}
      aria-pressed={active}
      data-testid="needs-attention-chip"
    />
  ) : null;
