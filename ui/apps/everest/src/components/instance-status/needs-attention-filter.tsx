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
import { type MRT_RowData, type MRT_TableInstance } from 'material-react-table';
import { InstanceTone } from 'shared-types/instance.types';
import { NEEDS_ATTENTION_TONES } from 'utils/instance-status';
import { INSTANCE_STATUS_COLUMN_ID } from './instance-status-columns';
import { Messages } from './instance-status.messages';

interface NeedsAttentionFilterProps<T extends MRT_RowData> {
  table: MRT_TableInstance<T>;
  // How many rows the shortcut would leave, whatever is filtered now.
  count: number;
}

// One click to what a person has to look at: the rows with a warning or an
// error. It sets the same filter the Status column offers.
export const NeedsAttentionFilter = <T extends MRT_RowData>({
  table,
  count,
}: NeedsAttentionFilterProps<T>) => {
  const column = table.getColumn(INSTANCE_STATUS_COLUMN_ID);
  const selected =
    (column.getFilterValue() as InstanceTone[] | undefined) ?? [];
  const active =
    selected.length === NEEDS_ATTENTION_TONES.length &&
    NEEDS_ATTENTION_TONES.every((tone) => selected.includes(tone));

  return (
    <Chip
      label={Messages.needsAttention(count)}
      color={active ? 'warning' : 'default'}
      variant={active ? 'filled' : 'outlined'}
      disabled={count === 0 && !active}
      onClick={() =>
        column.setFilterValue(active ? undefined : [...NEEDS_ATTENTION_TONES])
      }
      aria-pressed={active}
      data-testid="needs-attention-filter"
    />
  );
};
