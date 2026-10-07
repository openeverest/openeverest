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

import { type Meta, type StoryObj } from '@storybook/react';
import { MenuItem, TextField } from '@mui/material';
import SegmentedField from './segmented-field';

const shrinkLabel = { inputLabel: { shrink: true } };

const segments = [
  <TextField
    key="key"
    label="Key"
    placeholder="Key"
    size="small"
    slotProps={shrinkLabel}
    sx={{ flex: 3 }}
  />,
  <TextField
    key="operator"
    select
    label="Operator"
    size="small"
    defaultValue="In"
    sx={{ width: 130 }}
  >
    <MenuItem value="In">in</MenuItem>
    <MenuItem value="NotIn">not in</MenuItem>
  </TextField>,
  <TextField
    key="values"
    label="Values"
    placeholder="e.g. ssd, nvme"
    size="small"
    slotProps={shrinkLabel}
    sx={{ flex: 3 }}
  />,
];

const meta = {
  title: 'SegmentedField',
  component: SegmentedField,
  args: {
    label: 'Condition',
    dividers: true,
    children: segments,
  },
  parameters: { layout: 'padded' },
} satisfies Meta<typeof SegmentedField>;

export default meta;
type Story = StoryObj<typeof meta>;

// Child fields lose their own frame and labels (kept for screen readers); the
// group shows one border and one focus ring.
export const Default: Story = {};

// One error line for the whole group instead of per-field helpers.
export const WithError: Story = {
  args: {
    error: true,
    helperText: 'Key is required',
  },
};
