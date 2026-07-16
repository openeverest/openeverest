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

import { ToggleButton, useTheme } from '@mui/material';
import { ToggleRegularButtonProps } from './toggle-regular-button.types';

const ToggleRegularButton = ({
  children,
  sx,
  dataTestId,
  ...props
}: ToggleRegularButtonProps) => {
  const theme = useTheme();

  return (
    <ToggleButton
      data-testid={dataTestId ?? 'toggle-button-group-btn'}
      disableRipple
      sx={[
        {
          backgroundColor: 'background.default',
          color: theme.palette.text.primary,
          textTransform: 'none',
          borderStyle: 'solid',
          borderColor: theme.palette.primary.main,
          borderWidth: '2px',
          ':hover, &.Mui-selected:hover': {
            backgroundColor: theme.palette.action.hover,
            color: theme.palette.text.primary,
          },
          '&.Mui-selected, &.Mui-selected:hover': {
            backgroundColor: theme.palette.primary.main,
            color: 'background.default',
          },
          '&.MuiButtonBase-root': {
            wordWrap: 'break-word',
            whiteSpace: 'pre-wrap',
          },
        },
        ...(Array.isArray(sx) ? sx : [sx]),
      ]}
      {...props}
    >
      {children}
    </ToggleButton>
  );
};

export default ToggleRegularButton;
