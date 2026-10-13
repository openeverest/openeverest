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

import { ReactNode } from 'react';
import { SxProps, Theme } from '@mui/material';

export interface SegmentedFieldProps {
  // Field controls merged into a single frame. Meant for lean inputs
  // (text / select / number); each child sizes itself via its own sx flex.
  children: ReactNode;
  // Accessible name for the whole group.
  label?: string;
  error?: boolean;
  disabled?: boolean;
  // Single helper/error line rendered under the whole field.
  helperText?: string;
  // Draw vertical dividers between segments (off by default).
  dividers?: boolean;
  sx?: SxProps<Theme>;
}
