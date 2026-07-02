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
import {
  Alert,
  Box,
  FormGroup,
  FormHelperText,
  Stack,
  Tooltip,
} from '@mui/material';
import { TextInput, ToggleButtonGroupInput, ToggleCard } from '@percona/ui-lib';
import { DbType } from '@percona/types';
import { useFormContext } from 'react-hook-form';
import { useKubernetesClusterResourcesInfo } from 'hooks/api/kubernetesClusters/useKubernetesClusterResourcesInfo';
import { useActiveBreakpoint } from 'hooks/utils/useActiveBreakpoint';
import { CUSTOM_NR_UNITS_INPUT_VALUE, ResourceSize } from '../constants';
import { Messages } from '../messages';
import {
  checkResourceText,
  humanizeResourceSizeMap,
  isVersion84x,
} from '../utils';
import ResourceInput from '../resource-input';
import { ResourcesTogglesProps } from './resources-toggles.types';

const ResourcesToggles = ({
  dbType,
  dbVersion,
  unit = 'node',
  unitPlural = `${unit}s`,
  options,
  sizeOptions,
  resourceSizePerUnitInputName,
  cpuInputName,
  diskInputName = '',
  diskUnitInputName = '',
  memoryInputName,
  numberOfUnitsInputName,
  customNrOfUnitsInputName,
  disableDiskInput,
  allowDiskInputUpdate,
  disableCustom = false,
  warnForUpscaling = false,
}: ResourcesTogglesProps) => {
  const { isMobile, isDesktop } = useActiveBreakpoint();
  const { data: resourcesInfo, isFetching: resourcesInfoLoading } =
    useKubernetesClusterResourcesInfo();
  const { watch, setValue, setError, clearErrors, getFieldState, resetField } =
    useFormContext();

  const resourceSizePerUnit: ResourceSize = watch(resourceSizePerUnitInputName);
  const cpu: number = watch(cpuInputName);
  const memory: number = watch(memoryInputName);
  const disk: number = watch(diskInputName);
  const diskUnit: string = watch(diskUnitInputName);
  const numberOfUnits: string = watch(numberOfUnitsInputName);
  const customNrOfUnits: string = watch(customNrOfUnitsInputName);
  const intNumberOfUnits = parseInt(
    numberOfUnits === CUSTOM_NR_UNITS_INPUT_VALUE
      ? customNrOfUnits
      : numberOfUnits,
    10
  );
  const { error: numberOfUnitsInpuError } = getFieldState(
    numberOfUnitsInputName
  );

  const cpuCapacityExceeded = resourcesInfo
    ? cpu * 1000 > resourcesInfo?.available.cpuMillis
    : !resourcesInfoLoading;
  const memoryCapacityExceeded = resourcesInfo
    ? memory * 1000 ** 3 > resourcesInfo?.available.memoryBytes
    : !resourcesInfoLoading;
  const diskCapacityExceeded = resourcesInfo?.available?.diskSize
    ? disk * 1000 ** 3 > resourcesInfo?.available.diskSize
    : false;

  useEffect(() => {
    if (resourceSizePerUnit && resourceSizePerUnit !== ResourceSize.custom) {
      setValue(cpuInputName, sizeOptions[resourceSizePerUnit].cpu);
      if (allowDiskInputUpdate) {
        setValue(diskInputName, sizeOptions[resourceSizePerUnit].disk);
      }
      setValue(memoryInputName, sizeOptions[resourceSizePerUnit].memory);
    }
  }, [resourceSizePerUnit, allowDiskInputUpdate, setValue]);

  useEffect(() => {
    if (diskCapacityExceeded) {
      setError(diskInputName, { type: 'custom' });
    } else {
      clearErrors(diskInputName);
    }
  }, [diskCapacityExceeded, clearErrors, setError]);

  useEffect(() => {
    if (
      resourceSizePerUnit !== ResourceSize.custom &&
      cpu !== sizeOptions[resourceSizePerUnit].cpu
    ) {
      setValue(resourceSizePerUnitInputName, ResourceSize.custom);
    }
  }, [cpu, setValue]);

  useEffect(() => {
    if (
      allowDiskInputUpdate &&
      resourceSizePerUnit !== ResourceSize.custom &&
      disk !== sizeOptions[resourceSizePerUnit].disk
    ) {
      setValue(resourceSizePerUnitInputName, ResourceSize.custom);
    }
  }, [disk, allowDiskInputUpdate, setValue]);

  useEffect(() => {
    // for MySQL 8.4.0 with 3GB memory, the default resource size should be small
    const isMySQLSpecialMemory =
      dbType === DbType.Mysql && isVersion84x(dbVersion) && memory === 3;

    if (resourceSizePerUnit !== ResourceSize.custom) {
      const expectedMemory = sizeOptions[resourceSizePerUnit].memory;

      if (
        memory !== expectedMemory &&
        (!isMySQLSpecialMemory || expectedMemory !== 3)
      ) {
        setValue(resourceSizePerUnitInputName, ResourceSize.custom);
      }
    } else if (isMySQLSpecialMemory) {
      setValue(resourceSizePerUnitInputName, ResourceSize.small);
    }
  }, [memory, setValue]);

  return (
    <FormGroup sx={{ mt: 3 }}>
      <Stack position="relative">
        <ToggleButtonGroupInput
          name={numberOfUnitsInputName}
          label={`Number of ${unitPlural}`}
          toggleButtonGroupProps={{
            onChange: (_, value) => {
              if (value !== CUSTOM_NR_UNITS_INPUT_VALUE) {
                resetField(customNrOfUnitsInputName, {
                  keepError: true,
                });
              }
            },
          }}
        >
          {options.map((value) => (
            <ToggleCard
              value={value}
              data-testid={`toggle-button-${unitPlural}-${value}`}
              key={value}
            >
              {`${value} ${+value > 1 ? unitPlural : unit}`}
            </ToggleCard>
          ))}
          {!disableCustom && (
            <ToggleCard
              value={CUSTOM_NR_UNITS_INPUT_VALUE}
              data-testid={`toggle-button-${unitPlural}-custom`}
            >
              {Messages.customValue}
            </ToggleCard>
          )}
        </ToggleButtonGroupInput>
        {!!numberOfUnitsInpuError && (
          <FormHelperText
            error
            sx={{
              position: 'absolute',
              bottom: (theme) =>
                theme.spacing(
                  numberOfUnits === CUSTOM_NR_UNITS_INPUT_VALUE ? 4.5 : -1.5
                ),
            }}
          >
            {numberOfUnitsInpuError.message}
          </FormHelperText>
        )}
        {numberOfUnits === CUSTOM_NR_UNITS_INPUT_VALUE && (
          <TextInput
            name={customNrOfUnitsInputName}
            textFieldProps={{
              type: 'number',
              inputProps: {
                step: dbType !== DbType.Mongo ? 1 : 2,
                min: 1,
              },
              sx: {
                width: `${100 / (options.length + 1)}%`,
                alignSelf: 'flex-end',
                mt: 1,
                maxHeight: '50px',
              },
            }}
          />
        )}
      </Stack>
      <ToggleButtonGroupInput
        name={resourceSizePerUnitInputName}
        label={`Resource size per ${unit}`}
      >
        <ToggleCard
          value={ResourceSize.small}
          data-testid={`${unit}-resources-toggle-button-small`}
        >
          {humanizeResourceSizeMap(ResourceSize.small)}
        </ToggleCard>
        <ToggleCard
          value={ResourceSize.medium}
          data-testid={`${unit}-resources-toggle-button-medium`}
        >
          {humanizeResourceSizeMap(ResourceSize.medium)}
        </ToggleCard>
        <ToggleCard
          value={ResourceSize.large}
          data-testid={`${unit}-resources-toggle-button-large`}
        >
          {humanizeResourceSizeMap(ResourceSize.large)}
        </ToggleCard>
        <ToggleCard
          value={ResourceSize.custom}
          data-testid={`${unit}-resources-toggle-button-custom`}
        >
          {humanizeResourceSizeMap(ResourceSize.custom)}
        </ToggleCard>
      </ToggleButtonGroupInput>
      <Box
        sx={{
          display: 'flex',
          flexDirection: isMobile ? 'column' : 'row',
          justifyContent: 'center',
          gap: isDesktop ? 3 : 2,
          mt: 3,
          '& > *': {
            flex: '1 0 0 ',
          },
        }}
      >
        <ResourceInput
          unit={unit}
          unitPlural={unitPlural}
          name={cpuInputName}
          label="CPU limit"
          helperText={checkResourceText(
            resourcesInfo?.available?.cpuMillis,
            'CPU',
            'cpu',
            cpuCapacityExceeded
          )}
          endSuffix="CPU"
          numberOfUnits={intNumberOfUnits}
        />
        <ResourceInput
          unit={unit}
          unitPlural={unitPlural}
          name={memoryInputName}
          label="MEMORY limit"
          helperText={checkResourceText(
            resourcesInfo?.available?.memoryBytes,
            'GB',
            'memory',
            memoryCapacityExceeded
          )}
          endSuffix="GB"
          numberOfUnits={intNumberOfUnits}
        />

        {diskInputName && (
          <Tooltip
            title={disableDiskInput ? Messages.disabledDiskInputTooltip : ''}
            placement="top"
            arrow
            data-testid="disk-tooltip"
          >
            <Box>
              <ResourceInput
                unit={unit}
                unitPlural={unitPlural}
                name={diskInputName}
                disabled={disableDiskInput}
                label="DISK"
                helperText={checkResourceText(
                  resourcesInfo?.available?.diskSize,
                  diskUnit,
                  'disk',
                  diskCapacityExceeded
                )}
                endSuffix={diskUnit}
                numberOfUnits={intNumberOfUnits}
              />
            </Box>
          </Tooltip>
        )}
      </Box>
      {warnForUpscaling && (
        <Alert sx={{ mt: 2 }} severity="warning">
          {Messages.upscalingDiskWarning}
        </Alert>
      )}
    </FormGroup>
  );
};

export default ResourcesToggles;
