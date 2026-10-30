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

import { Stack, Typography } from '@mui/material';
import { CheckboxInput, TextInput } from '@percona/ui-lib';
import { useFormContext, useWatch } from 'react-hook-form';
import { UIGenerator } from 'components/ui-generator/ui-generator';
import { FormMode } from 'components/ui-generator/ui-generator.types';
import { MonitoringDestination } from 'shared-types/monitoring-destinations.types';
import { MonitoringSectionForm } from 'utils/monitoring-classes';
import {
  getCredentialsSecretName,
  getDefaultCredentialsSecretName,
} from 'utils/monitoring-destinations';
import { DestinationFormFields } from '../monitoring-destination-modal.constants';
import { Messages } from '../monitoring-destination-modal.messages';
import { MonitoringDestinationFormValues } from '../monitoring-destination-modal.types';
import { getCredentialsSecretAction } from '../monitoring-destination-modal.utils';

interface CredentialsFieldsProps {
  form?: MonitoringSectionForm;
  original?: MonitoringDestination;
  namespace: string;
}

export const CredentialsFields = ({
  form,
  original,
  namespace,
}: CredentialsFieldsProps) => {
  const { control } = useFormContext<MonitoringDestinationFormValues>();
  const [name, authMode, replaceCredentials] = useWatch({
    control,
    name: [
      DestinationFormFields.name,
      DestinationFormFields.authMode,
      DestinationFormFields.replaceCredentials,
    ],
  });
  const action = getCredentialsSecretAction(
    { authMode, replaceCredentials },
    original
  );
  const currentSecret = original && getCredentialsSecretName(original);
  const canKeepCredentials =
    !!currentSecret && authMode === original?.spec.authentication?.mode;

  if (!form || (!action && !canKeepCredentials)) {
    return null;
  }

  return (
    <Stack sx={{ mt: 3 }}>
      <Typography variant="sectionHeading">{Messages.credentials}</Typography>
      {canKeepCredentials && (
        <CheckboxInput
          name={DestinationFormFields.replaceCredentials}
          label={Messages.replaceCredentials(currentSecret)}
        />
      )}
      {action === 'create' && (
        <TextInput
          name={DestinationFormFields.credentialsSecretName}
          label={Messages.credentialsSecretName}
          textFieldProps={{
            placeholder: getDefaultCredentialsSecretName(name),
            helperText: Messages.credentialsSecretNameHelper(
              getDefaultCredentialsSecretName(name)
            ),
          }}
        />
      )}
      {action && (
        <UIGenerator
          sectionKey={form.sectionKey}
          sections={form.sections}
          root={DestinationFormFields.credentials}
          namespace={namespace}
          formMode={FormMode.New}
        />
      )}
    </Stack>
  );
};
