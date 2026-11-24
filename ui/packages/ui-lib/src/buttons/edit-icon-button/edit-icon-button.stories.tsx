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
import { Typography } from '@mui/material';
import OverviewCard from '../../cards/overview-card';
import { EditIconButton } from './edit-icon-button';

const meta = {
  title: 'EditIconButton',
  component: EditIconButton,
  args: { label: 'Edit monitoring', onClick: () => {} },
  parameters: { layout: 'centered' },
} satisfies Meta<typeof EditIconButton>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Basic: Story = {};

// Its usual place: the action of an overview card header.
export const InCardHeader: Story = {
  render: (args) => (
    <OverviewCard
      dataTestId="monitoring"
      cardHeaderProps={{
        title: 'Monitoring',
        action: <EditIconButton {...args} />,
      }}
    >
      <Typography variant="body2">Not configured</Typography>
    </OverviewCard>
  ),
};
