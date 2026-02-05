// everest
// Copyright (C) 2023 Percona LLC
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
  AutoCompleteInput,
  SelectInput,
  SwitchInput,
  TextArray,
  TextInput,
} from '@percona/ui-lib';
import { Messages } from './messages';
import {
  AdvancedConfigurationFields,
  AllowedFieldsToInitiallyLoadDefaults,
  PROXY_EXPOSE_TYPE_TO_LABEL,
} from './advanced-configuration.types';
import { ProxyExposeType } from 'shared-types/dbCluster.types';
import { useFormContext } from 'react-hook-form';
import { DbEngineType, DbType } from '@percona/types';
import { getParamsPlaceholderFromDbType } from './advanced-configuration.utils';
import {
  Box,
  FormGroup,
  IconButton,
  MenuItem,
  Stack,
  Tooltip,
  Typography,
} from '@mui/material';
import InfoIcon from '@mui/icons-material/InfoOutlined';
import { useKubernetesClusterInfo } from 'hooks/api/kubernetesClusters/useKubernetesClusterInfo';
import { useEffect, useMemo, useRef, useState } from 'react';
import {
  DbWizardFormFields,
  EKS_DEFAULT_LOAD_BALANCER_CONFIG,
  EMPTY_LOAD_BALANCER_CONFIGURATION,
  EVEREST_READ_ONLY_FINALIZER,
} from 'consts';
import { FormCard } from 'components/form-card';
import { usePodSchedulingPolicies, useSplitHorizonConfigs } from 'hooks';
import PoliciesDialog from './policies.dialog';
import { PodSchedulingPolicy } from 'shared-types/affinity.types';
import { dbTypeToDbEngine } from '@percona/utils';
import { emtpyConfig, SELECT_WIDTH } from './constants';
import { useLoadBalancerConfigs } from 'hooks/api/load-balancer';
import { LoadBalancerConfig } from 'shared-types/loadbalancer.types';
import LoadBalancerDialog from './load-balancer.dialog';
import WithInfoIcon from 'components/with-info-icon';
import { isSplitHorizonDNSEnabled } from 'utils/db';
import ToggableFormCard from 'components/toggable-form-card';

interface AdvancedConfigurationFormProps {
  dbType: DbType;
  showSplitHorizonDNS: boolean;
  loadingDefaultsForEdition?: boolean;
  automaticallyTogglePodSchedulingPolicySwitch?: boolean;
  allowedFieldsToInitiallyLoadDefaults?: AllowedFieldsToInitiallyLoadDefaults[];
  disableNoConfig?: boolean;
  activePolicy?: string;
  namespace?: string;
}

