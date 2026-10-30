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
import { SelectableCard } from './selectable-card';

const item = {
  id: 'pmm-prod',
  title: 'pmm-prod',
  subtitle: 'https://pmm.example.com',
  detail: 'Ready',
};

const meta = {
  title: 'SelectableCard',
  component: SelectableCard,
  args: { item, selected: false, onSelect: () => {} },
  parameters: { layout: 'padded' },
} satisfies Meta<typeof SelectableCard>;

export default meta;
type Story = StoryObj<typeof meta>;

const Toggleable = () => {
  const [selected, setSelected] = useState(false);
  return (
    <Box sx={{ maxWidth: 280 }}>
      <SelectableCard
        item={item}
        selected={selected}
        onSelect={() => setSelected((value) => !value)}
      />
    </Box>
  );
};

// Clicking toggles the selection ring.
export const Basic: Story = {
  render: () => <Toggleable />,
};

// Search tokens are highlighted in the texts.
export const Highlighted: Story = {
  args: { tokens: ['pmm'] },
};

// Inert and dimmed, e.g. when the item can't be picked right now.
export const Disabled: Story = {
  args: {
    item: { ...item, detail: 'Another destination is already selected' },
    disabled: true,
  },
};
