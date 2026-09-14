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
  FormControl,
  FormControlLabel,
  FormHelperText,
  Switch,
  Typography,
} from '@mui/material';
import { kebabize } from '@percona/utils';
import { useController, useFormContext } from 'react-hook-form';
import { SwitchInputProps } from './switch.types';

const SwitchInput = ({
  name,
  control,
  label,
  labelCaption,
  error,
  helperText,
  controllerProps,
  formControlLabelProps,
  formControlProps,
  switchFieldProps = {},
}: SwitchInputProps) => {
  const { control: contextControl } = useFormContext();
  const { onChange, ...restSwitchFieldProps } = switchFieldProps;
  const { field, fieldState } = useController({
    name,
    control: control ?? contextControl,
    ...controllerProps,
  });

  const hasError = !!fieldState.error || !!error;
  const helperMessage = fieldState.error?.message ?? helperText;

  const switchControl = (
    <Switch
      {...field}
      onChange={(e) => {
        onChange?.(e, e.target.checked);
        field.onChange(e);
      }}
      sx={[
        !!labelCaption && {
          alignSelf: 'flex-start',
          mt: -1,
        },
      ]}
      checked={field.value}
      data-testid={`switch-input-${kebabize(name)}`}
      {...restSwitchFieldProps}
    />
  );

  const labelElement = (
    <FormControlLabel
      label={
        <>
          <Typography variant="body1" component="span">
            {label}
          </Typography>
          {labelCaption && (
            <Typography variant="caption">{labelCaption}</Typography>
          )}
        </>
      }
      data-testid={`switch-input-${kebabize(name)}-label`}
      control={switchControl}
      {...formControlLabelProps}
    />
  );

  const helperTextElement =
    hasError || helperMessage ? (
      <FormHelperText error={hasError} sx={{ mx: 0 }}>
        {helperMessage}
      </FormHelperText>
    ) : null;

  if (formControlProps || helperTextElement) {
    const { sx: formControlSx, ...restFormControlProps } =
      formControlProps ?? {};

    return (
      <FormControl
        error={hasError}
        sx={formControlSx}
        {...restFormControlProps}
      >
        {labelElement}
        {helperTextElement}
      </FormControl>
    );
  }

  return labelElement;
};

export default SwitchInput;
