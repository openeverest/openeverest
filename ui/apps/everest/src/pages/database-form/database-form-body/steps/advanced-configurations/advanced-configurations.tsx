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

import { useFormContext } from 'react-hook-form';
import { Messages } from './advanced-configurations.messages.ts';

import { DbWizardFormFields } from 'consts.ts';
import { StepHeader } from '../step-header/step-header.tsx';
import AdvancedConfigurationForm from 'components/cluster-form/advanced-configuration/advanced-configuration.tsx';
import { StepProps } from 'pages/database-form/database-form.types.ts';
import { useDatabasePageMode } from 'pages/database-form/useDatabasePageMode.ts';
import { WizardMode } from 'shared-types/wizard.types.ts';
import { useMemo } from 'react';
import { AllowedFieldsToInitiallyLoadDefaults } from 'components/cluster-form/advanced-configuration/advanced-configuration.types';

export const AdvancedConfigurations = ({
  loadingDefaultsForEdition,
}: StepProps) => {
  const { watch, getValues } = useFormContext();
  const dbType = watch(DbWizardFormFields.dbType);
  const namespace = getValues(DbWizardFormFields.k8sNamespace);
  const mode = useDatabasePageMode();
  const allowedFieldsToInitiallyLoadDefaults: AllowedFieldsToInitiallyLoadDefaults[] =
    useMemo(() => {
      if (mode === WizardMode.New) {
        return [
          'storageClass',
          'podSchedulingPolicy',
          'loadBalancerConfigName',
          'splitHorizonDNS',
        ];
      }
      return [];
    }, [mode]);

  return (
    <>
      <StepHeader pageTitle={Messages.advanced} />

      <AdvancedConfigurationForm
        dbType={dbType}
        loadingDefaultsForEdition={loadingDefaultsForEdition}
        automaticallyTogglePodSchedulingPolicySwitch={mode === WizardMode.New}
        allowedFieldsToInitiallyLoadDefaults={
          allowedFieldsToInitiallyLoadDefaults
        }
        namespace={namespace}
        showSplitHorizonDNS={!getValues(DbWizardFormFields.sharding)}
      />
    </>
  );
};
