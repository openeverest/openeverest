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

import { IconButton } from '@mui/material';
import { TextField } from '@mui/material';
import UpgradeIcon from '@mui/icons-material/Upgrade';
import { TextFieldProps } from '@mui/material';
import { Controller, useFormContext } from 'react-hook-form';

type FileInputProps = {
  name: string;
  label: string;
  textFieldProps?: TextFieldProps;
  fileInputProps?: React.DetailedHTMLProps<
    React.InputHTMLAttributes<HTMLInputElement>,
    HTMLInputElement
  >;
};

const FileInput = ({
  name,
  label,
  textFieldProps = {},
  fileInputProps = {},
}: FileInputProps) => {
  const { control } = useFormContext();
  return (
    <Controller
      name={name}
      control={control}
      render={({ field, fieldState: { error } }) => (
        <TextField
          label={label}
          {...field}
          {...textFieldProps}
          value={
            field.value && field.value instanceof File ? field.value.name : ''
          }
          type="text"
          size="small"
          error={!!error}
          helperText={error ? error.message : textFieldProps?.helperText}
          slotProps={{
            input: {
              endAdornment: (
                <IconButton component="label">
                  <UpgradeIcon fontSize="medium" />
                  <input
                    style={{ display: 'none' }}
                    type="file"
                    hidden
                    onChange={(event) => {
                      const { files } = event.target;

                      if (files) {
                        const file = files[0];
                        field.onChange(file);
                      }
                    }}
                    {...fileInputProps}
                  />
                </IconButton>
              ),
            },
          }}
        />
      )}
    />
  );
};

export default FileInput;
