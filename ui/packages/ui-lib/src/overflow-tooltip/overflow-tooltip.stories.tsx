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

import { useRef } from 'react';
import { type Meta, type StoryObj } from '@storybook/react';
import { Button, Typography } from '@mui/material';
import { OverflowTooltip } from './overflow-tooltip';

interface OverflowTooltipStoryArgs {
  text: string;
}

const OverflowTooltipStory = ({ text }: OverflowTooltipStoryArgs) => {
  const labelRef = useRef<HTMLElement>(null);

  return (
    <OverflowTooltip title={text} placement="right" arrow measureRef={labelRef}>
      <Button variant="outlined" sx={{ width: 160 }}>
        <Typography ref={labelRef} noWrap>
          {text}
        </Typography>
      </Button>
    </OverflowTooltip>
  );
};

const meta: Meta<OverflowTooltipStoryArgs> = {
  title: 'OverflowTooltip',
  component: OverflowTooltipStory,
  parameters: {
    layout: 'centered',
  },
};

export default meta;
type Story = StoryObj<OverflowTooltipStoryArgs>;

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
