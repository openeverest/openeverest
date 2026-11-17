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
  AffinityOperator,
  AffinityOperatorValue,
  LABEL_SELECTOR_OPERATORS,
} from 'shared-types/affinity.types';
import { AffinityFormFields } from '../affinity-form.types';
import { SelectInput } from '@percona/ui-lib';
import { MenuItem, SxProps, Theme } from '@mui/material';

type Props = {
  disabled: boolean;
  namePrefix?: string;
  sx?: SxProps<Theme>;
  operators?: AffinityOperator[];
};

const OperatorInput = ({
  disabled,
  namePrefix = '',
  sx,
  operators = LABEL_SELECTOR_OPERATORS,
}: Props) => (
  <SelectInput
    name={`${namePrefix}${AffinityFormFields.operator}`}
    label="Operator"
    selectFieldProps={{
      sx: sx ?? { width: '213px' },
      label: 'Operator',
      disabled,
    }}
    data-testid="operator-select"
  >
    {operators.map((value) => (
      <MenuItem key={value} value={value} data-testid={value}>
        {AffinityOperatorValue[value]}
      </MenuItem>
    ))}
  </SelectInput>
);

export default OperatorInput;
