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

import { useState } from 'react';
import { Box, Collapse, FormControlLabel, Switch } from '@mui/material';
import { FormCard } from 'components/form-card';
import { SwitchGroupWrapperProps } from './switch-group-wrapper.types';

// Composes the shared FormCard (title left, control right) and reveals its
// children on toggle. Children stay mounted while collapsed so their form
// values and validation persist.
const SwitchGroupWrapper = ({
  label,
  labelCaption,
  switchLabel = '',
  defaultExpanded = false,
  children,
}: SwitchGroupWrapperProps) => {
  const [expanded, setExpanded] = useState(defaultExpanded);

  return (
    <FormCard
      title={label ?? ''}
      description={labelCaption}
      controlComponent={
        <FormControlLabel
          labelPlacement="start"
          label={switchLabel}
          control={
            <Switch
              checked={expanded}
              onChange={(event) => setExpanded(event.target.checked)}
            />
          }
        />
      }
      cardContent={
        <Collapse in={expanded}>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, mt: 1 }}>
            {children}
          </Box>
        </Collapse>
      }
    />
  );
};

export default SwitchGroupWrapper;
