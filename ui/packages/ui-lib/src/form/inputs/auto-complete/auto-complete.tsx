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
  Autocomplete,
  CircularProgress,
  TextField,
  Tooltip,
} from '@mui/material';
import { kebabize } from '@percona/utils';
import { Controller, useFormContext } from 'react-hook-form';
import { AutoCompleteInputProps } from './auto-complete.types';

function AutoCompleteInput<T>({
  name,
  control,
  controllerProps,
  label,
  autoCompleteProps = {},
  textFieldProps = {},
  options,
  loading = false,
  isRequired = false,
  disabled = false,
  tooltipText,
  onChange,
}: AutoCompleteInputProps<T>) {
  const { control: contextControl } = useFormContext();
  const { helperText, ...restTextFieldProps } = textFieldProps;
  const { sx, ...restAutocompleteProps } = autoCompleteProps;

  return (
    <Controller
      name={name}
      control={control ?? contextControl}
      render={({ field, fieldState: { error } }) => (
        <Tooltip title={tooltipText} placement="top" arrow>
          <Autocomplete
            {...field}
            sx={[{ mt: 3 }, ...(Array.isArray(sx) ? sx : [sx])]}
            options={options}
            forcePopupIcon
            disabled={disabled}
            onChange={(_, newValue) => {
              field.onChange(newValue);
              if (onChange) {
                onChange();
              }
            }}
            data-testid={`${kebabize(name)}-autocomplete`}
            // We might generalize this in the future, if we think renderInput should be defined from the outside
            renderInput={(params) => (
              <TextField
                {...params}
                error={!!error}
                label={label}
                helperText={error ? error.message : helperText}
                size="small"
                required={isRequired}
                {...restTextFieldProps}
                slotProps={{
                  input: {
                    ...params.InputProps,
                    endAdornment: (
                      <>
                        {loading ? (
                          <CircularProgress color="inherit" size={20} />
                        ) : null}
                        {params.InputProps.endAdornment}
                      </>
                    ),
                  },

                  htmlInput: {
                    'data-testid': `text-input-${kebabize(name)}`,
                    ...params.inputProps,
                    ...textFieldProps?.inputProps,
                  },
                }}
              />
            )}
            {...restAutocompleteProps}
          />
        </Tooltip>
      )}
      {...controllerProps}
    />
  );
}

export default AutoCompleteInput;
