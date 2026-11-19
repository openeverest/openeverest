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

import { useState } from 'react';
import { type Meta, type StoryObj } from '@storybook/react';
import { Chip } from '@mui/material';
import { type MRT_ColumnDef } from 'material-react-table';
import Table from './table';

interface Fruit {
  name: string;
  ripe: boolean;
}

const FRUITS: Fruit[] = [
  { name: 'Apple', ripe: true },
  { name: 'Banana', ripe: false },
  { name: 'Cherry', ripe: true },
];

const COLUMNS: MRT_ColumnDef<Fruit>[] = [
  { accessorKey: 'name', header: 'Name' },
  {
    id: 'ripe',
    header: 'Ripe',
    accessorFn: ({ ripe }) => (ripe ? 'Yes' : 'No'),
  },
];

const TableWithQuickFilter = () => {
  const [ripeOnly, setRipeOnly] = useState(false);

  return (
    <Table
      tableName="storybook-quick-filter"
      columns={COLUMNS}
      data={ripeOnly ? FRUITS.filter(({ ripe }) => ripe) : FRUITS}
      renderToolbarFilters={() => (
        <Chip
          clickable
          color="success"
          variant={ripeOnly ? 'filled' : 'outlined'}
          label="Ripe only"
          onClick={() => setRipeOnly((value) => !value)}
        />
      )}
    />
  );
};

const meta: Meta = {
  title: 'Table',
  component: TableWithQuickFilter,
};

export default meta;
type Story = StoryObj;

export const WithToolbarFilters: Story = {};
