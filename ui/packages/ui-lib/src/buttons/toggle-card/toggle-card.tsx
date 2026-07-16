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

import { ToggleButton } from '@mui/material';
import { ToggleCardProps } from './toggle-card.types';

const ToggleCard = ({ children, sx, ...props }: ToggleCardProps) => {
  return (
    <ToggleButton
      disableRipple
      sx={[
        (theme) => ({
          backgroundColor: 'background.default',
          boxShadow: 4,
          color: theme.palette.text.primary,
          p: 2,
          textTransform: 'none',
          border: 'none',
          ':hover, &.Mui-selected:hover': {
            backgroundColor: 'action.hover',
          },
          '&.Mui-selected': {
            outlineStyle: 'solid',
            outlineWidth: '2px',
            outlineColor: theme.palette.action.outlinedBorder,
            backgroundColor: 'background.default',
          },
          '&.MuiToggleButtonGroup-grouped': {
            '&:not(:last-of-type)': {
              borderTopRightRadius: `${theme.shape.borderRadius}px`,
              borderBottomRightRadius: `${theme.shape.borderRadius}px`,
              [theme.breakpoints.down('sm')]: {
                mb: 1,
              },
              [theme.breakpoints.up('sm')]: {
                mr: 0.5,
              },
            },
            '&:not(:first-of-type)': {
              borderTopLeftRadius: `${theme.shape.borderRadius}px`,
              borderBottomLeftRadius: `${theme.shape.borderRadius}px`,
            },
          },
          '&.MuiButtonBase-root': {
            wordWrap: 'break-word',
            whiteSpace: 'pre-wrap',
          },
        }),
        ...(Array.isArray(sx) ? sx : [sx]),
      ]}
      {...props}
    >
      {children}
    </ToggleButton>
  );
};

export default ToggleCard;
