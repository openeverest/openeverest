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

import type { DialogProps as UiDialogProps } from '../dialog/dialog.types';
import type { TypographyProps } from '@mui/material/Typography';

type TextStyleProps = Pick<TypographyProps, 'variant' | 'color' | 'sx'>;

export type ExpandableClampedTextExpandStrategy =
  | {
      type: 'inline';
      dialogTitle?: string;
      autoEscalateToDialog?: boolean;
    }
  | {
      type: 'dialog';
      dialogTitle?: string;
      closeDialogLabel?: string;
      dialogProps?: Omit<UiDialogProps, 'open' | 'onClose' | 'children'>;
    }
  | {
      type: 'navigate';
      onExpand: () => void;
    }
  | {
      type: 'scroll';
      maxHeight?: number;
    };

export type ExpandableClampedTextProps = {
  value: string;
  /** CSS line clamp when collapsed. Default 2 */
  lineClamp?: number;
  dataTestId?: string;
  /** Expansion behavior. Default is inline expand/collapse. */
  expandStrategy?: ExpandableClampedTextExpandStrategy;
  /** Applied to main text Typography (variant, color, etc.) */
  textTypographyProps?: TextStyleProps;
  /** Applied to Show more / Show less links */
  linkTypographyProps?: TextStyleProps;
};
