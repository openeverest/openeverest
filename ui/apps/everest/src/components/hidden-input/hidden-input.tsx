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
