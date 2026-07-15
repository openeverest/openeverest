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

import type { SxProps, Theme } from '@mui/material';

export const gridSx: SxProps<Theme> = {
  display: 'flex',
  flexWrap: 'wrap',
  justifyContent: 'center',
  gap: 2,
  width: '100%',
  maxWidth: 900,
};

export const cardSx: SxProps<Theme> = {
  flex: '1 1 200px',
  minWidth: 200,
  maxWidth: 220,
  display: 'flex',
  borderRadius: 2,
  border: '1px solid',
  borderColor: (theme) => theme.palette.dividers?.divider,
  backgroundColor: (theme) => theme.palette.surfaces?.elevation1,
  transition: 'border-color 0.15s, background-color 0.15s',
  '&:hover': {
    borderColor: (theme) => theme.palette.dividers?.dividerStrong,
    backgroundColor: (theme) => theme.palette.surfaces?.elevation0,
  },
};

export const cardActionAreaSx: SxProps<Theme> = {
  height: '100%',
  display: 'flex',
};

export const cardContentSx: SxProps<Theme> = {
  flexGrow: 1,
  display: 'flex',
  flexDirection: 'row',
  alignItems: 'center',
  gap: 1.5,
  py: 2,
  px: 2,
};
