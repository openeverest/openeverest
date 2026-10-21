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
            sx={{
              backgroundColor: (theme) =>
                theme.palette.mode === 'light'
                  ? theme.palette.grey[100]
                  : theme.palette.grey[900],
              borderRadius: '8px',
              padding: '4px',
              border: '1px solid',
              borderColor: 'divider',
              '& .MuiToggleButtonGroup-grouped': {
                margin: '2px',
                border: '0',
                '&.Mui-disabled': {
                  border: '0',
                },
                '&:not(:first-of-type)': {
                  borderRadius: '6px',
                },
                '&:first-of-type': {
                  borderRadius: '6px',
                },
              },
              ...toggleButtonGroupProps?.sx,
            }}
            {...toggleButtonGroupProps}
          >
            {options
              ? options.map((option) => (
                  <ToggleButton
                    key={option.value}
                    value={option.value}
                    data-testid={`toggle-button-${option.value}`}
                    sx={{
                      textTransform: 'none',
                      fontWeight: 500,
                      px: 3,
                      py: 1,
                      transition: 'all 0.2s ease-in-out',
                      '&.Mui-selected': {
                        backgroundColor: 'background.paper',
                        color: 'primary.main',
                        boxShadow: '0px 2px 4px rgba(0, 0, 0, 0.1)',
                        '&:hover': {
                          backgroundColor: 'background.paper',
                        },
                      },
                      '&:hover': {
                        backgroundColor: 'action.hover',
                      },
                    }}
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
