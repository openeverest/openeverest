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
import { Grid, Typography } from '@mui/material';
import { KeyValueSummary } from './key-value-summary';

const meta = {
  title: 'KeyValueSummary',
  component: KeyValueSummary,
  args: {
    label: 'Load balancer annotations',
    entries: [
      {
        key: 'service.beta.kubernetes.io/aws-load-balancer-type',
        value: 'nlb',
      },
      {
        key: 'service.beta.kubernetes.io/aws-load-balancer-internal',
        value: 'true',
      },
      { key: 'team', value: 'db' },
    ],
  },
  parameters: { layout: 'padded' },
} satisfies Meta<typeof KeyValueSummary>;

export default meta;
type Story = StoryObj<typeof meta>;

// The chip sits where a scalar value would; the table is one click away.
export const InAFieldRow: Story = {
  render: (args) => (
    <Grid container alignItems="center" sx={{ maxWidth: 620 }}>
      <Grid size={6}>
        <Typography variant="body2" sx={{ fontWeight: '700' }}>
          {args.label}
        </Typography>
      </Grid>
      <Grid size="grow">
        <KeyValueSummary {...args} />
      </Grid>
    </Grid>
  ),
};

export const SingleEntry: Story = {
  args: {
    entries: [{ key: 'team', value: 'db' }],
  },
};
