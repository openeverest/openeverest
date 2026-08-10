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

import { useEffect } from 'react';
import { Box, Stack, Tooltip } from '@mui/material';
import InfoOutlinedIcon from '@mui/icons-material/InfoOutlined';
import { SwitchInput } from '@percona/ui-lib';
import { useFormContext } from 'react-hook-form';
import { useActiveBreakpoint } from 'hooks/utils/useActiveBreakpoint';
import { CUSTOM_NR_UNITS_INPUT_VALUE } from '../constants';
import { Messages } from '../messages';
import ResourceInput from '../resource-input';
import { ResourceRequestsSectionProps } from './resource-requests-section.types';

const ResourceRequestsSection = ({
  switchName,
  synced,
  cpuInputName,
  memoryInputName,
  cpuLimitName,
  memoryLimitName,
  numberOfUnitsName,
  customNrOfUnitsName,
  unit,
  unitPlural,
  hasDiskColumn = false,
}: ResourceRequestsSectionProps) => {
  const { watch, setValue } = useFormContext();
  const { isDesktop } = useActiveBreakpoint();
  const cpuLimit: number = watch(cpuLimitName);
  const memoryLimit: number = watch(memoryLimitName);
  const numberOfUnits: string = watch(numberOfUnitsName);
  const customNrOfUnits: string = watch(customNrOfUnitsName);

  let intNumberOfUnits = parseInt(
    numberOfUnits === CUSTOM_NR_UNITS_INPUT_VALUE
      ? customNrOfUnits
      : numberOfUnits,
    10
  );
  if (Number.isNaN(intNumberOfUnits) || intNumberOfUnits < 1) {
    intNumberOfUnits = 1;
  }

  useEffect(() => {
    if (synced) {
      setValue(cpuInputName, cpuLimit, { shouldValidate: false });
      setValue(memoryInputName, memoryLimit, { shouldValidate: false });
    }
  }, [synced, cpuLimit, memoryLimit, cpuInputName, memoryInputName, setValue]);

  return (
    <Box sx={{ mt: 3 }}>
      <Stack spacing={1}>
        <Stack direction="row" alignItems="center" spacing={0.5}>
          <SwitchInput label={Messages.syncWithLimits} name={switchName} />
          <Tooltip title={Messages.requestsTooltip} placement="right" arrow>
            <InfoOutlinedIcon
              sx={{
                width: '18px',
                height: '18px',
                color: 'text.secondary',
              }}
            />
          </Tooltip>
        </Stack>
        {!synced && (
          <Box
            sx={{
              display: 'flex',
              flexDirection: 'row',
              gap: isDesktop ? 3 : 2,
              '& > *': {
                flex: '1 0 0',
              },
            }}
          >
            <ResourceInput
              unit={unit}
              unitPlural={unitPlural}
              name={cpuInputName}
              label="CPU request"
              helperText={Messages.requestCeiling(cpuLimit, 'CPU')}
              endSuffix="CPU"
              numberOfUnits={intNumberOfUnits}
              disableTopMargin
            />
            <ResourceInput
              unit={unit}
              unitPlural={unitPlural}
              name={memoryInputName}
              label="Memory request"
              helperText={Messages.requestCeiling(memoryLimit, 'GB')}
              endSuffix="GB"
              numberOfUnits={intNumberOfUnits}
              disableTopMargin
            />
            {/* When the limits above have a DISK column, an empty third slot
                keeps the request fields aligned with CPU/MEMORY. */}
            {hasDiskColumn && <Box aria-hidden />}
          </Box>
        )}
      </Stack>
    </Box>
  );
};

export default ResourceRequestsSection;
