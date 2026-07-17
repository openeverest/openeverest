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

import { useState } from 'react';
import { TextInput } from '@percona/ui-lib';
import VisibilityOutlinedIcon from '@mui/icons-material/VisibilityOutlined';
import VisibilityOffOutlinedIcon from '@mui/icons-material/VisibilityOffOutlined';
import { HiddenInputProps } from './hidden-input.types';
import { IconButton } from '@mui/material';

export const HiddenInput = ({
  isRequired = true,
  ...textInputProps
}: HiddenInputProps) => {
  const [showKey, setShowKey] = useState(false);
  const { textFieldProps = { sx: {} }, ...restTextInputProps } = textInputProps;
  const { sx, InputProps, ...restTextFieldProps } = textFieldProps;

  return (
    <TextInput
      data-testid={`hidden-input-${name}`}
      textFieldProps={{
        sx: [
          {
            marginTop: 3,
            'input::-ms-reveal': {
              display: 'none',
            },
          },
          ...(Array.isArray(sx) ? sx : [sx]),
        ],
        type: showKey ? 'text' : 'password',
        InputProps: {
          endAdornment: (
            <IconButton onClick={() => setShowKey(!showKey)}>
              {showKey ? (
                <VisibilityOutlinedIcon />
              ) : (
                <VisibilityOffOutlinedIcon />
              )}
            </IconButton>
          ),
          ...InputProps,
        },
        ...restTextFieldProps,
      }}
      isRequired={isRequired}
      {...restTextInputProps}
    />
  );
};
