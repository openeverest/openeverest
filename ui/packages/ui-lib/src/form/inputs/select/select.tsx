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

import {
  CircularProgress,
  FormControl,
  FormHelperText,
  InputLabel,
  MenuItem,
  Select,
} from '@mui/material';
import { kebabize } from '@percona/utils';
import { Controller, useFormContext } from 'react-hook-form';
import { SelectInputProps } from './select.types';
import { Messages } from './select.messages';

const SelectInput = ({
  name,
  control,
  label,
  helperText,
  controllerProps,
  selectFieldProps,
  formControlProps,
  loading,
  children,
}: SelectInputProps) => {
  const { control: contextControl } = useFormContext();

  return (
    <FormControl
      sx={{ mt: 3 }}
      size={formControlProps?.size || 'small'}
      {...formControlProps}
    >
      <InputLabel
        id={`${name}-input-label`}
        shrink={selectFieldProps?.displayEmpty || undefined}
      >
        {label}
      </InputLabel>
      <Controller
        name={name}
        control={control ?? contextControl}
        {...controllerProps}
        render={({ field, fieldState: { error } }) => (
          <>
            <Select
              {...field}
              label={label}
              labelId={`${name}-input-label`}
              variant="outlined"
              error={error !== undefined}
              data-testid={`select-${kebabize(name)}-button`}
              inputProps={{
                'data-testid': `select-input-${kebabize(name)}`,
                ...selectFieldProps?.inputProps,
              }}
              IconComponent={
                loading
                  ? () => (
                      <CircularProgress
                        color="inherit"
                        size={20}
                        sx={{ mr: 1 }}
                      />
                    )
                  : undefined
              }
              {...selectFieldProps}
            >
              {children}
              {(!children || (Array.isArray(children) && !children.length)) && (
                <MenuItem
                  disabled
                  key="noOptions"
                  value=""
                  data-testid="no-options-select"
                  sx={{
                    fontWeight: '400',
                    '&.Mui-disabled.Mui-selected': {
                      backgroundColor: 'transparent',
                    },
                  }}
                >
                  {Messages.noOptions}
                </MenuItem>
              )}
            </Select>
            {(error || helperText) && (
              <FormHelperText error={!!error}>
                {error?.message || helperText}
              </FormHelperText>
            )}
          </>
        )}
      />
    </FormControl>
  );
};

export default SelectInput;
