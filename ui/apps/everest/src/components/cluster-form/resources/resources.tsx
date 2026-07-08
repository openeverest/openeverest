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

import React, { useEffect, useState } from 'react';
import { useFormContext } from 'react-hook-form';
import {
  Accordion,
  Divider,
  FormHelperText,
  Stack,
  Typography,
} from '@mui/material';
import {
  TextInput,
  ToggleRegularButton,
  ToggleButtonGroupInputRegular,
} from '@percona/ui-lib';
import {
  CUSTOM_NR_UNITS_INPUT_VALUE,
  DEFAULT_CONFIG_SERVERS,
  NODES_DEFAULT_SIZES,
  getDefaultNumberOfconfigServersByNumberOfNodes,
  MIN_NUMBER_OF_SHARDS,
  NODES_DB_TYPE_MAP,
  PROXIES_DEFAULT_SIZES,
  resourcesFormSchema,
} from './constants';
import { DbWizardFormFields } from 'consts';
import { DbType } from '@percona/types';

import { Messages } from './messages';
import { z } from 'zod';
import { memoryParser } from 'utils/k8ResourceParser';
import { DbWizardType } from 'pages/database-form/database-form-schema';
import { getProxyUnitNamesFromDbType, someErrorInStateFields } from 'utils/db';
import ResourceRequestsSection from './resource-requests-section';
import ResourcesToggles from './resources-toggles';
import CustomAccordionSummary from './custom-accordion-summary';
import CustomPaper from './custom-paper';

