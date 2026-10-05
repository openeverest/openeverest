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

import { Breakpoint, useMediaQuery, useTheme } from '@mui/material';
import { MasonryColumnCount } from './masonry-columns.types';

const BREAKPOINTS_LARGEST_FIRST: Breakpoint[] = ['xl', 'lg', 'md', 'sm', 'xs'];

export const useColumnCount = (columns: MasonryColumnCount): number => {
  const theme = useTheme();
  const active: Record<Breakpoint, boolean> = {
    xs: true,
    sm: useMediaQuery(theme.breakpoints.up('sm')),
    md: useMediaQuery(theme.breakpoints.up('md')),
    lg: useMediaQuery(theme.breakpoints.up('lg')),
    xl: useMediaQuery(theme.breakpoints.up('xl')),
  };

  if (typeof columns === 'number') {
    return columns;
  }
  const breakpoint = BREAKPOINTS_LARGEST_FIRST.find(
    (key) => active[key] && columns[key] !== undefined
  );
  return (breakpoint && columns[breakpoint]) || 1;
};
