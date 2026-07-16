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
import { Box, IconButton, IconButtonProps } from '@mui/material';
import * as DocBlock from '@storybook/blocks';
import { CachedOutlined } from '@mui/icons-material';

const meta = {
  title: 'Button (Icon Button)',
  component: IconButton,
  tags: ['autodocs'],
  parameters: {
    layout: 'centered',
    docs: {
      source: {
        code: null,
      },
      page: () => (
        <>
          <DocBlock.Title />
          <DocBlock.Subtitle />
          <DocBlock.Description />
          <DocBlock.Stories />
        </>
      ),
    },
  },
  argTypes: {
    color: {
      options: ['inherit', 'default', 'primary', 'secondary'],
      control: { type: 'inline-radio' },
    },
  },
  render: function Render({ color }) {
    return (
      <Box
        sx={{
          display: 'flex',
          alignItems: 'center',
          gap: '10px',
        }}
      >
        <IconButton color={color} size="large">
          <CachedOutlined />
        </IconButton>
        <IconButton color={color} size="medium">
          <CachedOutlined />
        </IconButton>
        <IconButton color={color} size="small">
          <CachedOutlined />
        </IconButton>
      </Box>
    );
  },
} satisfies Meta<IconButtonProps>;
export default meta;
type Story = StoryObj<IconButtonProps>;
export const Primary: Story = {
  args: {
    color: 'primary',
  },
};
export const Secondary: Story = {
  args: {
    color: 'secondary',
  },
};
