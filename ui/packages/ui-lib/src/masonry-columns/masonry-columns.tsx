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

import { Children } from 'react';
import { Box, Stack } from '@mui/material';
import { MasonryColumnsProps } from './masonry-columns.types';
import { distributeIntoColumns } from './masonry-columns.utils';
import { useColumnCount } from './useColumnCount';

// Masonry of cards with stable placement. Unlike CSS `columns`, which
// rebalances, expanding one item never moves the others to another column.
export const MasonryColumns = ({
  children,
  columns,
  spacing = 2,
  dataTestId,
}: MasonryColumnsProps) => {
  const count = useColumnCount(columns);
  const items = Children.toArray(children);

  return (
    <Box
      data-testid={dataTestId}
      sx={{ display: 'flex', alignItems: 'flex-start', gap: spacing }}
    >
      {distributeIntoColumns(items, count).map((columnItems, index) => (
        <Stack
          key={index}
          data-testid="masonry-column"
          sx={{ flex: 1, minWidth: 0, gap: spacing }}
        >
          {columnItems}
        </Stack>
      ))}
    </Box>
  );
};

export default MasonryColumns;
