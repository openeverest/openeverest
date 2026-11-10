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

import { Box } from '@mui/material';
import { useFormContext, useWatch } from 'react-hook-form';
import { DbWizardFormFields } from 'consts';
import { StepHeader } from 'pages/database-form/database-form-body/steps-old/step-header/step-header';
import { StepProps } from 'pages/database-form/database-form.types';
import { MonitoringDestinationsPicker } from './monitoring-destinations-picker/monitoring-destinations-picker';
import { Messages } from './monitoring-step.messages';

export const MonitoringStep = (
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  _props: StepProps
) => {
  const { control } = useFormContext();
  const [namespace, provider] = useWatch({
    control,
    name: [DbWizardFormFields.k8sNamespace, DbWizardFormFields.provider],
  });

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column' }}>
      <StepHeader
        pageTitle={Messages.title}
        pageDescription={Messages.caption}
      />
      <MonitoringDestinationsPicker
        namespace={typeof namespace === 'string' ? namespace : ''}
        provider={typeof provider === 'string' ? provider : ''}
        pruneUnavailable
      />
    </Box>
  );
};
