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
  FormControlLabel,
  RadioGroup as MuiRadioGroup,
  Radio,
} from '@mui/material';
import { Controller, useFormContext } from 'react-hook-form';
import LabeledContent from '../../../labeled-content';
import { RadioGroupProps } from './radio-group.types';

const RadioGroup = ({
  name,
  control,
  label,
  controllerProps,
  labelProps,
  radioGroupFieldProps,
  isRequired = false,
  options,
}: RadioGroupProps) => {
  const { control: contextControl } = useFormContext();
  const content = (
    <Controller
      name={name}
      control={control ?? contextControl}
      render={({ field }) => (
        <MuiRadioGroup
          {...field}
          row
          name={`radio-group-${name}`}
          {...radioGroupFieldProps}
        >
          {options.map((option) => (
            <FormControlLabel
              key={option.label}
              value={option.value}
              control={
                <Radio
                  {...option.radioProps}
                  slotProps={{
                    input: {
                      // @ts-expect-error
                      'data-testid': `radio-option-${option.value}`,
                    },
                  }}
                />
              }
              label={option.label}
              disabled={option.disabled}
            />
          ))}
        </MuiRadioGroup>
      )}
      {...controllerProps}
    />
  );

  return label ? (
    <LabeledContent label={label} isRequired={isRequired} {...labelProps}>
      {content}
    </LabeledContent>
  ) : (
    content
  );
};

export default RadioGroup;
