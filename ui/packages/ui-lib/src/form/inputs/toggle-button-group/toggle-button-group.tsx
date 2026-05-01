// everest
// Copyright (C) 2023 Percona LLC
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

import { ToggleButton, ToggleButtonGroup } from '@mui/material';
import { Controller, useFormContext } from 'react-hook-form';
import { LabeledContent } from '../../../labeled-content';
import { ToggleButtonGroupInputProps } from './toggle-button-group.types';

export const ToggleButtonGroupInput = ({
  control,
  name,
  label,
  controllerProps,
  toggleButtonGroupProps,
  options,
  children,
}: ToggleButtonGroupInputProps) => {
  const { control: contextControl } = useFormContext();

  return (
    <Controller
      name={name}
      control={control || contextControl}
      {...controllerProps}
      render={({ field, fieldState: { error } }) => (
        <LabeledContent
          label={label}
          error={!!error}
          helperText={error?.message}
        >
          <ToggleButtonGroup
            {...field}
            exclusive
            onChange={(_, value) => {
              if (value !== null) {
                field.onChange(value);
              }
            }}
            {...toggleButtonGroupProps}
          >
            {options
              ? options.map((option) => (
                  <ToggleButton
                    key={option.value}
                    value={option.value}
                    data-testid={`toggle-button-${option.value}`}
                  >
                    {option.label}
                  </ToggleButton>
                ))
              : children}
          </ToggleButtonGroup>
        </LabeledContent>
      )}
    />
  );
};
