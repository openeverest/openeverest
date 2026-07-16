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

import { TextField } from '@mui/material';
import { kebabize } from '@percona/utils';
import { Controller, useFormContext } from 'react-hook-form';
import { TextInputProps } from './text.types';

const TextInput = ({
  control,
  name,
  label,
  controllerProps,
  textFieldProps = {},
  isRequired,
  formHelperTextProps = {},
}: TextInputProps) => {
  const { control: contextControl } = useFormContext();
  const { sx: textFieldPropsSx, onChange, ...restFieldProps } = textFieldProps;
  const { sx: formHelperTextPropsSx } = formHelperTextProps;

  return (
    <Controller
      name={name}
      control={control ?? contextControl}
      render={({ field, fieldState: { error } }) => (
        <TextField
          label={label}
          {...field}
          size={restFieldProps?.size || 'small'}
          sx={[
            { mt: 3 },
            ...(Array.isArray(textFieldPropsSx)
              ? textFieldPropsSx
              : [textFieldPropsSx]),
          ]}
          {...restFieldProps}
          variant="outlined"
          required={isRequired}
          error={!!error}
          helperText={error ? error.message : restFieldProps?.helperText}
          slotProps={{
            htmlInput: {
              'data-testid': `text-input-${kebabize(name)}`,
              onWheel: (e: React.WheelEvent<HTMLInputElement>) => {
                (e.target as HTMLElement).blur();
              },
              onBlur: (event: React.FocusEvent<HTMLInputElement>) => {
                if (textFieldProps?.onBlur) {
                  const modifiedEvent = {
                    ...event,
                    target: {
                      ...event.target,
                      value: textFieldProps.onBlur(event),
                    },
                  };
                  field.onChange(modifiedEvent);
                }
              },
              onChange: (event: React.ChangeEvent<HTMLInputElement>) => {
                if (onChange) {
                  const modifiedEvent = {
                    ...event,
                    target: {
                      ...event.target,
                      value: onChange(event),
                    },
                  };
                  field.onChange(modifiedEvent);
                } else {
                  field.onChange(event);
                }
              },
              ...restFieldProps?.inputProps,
            },

            inputLabel: {
              shrink: true,
            },

            formHelperText: {
              sx: formHelperTextPropsSx,
            },
          }}
        />
      )}
      {...controllerProps}
    />
  );
};

export default TextInput;
