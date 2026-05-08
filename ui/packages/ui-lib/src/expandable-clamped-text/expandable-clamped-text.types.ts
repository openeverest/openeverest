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

import type { DialogProps as MuiDialogProps } from '@mui/material';
import type { TypographyProps } from '@mui/material/Typography';

export type ExpandableClampedTextProps = {
  value: string;
  /** CSS line clamp when collapsed. Default 2 */
  lineClamp?: number;
  /** Above this line count expanded content opens in a dialog instead of inline. Default 16 */
  inlineMaxLines?: number;
  dataTestId?: string;
  /** Applied to main text Typography (variant, color, etc.) */
  textTypographyProps?: TypographyProps;
  /** Applied to Show more / Show less links */
  linkTypographyProps?: TypographyProps;
  dialogTitle?: string;
  closeDialogLabel?: string;
  /** Forwarded to the ui-lib Dialog wrapper (portal target, maxWidth, etc.) */
  dialogProps?: Omit<MuiDialogProps, 'open' | 'onClose' | 'children'>;
};
