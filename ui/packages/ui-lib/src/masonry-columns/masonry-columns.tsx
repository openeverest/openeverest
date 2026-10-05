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

import { Children, isValidElement, useLayoutEffect, useRef } from 'react';
import { Box, useTheme } from '@mui/material';
import { MasonryColumnsProps } from './masonry-columns.types';
import { startMasonryLayout } from './masonry-layout';
import { useColumnCount } from './useColumnCount';

// Masonry of cards with stable placement. Unlike CSS `columns`, which
// rebalances, expanding one item never moves the others to another column.
// Cards may re-balance only until the first click or key press inside.
// All items share one parent, so a column change never remounts them.
export const MasonryColumns = ({
  children,
  columns,
  spacing = 2,
  dataTestId,
}: MasonryColumnsProps) => {
  const theme = useTheme();
  const containerRef = useRef<HTMLDivElement>(null);
  const columnCount = useColumnCount(columns);
  // Assumes theme spacing resolves to px, as with the default theme.
  const gap = parseFloat(theme.spacing(spacing));

  useLayoutEffect(() => {
    if (!containerRef.current) return;
    return startMasonryLayout(containerRef.current, { columnCount, gap });
  }, [columnCount, gap]);

  return (
    // Items are positioned by startMasonryLayout: no CSS flow keeps columns
    // independent and stable at the same time.
    <Box
      ref={containerRef}
      data-testid={dataTestId}
      sx={{ position: 'relative' }}
    >
      {Children.toArray(children).map((child, index) => (
        <Box
          key={isValidElement(child) && child.key !== null ? child.key : index}
          sx={{ position: 'absolute' }}
        >
          {child}
        </Box>
      ))}
    </Box>
  );
};