export const AdvancedConfigurationForm = ({
  dbType,
  loadingDefaultsForEdition,
  automaticallyTogglePodSchedulingPolicySwitch = false,
  allowedFieldsToInitiallyLoadDefaults = [],
  disableNoConfig = false,
  activePolicy,
  namespace,
  showSplitHorizonDNS,
}: AdvancedConfigurationFormProps) => {
  const { watch, setValue, getFieldState, getValues, trigger } =
    useFormContext();
  const [policyDialogOpen, setPolicyDialogOpen] = useState(false);
  const selectedPolicy = useRef<PodSchedulingPolicy>();
  const [loadBalancerConfigDialogOpen, setLoadBalancerConfigDialogOpen] =
    useState(false);
  const initialLbConfigName = useRef(
    getValues(AdvancedConfigurationFields.loadBalancerConfigName)
  );
  const selectedLoadBalancerConfig = useRef<
    LoadBalancerConfig | null | undefined
  >(null);
  const [
    engineParametersEnabled,
    policiesEnabled,
    exposureMethod,
    splitHorizonDNSEnabled,
  ] = watch([
    AdvancedConfigurationFields.engineParametersEnabled,
    AdvancedConfigurationFields.podSchedulingPolicyEnabled,
    AdvancedConfigurationFields.exposureMethod,
    AdvancedConfigurationFields.splitHorizonDNSEnabled,
  ]);
  const { data: clusterInfo, isLoading: clusterInfoLoading } =
    useKubernetesClusterInfo(['wizard-k8-info']);
  const { data: policies = [], isLoading: fetchingPolicies } =
    usePodSchedulingPolicies(dbTypeToDbEngine(dbType), false, {
      refetchInterval: 2000,
    });
  const {
    data: splitHorizonDNSConfigs = [],
    isLoading: fetchingSplitHorizonDNS,
  } = useSplitHorizonConfigs(namespace!, {
    enabled: !!namespace && showSplitHorizonDNS,
  });

  const clusterType = clusterInfo?.clusterType;

  const exposureMethods = Object.values(ProxyExposeType);
  const splitHorizonDNSDomain = splitHorizonDNSConfigs.find(
    (c) =>
      c.metadata.name === getValues(AdvancedConfigurationFields.splitHorizonDNS)
  )?.spec.baseDomainNameSuffix;

  const { data: loadBalancerConfigs, isLoading: fetchingLoadBalancer } =
    useLoadBalancerConfigs('load-balancer-configs', dbTypeToDbEngine(dbType), {
      refetchInterval: 10000,
    });

  const handleOnPolicyInfoClick = () => {
    const policyName = getValues<string>(
      AdvancedConfigurationFields.podSchedulingPolicy
    );
    selectedPolicy.current = policies.find(
      (p) => policyName === p.metadata.name
    );

    if (selectedPolicy.current) {
      setPolicyDialogOpen(true);
    }
  };

  const revalidateSourceRanges = () => {
    const sourceRanges = getValues(AdvancedConfigurationFields.sourceRanges);
    // @ts-ignore
    sourceRanges.forEach((range, i) => {
      if (range.sourceRange) {
        trigger(`${AdvancedConfigurationFields.sourceRanges}.${i}.sourceRange`);
      }
    });
  };

  const isEksDefault = useMemo(
    () =>
      clusterType &&
      clusterType === 'eks' &&
      !initialLbConfigName.current &&
      loadBalancerConfigs?.items.some(
        (config) => config.metadata.name === EKS_DEFAULT_LOAD_BALANCER_CONFIG
      ),
    [clusterType, loadBalancerConfigs?.items]
  );

  const selectOptions: LoadBalancerConfig[] = [
    emtpyConfig,
    ...(loadBalancerConfigs?.items ?? []),
  ];

  const selectDefaultValue = useMemo(
    () =>
      isEksDefault
        ? EKS_DEFAULT_LOAD_BALANCER_CONFIG
        : EMPTY_LOAD_BALANCER_CONFIGURATION,
    [isEksDefault]
  );

  useEffect(() => {
    if (
      !initialLbConfigName.current &&
      allowedFieldsToInitiallyLoadDefaults.includes('loadBalancerConfigName')
    ) {
      setValue(
        AdvancedConfigurationFields.loadBalancerConfigName,
        selectDefaultValue
      );
    }

    if (
      isSplitHorizonDNSEnabled(dbType) &&
      showSplitHorizonDNS &&
      allowedFieldsToInitiallyLoadDefaults.includes('splitHorizonDNS') &&
      splitHorizonDNSConfigs.length > 0
    ) {
      setValue(
        AdvancedConfigurationFields.splitHorizonDNS,
        splitHorizonDNSConfigs[0].metadata.name
      );
    }
  }, [
    allowedFieldsToInitiallyLoadDefaults,
    getValues,
    selectDefaultValue,
    setValue,
    splitHorizonDNSConfigs,
    dbType,
    showSplitHorizonDNS,
  ]);

  const handleOnLoadBalancerConfigInfoClick = () => {
    const configName = getValues<string>(
      AdvancedConfigurationFields.loadBalancerConfigName
    );

    if (configName === EMPTY_LOAD_BALANCER_CONFIGURATION || !configName) {
      return;
    }

    selectedLoadBalancerConfig.current = loadBalancerConfigs?.items.find(
      (config) => config.metadata?.name === configName
    );

    if (selectedLoadBalancerConfig.current) {
      setLoadBalancerConfigDialogOpen(true);
    }
  };

  const noConfig =
    getValues(AdvancedConfigurationFields.loadBalancerConfigName) ===
    EMPTY_LOAD_BALANCER_CONFIGURATION;

  // setting the storage class default value
  useEffect(() => {
    const { isTouched: storageClassTouched } = getFieldState(
      DbWizardFormFields.storageClass
    );

    revalidateSourceRanges();

    if (
      allowedFieldsToInitiallyLoadDefaults.includes(
        AdvancedConfigurationFields.storageClass
      ) &&
      !storageClassTouched &&
      clusterInfo?.storageClassNames &&
      clusterInfo.storageClassNames.length > 0
    ) {
      setValue(
        DbWizardFormFields.storageClass,
        clusterInfo?.storageClassNames[0],
        { shouldValidate: true }
      );
    }
  }, [
    clusterInfo,
    allowedFieldsToInitiallyLoadDefaults,
    getFieldState,
    setValue,
  ]);

  useEffect(() => {
    if (!policies.length) {
      return;
    }

    const { isTouched: policyTouched } = getFieldState(
      DbWizardFormFields.podSchedulingPolicy
    );

    const { isTouched: policyEnabledTouched } = getFieldState(
      DbWizardFormFields.podSchedulingPolicyEnabled
    );

    const defaultPolicy = policies.find((policy) =>
      policy.metadata.finalizers.includes(EVEREST_READ_ONLY_FINALIZER)
    );

    if (
      allowedFieldsToInitiallyLoadDefaults.includes(
        AdvancedConfigurationFields.podSchedulingPolicy
      ) &&
      !policyTouched
    ) {
      setValue(
        AdvancedConfigurationFields.podSchedulingPolicy,
        defaultPolicy ? defaultPolicy.metadata.name : policies[0].metadata.name
      );
    }

    if (
      defaultPolicy &&
      automaticallyTogglePodSchedulingPolicySwitch &&
      !policyEnabledTouched
    ) {
      setValue(AdvancedConfigurationFields.podSchedulingPolicyEnabled, true);
    }
  }, [
    policies,
    setValue,
    allowedFieldsToInitiallyLoadDefaults,
    automaticallyTogglePodSchedulingPolicySwitch,
    getFieldState,
  ]);

  const handleBlur = (value: string, fieldName: string, hasError: boolean) => {
    if (!hasError && !value.includes('/') && value !== '') {
      setValue(fieldName, `${value}/32`, { shouldValidate: true });
    }
  };

  useEffect(() => {
    trigger();
  }, [exposureMethod, trigger]);

  return (
    <FormGroup
      sx={{
        mt: 3,
        '& > .percona-rounded-box:not(:last-child)': {
          mb: 2,
        },
      }}
    >
      <FormCard
        title={Messages.cards.storage.title}
        description={
          !loadingDefaultsForEdition ? Messages.cards.storage.description : ''
        }
        controlComponent={
          <AutoCompleteInput
            name={AdvancedConfigurationFields.storageClass}
            loading={clusterInfoLoading}
            options={clusterInfo?.storageClassNames || []}
            disabled={
              !allowedFieldsToInitiallyLoadDefaults.includes('storageClass') ||
              loadingDefaultsForEdition
            }
            textFieldProps={{
              placeholder: Messages.placeholders.storageClass,
            }}
            tooltipText={
              allowedFieldsToInitiallyLoadDefaults.includes('storageClass')
                ? undefined
                : Messages.tooltipTexts.storageClass
            }
            autoCompleteProps={{
              disableClearable: true,
              sx: {
                width: SELECT_WIDTH,
                mt: 0,
              },
            }}
          />
        }
      />
      <ToggableFormCard
        title={Messages.cards.policies.title}
        description={Messages.cards.policies.description}
        tooltipText={!policies?.length ? Messages.tooltipTexts.noPolicies : ''}
        switchInputName={AdvancedConfigurationFields.podSchedulingPolicyEnabled}
        switchFieldProps={{
          disabled: !policies?.length,
        }}
        bottomSlot={
          <Box
            display="flex"
            justifyContent="space-between"
            alignItems="top"
            minHeight={'50px'}
          >
            {!!policiesEnabled && (
              <Box display="flex" ml="auto" alignItems="center">
                <SelectInput
                  name={AdvancedConfigurationFields.podSchedulingPolicy}
                  loading={fetchingPolicies || loadingDefaultsForEdition}
                  formControlProps={{
                    sx: {
                      width: SELECT_WIDTH,
                      mt: 0,
                    },
                  }}
                >
                  {policies
                    .filter(
                      (p) =>
                        !!p.spec.affinityConfig ||
                        p.metadata.name === activePolicy
                    )
                    .map((policy) => (
                      <MenuItem
                        value={policy.metadata.name}
                        key={policy.metadata.name}
                      >
                        {policy.metadata.name}
                      </MenuItem>
                    ))}
                </SelectInput>
                {!!policies.length && (
                  <IconButton onClick={handleOnPolicyInfoClick}>
                    <InfoIcon sx={{ width: '20px' }} />
                  </IconButton>
                )}
              </Box>
            )}
          </Box>
        }
      />
      <FormCard
        title={Messages.cards.enableExternalAccess.title}
        description={Messages.cards.enableExternalAccess.description}
        cardContent={
          <Box display="flex" flexDirection="column" gap={2}>
            <Box
              display="flex"
              alignItems="center"
              justifyContent="space-between"
            >
              <Typography variant="sectionHeading">
                {Messages.cards.exposureMethod.title}
              </Typography>
              <SelectInput
                name={AdvancedConfigurationFields.exposureMethod}
                loading={loadingDefaultsForEdition}
                formControlProps={{
                  sx: {
                    width: SELECT_WIDTH,
                    mt: 0,
                  },
                }}
              >
                {exposureMethods.map((method) => (
                  <MenuItem value={method} key={method}>
                    {PROXY_EXPOSE_TYPE_TO_LABEL[method]}
                  </MenuItem>
                ))}
              </SelectInput>
            </Box>

            {exposureMethod === ProxyExposeType.LoadBalancer && (
              <>
                <FormCard
                  title={Messages.cards.loadBalancerConfiguration.title}
                  controlComponent={
                    <WithInfoIcon
                      onClick={handleOnLoadBalancerConfigInfoClick}
                      disabled={noConfig}
                      tooltip={noConfig ? Messages.tooltipTexts.noConfig : ''}
                    >
                      <SelectInput
                        name={
                          AdvancedConfigurationFields.loadBalancerConfigName
                        }
                        loading={
                          fetchingLoadBalancer || loadingDefaultsForEdition
                        }
                        formControlProps={{
                          sx: {
                            width: SELECT_WIDTH,
                            mt: 0,
                            textAlign: 'left',
                          },
                        }}
                      >
                        {selectOptions.map((config) =>
                          disableNoConfig &&
                          config.metadata.name ===
                            EMPTY_LOAD_BALANCER_CONFIGURATION ? (
                            <Tooltip
                              title={Messages.tooltipTexts.limitations}
                              placement="right"
                              arrow
                            >
                              <span>
                                <MenuItem
                                  value={config.metadata.name}
                                  key={config.metadata.name}
                                  disabled
                                >
                                  {config.metadata.name}
                                </MenuItem>
                              </span>
                            </Tooltip>
                          ) : (
                            <MenuItem
                              value={config.metadata.name}
                              key={config.metadata.name}
                            >
                              {config.metadata.name}
                            </MenuItem>
                          )
                        )}
                      </SelectInput>
                    </WithInfoIcon>
                  }
                />
                <FormCard
                  title={Messages.cards.sourceRange.title}
                  description={Messages.cards.sourceRange.description}
                  cardContent={
                    <Stack data-testid="external-access-fields">
                      <TextArray
                        placeholder={Messages.sourceRangePlaceholder}
                        fieldName={AdvancedConfigurationFields.sourceRanges}
                        fieldKey="sourceRange"
                        handleBlur={(value, fieldName, hasError) => {
                          handleBlur(value, fieldName, hasError);
                          revalidateSourceRanges();
                        }}
                        onRemove={revalidateSourceRanges}
                      />
                    </Stack>
                  }
                />
              </>
            )}
          </Box>
        }
      />
      {isSplitHorizonDNSEnabled(dbType) && showSplitHorizonDNS && (
        <ToggableFormCard
          title={Messages.cards.splitHorizonDNS.title}
          description={Messages.cards.splitHorizonDNS.description}
          switchInputName={AdvancedConfigurationFields.splitHorizonDNSEnabled}
          bottomSlot={
            <Box
              display="flex"
              justifyContent="space-between"
              alignItems="top"
              minHeight={'50px'}
            >
              {!!splitHorizonDNSEnabled && (
                <WithInfoIcon
                  tooltip={
                    splitHorizonDNSDomain
                      ? Messages.tooltipTexts.splitHorizonDNS(
                          splitHorizonDNSDomain
                        )
                      : ''
                  }
                >
                  <SelectInput
                    name={AdvancedConfigurationFields.splitHorizonDNS}
                    loading={fetchingSplitHorizonDNS}
                    formControlProps={{
                      sx: {
                        width: SELECT_WIDTH,
                        mt: 0,
                        textAlign: 'left',
                      },
                    }}
                  >
                    {splitHorizonDNSConfigs.map((config) => (
                      <MenuItem
                        value={config.metadata.name}
                        key={config.metadata.name}
                      >
                        {config.metadata.name}
                      </MenuItem>
                    ))}
                  </SelectInput>
                </WithInfoIcon>
              )}
            </Box>
          }
        />
      )}
      <FormCard
        title={Messages.cards.engineParameters.title}
        description={Messages.cards.engineParameters.description}
        cardContent={
          <Stack>
            {engineParametersEnabled && (
              <TextInput
                name={AdvancedConfigurationFields.engineParameters}
                textFieldProps={{
                  placeholder: getParamsPlaceholderFromDbType(dbType),
                  multiline: true,
                  minRows: 3,
                }}
              />
            )}
          </Stack>
        }
        controlComponent={
          <SwitchInput
            label={Messages.enable}
            name={AdvancedConfigurationFields.engineParametersEnabled}
          />
        }
      />
      {policyDialogOpen && (
        <PoliciesDialog
          engineType={selectedPolicy.current!.spec.engineType}
          policy={selectedPolicy.current!}
          handleClose={() => setPolicyDialogOpen(false)}
        />
      )}
      {loadBalancerConfigDialogOpen && (
        <LoadBalancerDialog
          engineType={
            selectedLoadBalancerConfig.current!.spec.engineType ||
            DbEngineType.PSMDB
          }
          config={selectedLoadBalancerConfig.current!}
          handleClose={() => setLoadBalancerConfigDialogOpen(false)}
        />
      )}
    </FormGroup>
  );
};

export default AdvancedConfigurationForm;
