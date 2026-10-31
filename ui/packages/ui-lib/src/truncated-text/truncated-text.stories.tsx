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
import { Box } from '@mui/material';
import TruncatedText from './truncated-text';
import { type TruncatedTextProps } from './truncated-text.types';

const meta: Meta<TruncatedTextProps> = {
  title: 'TruncatedText',
  component: TruncatedText,
  parameters: {
    layout: 'centered',
  },
  decorators: [
    (Story) => (
      <Box sx={{ width: 160 }}>
        <Story />
      </Box>
    ),
  ],
};

export default meta;
type Story = StoryObj<TruncatedTextProps>;

export const Overflowing: Story = {
  args: {
    text: 'A very long plugin name that does not fit',
  },
};

export const Fitting: Story = {
  args: {
    text: 'Short name',
  },
};
