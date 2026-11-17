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

import { ToggleButtonGroupInput, ToggleCard } from '@percona/ui-lib';
import { SxProps, Theme } from '@mui/material';
import {
  AffinityPriority,
  AffinityPriorityValue,
} from 'shared-types/affinity.types';
import { AffinityFormFields } from '../affinity-form.types';

const PriorityToggle = ({ sx }: { sx?: SxProps<Theme> }) => (
  <ToggleButtonGroupInput // TODO needs extra styling to look like FIGMA
    name={AffinityFormFields.priority}
    toggleButtonGroupProps={{
      size: 'small',
      sx: sx ?? {
        height: '30px',
        width: '160px',
        marginTop: '20px',
        alignSelf: 'center',
      },
    }}
  >
    {Object.values(AffinityPriority).map((value) => (
      <ToggleCard
        sx={{ borderRadius: '15px' }}
        value={value}
        data-testid={`toggle-button-${value}`}
        key={value}
      >
        {AffinityPriorityValue[value]}
      </ToggleCard>
    ))}
  </ToggleButtonGroupInput>
);
export default PriorityToggle;