const ResourcesForm = ({
  dbType,
  disableDiskInput,
  allowDiskInputUpdate,
  pairProxiesWithNodes,
  showSharding,
  hideProxies = false,
  disableShardingInput = false,
  defaultValues,
}: {
  dbType: DbType;
  hideProxies?: boolean;
  disableDiskInput?: boolean;
  allowDiskInputUpdate?: boolean;
  pairProxiesWithNodes?: boolean;
  showSharding?: boolean;
  disableShardingInput?: boolean;
  defaultValues?: z.infer<ReturnType<typeof resourcesFormSchema>>;
}) => {
  const [expanded, setExpanded] = useState<'nodes' | 'proxies' | false>(
    'nodes'
  );
  const [showDiskWarning, setShowDiskWarning] = useState(false);
  const {
    watch,
    getFieldState,
    setValue,
    trigger,
    clearErrors,
    getValues,
    formState: { errors },
  } = useFormContext<DbWizardType>();

  const numberOfNodes: string = watch(DbWizardFormFields.numberOfNodes);
  const dbVersion: string = watch(DbWizardFormFields.dbVersion);

  const sharding: boolean = watch(DbWizardFormFields.sharding);
  const shardConfigServers = watch(DbWizardFormFields.shardConfigServers);

  const numberOfProxies: string = watch(DbWizardFormFields.numberOfProxies);
  const customNrOfNodes = watch(DbWizardFormFields.customNrOfNodes);
  const customNrOfProxies = watch(DbWizardFormFields.customNrOfProxies);
  const disk: number = watch(DbWizardFormFields.disk);
  const nodeRequestsSynced =
    watch(DbWizardFormFields.nodeRequestsSynced) !== false;
  const proxyRequestsSynced =
    watch(DbWizardFormFields.proxyRequestsSynced) !== false;
  const proxyUnitNames = getProxyUnitNamesFromDbType(dbType);
  const nodesAccordionSummaryNumber =
    numberOfNodes === CUSTOM_NR_UNITS_INPUT_VALUE
      ? customNrOfNodes
      : numberOfNodes;
  const proxiesAccordionSummaryNumber =
    numberOfProxies === CUSTOM_NR_UNITS_INPUT_VALUE
      ? customNrOfProxies
      : numberOfProxies;

  const someErrorInProxies = someErrorInStateFields(getFieldState, [
    DbWizardFormFields.numberOfProxies,
    DbWizardFormFields.customNrOfProxies,
    DbWizardFormFields.proxyCpu,
    DbWizardFormFields.proxyMemory,
    DbWizardFormFields.proxyCpuRequests,
    DbWizardFormFields.proxyMemoryRequests,
  ]);

  const someErrorInNodes = someErrorInStateFields(getFieldState, [
    DbWizardFormFields.numberOfNodes,
    DbWizardFormFields.customNrOfNodes,
    DbWizardFormFields.cpu,
    DbWizardFormFields.memory,
    DbWizardFormFields.disk,
    DbWizardFormFields.cpuRequests,
    DbWizardFormFields.memoryRequests,
  ]);

  const handleAccordionChange =
    (panel: 'nodes' | 'proxies') =>
    (_: React.SyntheticEvent, newExpanded: boolean) => {
      setExpanded(newExpanded ? panel : false);
    };

  useEffect(() => {
    const { isTouched: numberOfProxiesTouched } = getFieldState(
      DbWizardFormFields.numberOfProxies
    );

    if (!pairProxiesWithNodes || numberOfProxiesTouched) {
      return;
    }

    if (NODES_DB_TYPE_MAP[dbType].includes(numberOfNodes)) {
      setValue(DbWizardFormFields.numberOfProxies, numberOfNodes);
      setValue(DbWizardFormFields.customNrOfProxies, numberOfNodes);
    } else {
      setValue(DbWizardFormFields.numberOfProxies, CUSTOM_NR_UNITS_INPUT_VALUE);
      setValue(DbWizardFormFields.customNrOfProxies, customNrOfNodes);
    }
  }, [
    setValue,
    getFieldState,
    customNrOfNodes,
    dbType,
    numberOfNodes,
    pairProxiesWithNodes,
  ]);

  useEffect(() => {
    const { isTouched: isConfigServersTouched } = getFieldState(
      DbWizardFormFields.shardConfigServers
    );
    const { isTouched: isNumberOfNodesTouched } = getFieldState(
      DbWizardFormFields.numberOfNodes
    );

    if (!isNumberOfNodesTouched) {
      return;
    }

    if (!isConfigServersTouched) {
      if (numberOfNodes && numberOfNodes !== CUSTOM_NR_UNITS_INPUT_VALUE) {
        setValue(
          DbWizardFormFields.shardConfigServers,
          getDefaultNumberOfconfigServersByNumberOfNodes(+numberOfNodes)
        );
      } else {
        clearErrors(DbWizardFormFields.shardConfigServers);
        setValue(
          DbWizardFormFields.shardConfigServers,
          getDefaultNumberOfconfigServersByNumberOfNodes(
            +(customNrOfNodes || '')
          )
        );
      }
    }

    trigger(DbWizardFormFields.shardConfigServers);
  }, [
    setValue,
    getFieldState,
    numberOfNodes,
    customNrOfNodes,
    trigger,
    clearErrors,
  ]);

  useEffect(() => {
    if (defaultValues) {
      const diskUnit = getValues(DbWizardFormFields.diskUnit);
      const defaultDiskUnit = defaultValues.diskUnit;
      const defaultGiValue = memoryParser(
        `${defaultValues.disk}${defaultDiskUnit}`,
        'Gi'
      ).value;
      const newGiValue = memoryParser(`${disk}${diskUnit}`, 'Gi').value;
      setShowDiskWarning(newGiValue > defaultGiValue);
    }
  }, [defaultValues, disk, getValues]);

  useEffect(() => {
    trigger();
  }, [numberOfNodes, customNrOfNodes, trigger, numberOfProxies]);

  useEffect(() => {
    if (someErrorInNodes && !someErrorInProxies) {
      setExpanded('nodes');
    } else if (someErrorInProxies && !someErrorInNodes) {
      setExpanded('proxies');
    }
  }, [expanded, someErrorInNodes, someErrorInProxies]);

  // TODO test the following:
  // when in restore mode, the number of shards should be disabled
  return (
    <>
      {!!showSharding && !!sharding && (
        <CustomPaper sx={{ mb: 2 }}>
          <Typography variant="sectionHeading">
            {' '}
            {Messages.sharding.numberOfShards}
          </Typography>
          <TextInput
            name={DbWizardFormFields.shardNr}
            textFieldProps={{
              disabled: disableShardingInput,
              sx: { maxWidth: '200px' },
              type: 'number',
              required: true,
              inputProps: {
                min: MIN_NUMBER_OF_SHARDS,
              },
            }}
          />
        </CustomPaper>
      )}
      <Accordion
        expanded={expanded === 'nodes'}
        data-testid="nodes-accordion"
        onChange={handleAccordionChange('nodes')}
        sx={{
          px: 2,
          pb: expanded === 'nodes' ? 2 : 0,
        }}
      >
        <CustomAccordionSummary
          unitPlural={sharding ? `Nodes per shard` : 'Nodes'}
          nr={parseInt(nodesAccordionSummaryNumber || '', 10)}
          hasError={someErrorInNodes}
        />
        <Divider />
        <ResourcesToggles
          dbType={dbType}
          dbVersion={dbVersion}
          options={NODES_DB_TYPE_MAP[dbType]}
          sizeOptions={NODES_DEFAULT_SIZES(dbType, dbVersion)}
          resourceSizePerUnitInputName={DbWizardFormFields.resourceSizePerNode}
          cpuInputName={DbWizardFormFields.cpu}
          diskInputName={DbWizardFormFields.disk}
          diskUnitInputName={DbWizardFormFields.diskUnit}
          memoryInputName={DbWizardFormFields.memory}
          numberOfUnitsInputName={DbWizardFormFields.numberOfNodes}
          customNrOfUnitsInputName={DbWizardFormFields.customNrOfNodes}
          disableDiskInput={disableDiskInput}
          allowDiskInputUpdate={allowDiskInputUpdate}
          disableCustom={dbType === DbType.Mysql}
          warnForUpscaling={showDiskWarning}
        />
        <ResourceRequestsSection
          switchName={DbWizardFormFields.nodeRequestsSynced}
          synced={nodeRequestsSynced}
          cpuInputName={DbWizardFormFields.cpuRequests}
          memoryInputName={DbWizardFormFields.memoryRequests}
          cpuLimitName={DbWizardFormFields.cpu}
          memoryLimitName={DbWizardFormFields.memory}
          numberOfUnitsName={DbWizardFormFields.numberOfNodes}
          customNrOfUnitsName={DbWizardFormFields.customNrOfNodes}
          unit="node"
          unitPlural="nodes"
          hasDiskColumn
        />
      </Accordion>
      {!hideProxies && (
        <Accordion
          expanded={expanded === 'proxies'}
          data-testid="proxies-accordion"
          onChange={handleAccordionChange('proxies')}
          sx={{
            px: 2,
            mt: 1,
            pb: expanded === 'proxies' ? 2 : 0,
          }}
        >
          <CustomAccordionSummary
            unitPlural={proxyUnitNames.plural}
            nr={parseInt(proxiesAccordionSummaryNumber || '', 10)}
            hasError={someErrorInProxies}
          />
          <Divider />
          <ResourcesToggles
            dbType={dbType}
            dbVersion={dbVersion}
            unit={proxyUnitNames.singular}
            unitPlural={proxyUnitNames.plural}
            options={NODES_DB_TYPE_MAP[dbType]}
            sizeOptions={PROXIES_DEFAULT_SIZES[dbType]}
            resourceSizePerUnitInputName={
              DbWizardFormFields.resourceSizePerProxy
            }
            cpuInputName={DbWizardFormFields.proxyCpu}
            memoryInputName={DbWizardFormFields.proxyMemory}
            numberOfUnitsInputName={DbWizardFormFields.numberOfProxies}
            customNrOfUnitsInputName={DbWizardFormFields.customNrOfProxies}
          />
          <ResourceRequestsSection
            switchName={DbWizardFormFields.proxyRequestsSynced}
            synced={proxyRequestsSynced}
            cpuInputName={DbWizardFormFields.proxyCpuRequests}
            memoryInputName={DbWizardFormFields.proxyMemoryRequests}
            cpuLimitName={DbWizardFormFields.proxyCpu}
            memoryLimitName={DbWizardFormFields.proxyMemory}
            numberOfUnitsName={DbWizardFormFields.numberOfProxies}
            customNrOfUnitsName={DbWizardFormFields.customNrOfProxies}
            unit={proxyUnitNames.singular}
            unitPlural={proxyUnitNames.plural}
          />
        </Accordion>
      )}
      {!!showSharding && !!sharding && (
        <CustomPaper sx={{ mt: 2 }}>
          <Typography variant="sectionHeading">
            {' '}
            {Messages.sharding.numberOfConfigurationServers}
          </Typography>
          <Stack
            sx={{
              alignItems: 'flex-end',
            }}
          >
            <ToggleButtonGroupInputRegular
              name={DbWizardFormFields.shardConfigServers}
              toggleButtonGroupProps={{
                size: 'small',
                onChange: (_, value) => {
                  setValue(DbWizardFormFields.shardConfigServers, value, {
                    shouldValidate: true,
                  });
                },
              }}
            >
              {DEFAULT_CONFIG_SERVERS.map((number) => (
                <ToggleRegularButton
                  dataTestId={`shard-config-servers-${number}`}
                  sx={{
                    px: 2,
                  }}
                  key={number}
                  value={number}
                  onClick={() => {
                    if (number !== shardConfigServers) {
                      setValue(DbWizardFormFields.shardConfigServers, number, {
                        shouldValidate: true,
                      });
                    }
                  }}
                >
                  {number}
                </ToggleRegularButton>
              ))}
            </ToggleButtonGroupInputRegular>
            {errors.shardConfigServers && (
              <FormHelperText
                data-testid="shard-config-servers-error"
                error={true}
              >
                {errors.shardConfigServers.message}
              </FormHelperText>
            )}
          </Stack>
        </CustomPaper>
      )}
    </>
  );
};

export default ResourcesForm;
