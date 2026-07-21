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

export const rowSx: SxProps<Theme> = {
  display: 'flex',
  flexWrap: 'wrap',
  justifyContent: 'center',
  gap: 1.5,
  width: '100%',
  // Cap the row so buttons wrap onto new lines instead of stretching across
  // the whole empty state on wide screens.
  maxWidth: 560,
};

export const buttonSx: SxProps<Theme> = {
  // Natural, content-based width: buttons sit side by side and wrap when the
  // row runs out of space (never grow to fill it).
  flex: '0 0 auto',
  textTransform: 'none',
};
