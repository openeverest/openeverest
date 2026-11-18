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

import { MenuItem, Typography } from '@mui/material';
import { SelectInput } from '@percona/ui-lib';
import { useFormContext, useWatch } from 'react-hook-form';
import { ClassPickerProps } from './class-picker.types';

// The class select, with the picked class's description below it.
export const ClassPicker = ({
  name,
  label,
  classes,
  disabled,
}: ClassPickerProps) => {
  const { control } = useFormContext();
  const className: unknown = useWatch({ control, name });
  const description = classes.find(
    (contract) => contract.name === className
  )?.description;

  return (
    <>
      <SelectInput
        name={name}
        label={label}
        selectFieldProps={{ disabled }}
        isRequired
      >
        {classes.map((contract) => (
          <MenuItem key={contract.name} value={contract.name}>
            {contract.title}
          </MenuItem>
        ))}
      </SelectInput>
      {description && (
        <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
          {description}
        </Typography>
      )}
    </>
  );
};
