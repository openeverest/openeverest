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

import KeyboardArrowDownOutlinedIcon from '@mui/icons-material/KeyboardArrowDownOutlined';
import KeyboardArrowUpOutlined from '@mui/icons-material/KeyboardArrowUpOutlined';
import { Button } from '@mui/material';
import { ExpandableSectionToggleProps } from './expandable-section-toggle.types';
import {
  CARET_HALF_WIDTH_PX,
  CARET_HEIGHT_PX,
} from './expandable-section-toggle.constants';

export const ExpandableSectionToggle = ({
  label,
  open,
  onToggle,
  dataTestId,
}: ExpandableSectionToggleProps) => (
  <Button
    size="small"
    data-testid={dataTestId}
    sx={[
      { mr: 2, position: 'relative' },
      open &&
        ((theme) => ({
          '&::after': {
            content: '""',
            position: 'absolute',
            bottom: `-${CARET_HEIGHT_PX}px`,
            width: '0px',
            height: '0px',
            borderStyle: 'solid',
            borderWidth: `0 ${CARET_HALF_WIDTH_PX}px ${CARET_HEIGHT_PX}px ${CARET_HALF_WIDTH_PX}px`,
            borderColor: `transparent transparent ${theme.palette.surfaces?.elevation0} transparent`,
          },
        })),
    ]}
    onClick={onToggle}
    endIcon={
      open ? <KeyboardArrowUpOutlined /> : <KeyboardArrowDownOutlinedIcon />
    }
  >
    {label}
  </Button>
);
