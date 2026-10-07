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
import { Button, Card, CardContent, Typography } from '@mui/material';
import { MasonryColumns } from './masonry-columns';

const ExpandableCard = ({ title, lines }: { title: string; lines: number }) => {
  const [expanded, setExpanded] = useState(false);
  return (
    <Card variant="outlined">
      <CardContent>
        <Typography variant="h6">{title}</Typography>
        {Array.from({ length: expanded ? lines * 4 : lines }, (_, i) => (
          <Typography key={i} variant="body2">
            Line {i + 1}
          </Typography>
        ))}
        <Button size="small" onClick={() => setExpanded((value) => !value)}>
          {expanded ? 'Collapse' : 'Expand'}
        </Button>
      </CardContent>
    </Card>
  );
};

const cards: [string, number][] = [
  ['Database details', 6],
  ['Version', 1],
  ['Resources', 3],
  ['Monitoring', 1],
  ['Advanced configuration', 4],
  ['Backups', 2],
];

const meta = {
  title: 'MasonryColumns',
  component: MasonryColumns,
  args: {
    minColumnWidth: 280,
    maxColumns: 3,
    children: cards.map(([title, lines]) => (
      <ExpandableCard key={title} title={title} lines={lines} />
    )),
  },
  parameters: { layout: 'padded' },
} satisfies Meta<typeof MasonryColumns>;

export default meta;
type Story = StoryObj<typeof meta>;

// Expand any card: the others stay in their columns.
export const Basic: Story = {};

export const NarrowContainer: Story = {
  decorators: [
    (Story) => (
      <div style={{ width: 600 }}>
        <Story />
      </div>
    ),
  ],
};
