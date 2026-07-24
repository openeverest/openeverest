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

export const listSx: SxProps<Theme> = {
  flexGrow: 1,
  overflowY: 'auto',
  p: 2,
  display: 'flex',
  flexDirection: 'column',
  gap: 1,
};

export const rowSx: SxProps<Theme> = {
  display: 'flex',
  alignItems: 'center',
  gap: 1.5,
  px: 2,
  py: 1.25,
  borderRadius: 1,
  border: (theme) => `1px solid ${theme.palette.divider}`,
  textDecoration: 'none',
  color: 'inherit',
  transition: 'border-color 0.15s, background-color 0.15s',
  '&:hover': {
    borderColor: (theme) => theme.palette.primary.main,
    backgroundColor: (theme) => theme.palette.action.hover,
  },
};
