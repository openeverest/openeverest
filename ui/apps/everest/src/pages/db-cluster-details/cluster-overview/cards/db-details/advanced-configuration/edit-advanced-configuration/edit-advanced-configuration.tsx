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

import { FormDialog } from 'components/form-dialog/form-dialog';
import { SubmitHandler } from 'react-hook-form';
import AdvancedConfigurationForm from 'components/cluster-form/advanced-configuration';
import { AdvancedConfigurationModalProps } from './edit-advanced-configuration.types';
import {
  AdvancedConfigurationFormType,
  advancedConfigurationsSchema,
} from 'components/cluster-form/advanced-configuration/advanced-configuration-schema';
import { Messages } from './edit-advanced-configuration.messages';
import { dbEngineToDbType } from '@percona/utils';
import { advancedConfigurationModalDefaultValues } from 'components/cluster-form/advanced-configuration/advanced-configuration.utils';
import { EMPTY_LOAD_BALANCER_CONFIGURATION } from 'consts';
import { AllowedFieldsToInitiallyLoadDefaults } from 'components/cluster-form/advanced-configuration/advanced-configuration.types';
import { useMemo } from 'react';
import { ProxyExposeType } from 'shared-types/dbCluster.types';

export const AdvancedConfigurationEditModal = ({
  open,
  handleCloseModal,
  handleSubmitModal,
  dbCluster,
  submitting,
}: AdvancedConfigurationModalProps) => {
  const onSubmit: SubmitHandler<AdvancedConfigurationFormType> = ({
    exposureMethod,
    engineParametersEnabled,
    engineParameters,
    sourceRanges,
    storageClass,
    podSchedulingPolicyEnabled,
    podSchedulingPolicy,
    loadBalancerConfigName,
    splitHorizonDNSEnabled,
    splitHorizonDNS,
  }) => {
    handleSubmitModal({
      engineParametersEnabled,
      engineParameters,
      sourceRanges,
      storageClass,
      podSchedulingPolicyEnabled,
      podSchedulingPolicy,
      exposureMethod,
      loadBalancerConfigName:
        loadBalancerConfigName !== EMPTY_LOAD_BALANCER_CONFIGURATION
          ? loadBalancerConfigName
          : '',
      splitHorizonDNSEnabled,
      splitHorizonDNS,
    });
  };

  const allowedFieldsToInitiallyLoadDefaults: AllowedFieldsToInitiallyLoadDefaults[] =
    useMemo(() => {
      const result: AllowedFieldsToInitiallyLoadDefaults[] = [
        'loadBalancerConfigName',
      ];
      if (!dbCluster?.spec.podSchedulingPolicyName) {
        result.push('podSchedulingPolicy');
      }
      return result;
    }, [dbCluster?.spec.podSchedulingPolicyName]);

  return (
    <FormDialog
      dataTestId="edit-advanced-configuration"
      size="XL"
      isOpen={open}
      closeModal={handleCloseModal}
      schema={advancedConfigurationsSchema()}
      headerMessage={Messages.formDialogHeader}
      onSubmit={onSubmit}
      submitting={submitting}
      submitMessage={Messages.save}
      defaultValues={advancedConfigurationModalDefaultValues(dbCluster)}
    >
      <AdvancedConfigurationForm
        dbType={dbEngineToDbType(dbCluster?.spec?.engine?.type)}
        allowedFieldsToInitiallyLoadDefaults={
          allowedFieldsToInitiallyLoadDefaults
        }
        disableNoConfig={
          dbCluster.spec.proxy?.expose?.type === ProxyExposeType.LoadBalancer &&
          dbCluster.spec.proxy?.expose?.loadBalancerConfigName !== undefined
        }
        activePolicy={dbCluster?.spec.podSchedulingPolicyName}
        namespace={dbCluster?.metadata.namespace}
        showSplitHorizonDNS={false}
      />
    </FormDialog>
  );
};
