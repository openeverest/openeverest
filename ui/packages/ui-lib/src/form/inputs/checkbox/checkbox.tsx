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

import { Controller, useFormContext } from 'react-hook-form';
import { Checkbox as MUICheckbox } from '@mui/material';
import { CheckboxProps } from './checkbox.types';
import { kebabize } from '@percona/utils';
import LabeledContent from '../../../labeled-content';

const Checkbox = ({
  name,
  label,
  labelProps,
  control,
  controllerProps,
  checkboxProps,
  disabled,
}: CheckboxProps) => {
  const { control: contextControl } = useFormContext();

  const content = (
    <Controller
      name={name}
      control={control ?? contextControl}
      render={({ field }) => (
        <MUICheckbox
          {...field}
          checked={field.value}
          disabled={disabled}
          {...checkboxProps}
          inputProps={{
            // @ts-expect-error
            'data-testid': `checkbox-${kebabize(name)}`,
            ...checkboxProps?.inputProps,
          }}
        />
      )}
      {...controllerProps}
    />
  );

  return label ? (
    <LabeledContent label={label} {...labelProps}>
      {content}
    </LabeledContent>
  ) : (
    content
  );
};

export default Checkbox;
