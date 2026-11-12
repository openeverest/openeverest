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

import { type MRT_ColumnDef, type MRT_RowData } from 'material-react-table';
import {
  InstanceAvailability as Availability,
  InstanceStatusDisplay,
  InstanceTone,
} from 'shared-types/instance.types';
import { InstanceConditionEntry } from 'utils/instance-conditions';
import {
  compareInstanceAvailability,
  compareInstanceStatus,
  INSTANCE_AVAILABILITIES,
  INSTANCE_TONES,
  NEEDS_ATTENTION_TONES,
} from 'utils/instance-status';
import { Messages as StatusMessages } from 'utils/instance-status.messages';
import { InstanceAvailability } from './instance-availability';
import { InstanceStatus } from './instance-status';
import { InstanceStatusMarker } from './instance-status-marker';
import { Messages } from './instance-status.messages';

export const INSTANCE_STATUS_COLUMN_ID = 'status';
export const INSTANCE_AVAILABLE_COLUMN_ID = 'available';
export const INSTANCE_NOTICES_COLUMN_ID = 'notices';

// The most urgent rows first.
export const INSTANCE_STATUS_SORTING = [
  { id: INSTANCE_STATUS_COLUMN_ID, desc: false },
];

// What a list needs to know about an Instance to show how it is doing.
export interface InstanceStatusRow {
  name: string;
  status: InstanceStatusDisplay;
  notices: InstanceConditionEntry[];
}

// The columns every list of Instances shows the same way: what is happening
// and whether to act, whether the database is up, and what else deserves a
// look. A row without an Instance leaves them empty.
//
// The lists sort and filter on the tone and on availability, which are short,
// closed sets, and search on the label, which is what the cell says.
export const getInstanceStatusColumns = <T extends MRT_RowData>(
  getRow: (row: T) => InstanceStatusRow | undefined
): MRT_ColumnDef<T>[] => [
  {
    id: INSTANCE_STATUS_COLUMN_ID,
    header: Messages.columns.status,
    accessorFn: (row) => getRow(row)?.status.label ?? '',
    sortingFn: (a, b) => {
      const [first, second] = [getRow(a.original), getRow(b.original)];
      if (!first || !second) {
        return Number(!first) - Number(!second);
      }
      return compareInstanceStatus(first.status, second.status);
    },
    filterVariant: 'multi-select',
    filterSelectOptions: INSTANCE_TONES.map((tone) => ({
      label: StatusMessages.tones[tone],
      value: tone,
    })),
    filterFn: (row, _, tones?: InstanceTone[]) => {
      const tone = getRow(row.original)?.status.tone;
      return !tones?.length || (!!tone && tones.includes(tone));
    },
    size: 240,
    Cell: ({ row }) => {
      const item = getRow(row.original);
      return item ? (
        <InstanceStatus status={item.status} dataTestId={item.name} />
      ) : (
        Messages.none
      );
    },
  },
  {
    id: INSTANCE_AVAILABLE_COLUMN_ID,
    header: Messages.columns.available,
    accessorFn: (row) => getRow(row)?.status.available ?? '',
    sortingFn: (a, b) => {
      const [first, second] = [getRow(a.original), getRow(b.original)];
      if (!first || !second) {
        return Number(!first) - Number(!second);
      }
      return compareInstanceAvailability(
        first.status.available,
        second.status.available
      );
    },
    filterVariant: 'multi-select',
    filterSelectOptions: INSTANCE_AVAILABILITIES.map((available) => ({
      label: Messages.availabilityFilter[available],
      value: available,
    })),
    filterFn: (row, _, selected?: Availability[]) => {
      const available = getRow(row.original)?.status.available;
      return !selected?.length || (!!available && selected.includes(available));
    },
    size: 150,
    Cell: ({ row }) => {
      const item = getRow(row.original);
      return item ? (
        <InstanceAvailability
          available={item.status.available}
          dataTestId={item.name}
        />
      ) : (
        Messages.none
      );
    },
  },
  {
    id: INSTANCE_NOTICES_COLUMN_ID,
    header: Messages.columns.notices,
    accessorFn: (row) => getRow(row)?.notices.length ?? 0,
    enableColumnFilter: false,
    enableGlobalFilter: false,
    size: 120,
    Cell: ({ row }) => {
      const item = getRow(row.original);
      return item ? (
        <InstanceStatusMarker entries={item.notices} dataTestId={item.name} />
      ) : null;
    },
  },
];

export const countNeedingAttention = (
  rows: (Pick<InstanceStatusRow, 'status'> | undefined)[]
) =>
  rows.filter((row) => row && NEEDS_ATTENTION_TONES.includes(row.status.tone))
    .length;
