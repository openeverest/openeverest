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
import { Box } from '@mui/material';
import { CardPicker } from './card-picker';
import { CardPickerItem } from './card-picker.types';

const lead: CardPickerItem = {
  id: '',
  title: 'Start from scratch',
  subtitle: 'Configure everything yourself',
};

const buildItems = (count: number): CardPickerItem[] =>
  Array.from({ length: count }, (_, i) => ({
    id: `preset-${i + 1}`,
    title: `Preset ${i + 1}`,
    subtitle: i % 2 === 0 ? 'replicaSet' : 'sharded',
    detail: `${3 + (i % 3) * 2} nodes`,
    searchText: `preset ${i + 1} ${i % 2 === 0 ? 'replicaset' : 'sharded'} storageclass local-path`,
  }));

const meta = {
  title: 'CardPicker',
  component: CardPicker,
  parameters: { layout: 'padded' },
} satisfies Meta<typeof CardPicker>;

export default meta;
type Story = StoryObj<typeof meta>;

const Interactive = ({ items }: { items: CardPickerItem[] }) => {
  const [selectedId, setSelectedId] = useState('');
  return (
    <Box sx={{ maxWidth: 900 }}>
      <CardPicker
        items={items}
        leadCard={lead}
        selectedId={selectedId}
        onSelect={setSelectedId}
      />
    </Box>
  );
};

// Everything fits on one row — no Browse tile.
export const Inline: Story = {
  render: () => <Interactive items={buildItems(3)} />,
};

// Overflow: the last slot becomes a Browse tile that opens the searchable modal.
export const WithOverflow: Story = {
  render: () => <Interactive items={buildItems(12)} />,
};

export const Loading: Story = {
  render: () => (
    <Box sx={{ maxWidth: 900 }}>
      <CardPicker items={[]} selectedId="" onSelect={() => {}} loading />
    </Box>
  ),
};
