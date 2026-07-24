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

// Delay before revealing the trigger button once providers finish loading —
// avoids flashing the skeleton for a single tick when the query resolves fast.
export const REVEAL_DELAY_MS = 300;

export const buttonSx: SxProps<Theme> = {
  display: 'flex',
  minHeight: '34px',
  width: '165px',
};

export const skeletonSx: SxProps<Theme> = {
  ...buttonSx,
  borderRadius: '128px',
};

export const drawerPaperSx: SxProps<Theme> = {
  width: { xs: '100vw', sm: 420 },
  maxWidth: '100vw',
};

export const drawerContainerSx: SxProps<Theme> = {
  display: 'flex',
  flexDirection: 'column',
  height: '100%',
};

export const drawerHeaderSx: SxProps<Theme> = {
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'space-between',
  px: 3,
  py: 2,
  borderBottom: (theme) => `1px solid ${theme.palette.divider}`,
};

export const drawerBannerSx: SxProps<Theme> = {
  px: 3,
  py: 1.5,
  bgcolor: (theme) => theme.palette.action.hover,
  borderBottom: (theme) => `1px solid ${theme.palette.divider}`,
};
