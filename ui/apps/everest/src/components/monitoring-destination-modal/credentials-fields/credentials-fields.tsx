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

import { Alert, Stack, Typography } from '@mui/material';
import { RadioGroup, TextInput } from '@percona/ui-lib';
import { useFormContext, useWatch } from 'react-hook-form';
import { UIGenerator } from 'components/ui-generator/ui-generator';
import { FormMode } from 'components/ui-generator/ui-generator.types';
import { MonitoringDestination } from 'shared-types/monitoring-destinations.types';
import {
  isSecretBackedAuthMode,
  MonitoringSectionForm,
} from 'utils/monitoring-classes';
import { getDefaultCredentialsSecretName } from 'utils/monitoring-destinations';
import { DestinationFormFields } from '../monitoring-destination-modal.constants';
import { Messages } from '../monitoring-destination-modal.messages';
import {
  CredentialsSource,
  MonitoringDestinationFormValues,
  SecretNamespacePermissions,
} from '../monitoring-destination-modal.types';
import {
  getCredentialsSource,
  getCredentialsSources,
  getKeptCredentialsSecretName,
  takesCredentials,
} from '../monitoring-destination-modal.utils';
import { ExistingSecretSelect } from './existing-secret-select/existing-secret-select';

interface CredentialsFieldsProps {
  form?: MonitoringSectionForm;
  original?: MonitoringDestination;
  namespace: string;
  secretPermissions: SecretNamespacePermissions;
  secretPermissionsLoading: boolean;
}

export const CredentialsFields = ({
  form,
  original,
  namespace,
  secretPermissions,
  secretPermissionsLoading,
}: CredentialsFieldsProps) => {
  const { control } = useFormContext<MonitoringDestinationFormValues>();
  const [name, className, authMode, credentialsSource] = useWatch({
    control,
    name: [
      DestinationFormFields.name,
      DestinationFormFields.className,
      DestinationFormFields.authMode,
      DestinationFormFields.credentialsSource,
    ],
  });
  const values = { authMode, namespace, credentialsSource };
  const sources = getCredentialsSources(values, original, secretPermissions);
  const source = getCredentialsSource(values, original, secretPermissions);
  const keptSecret = getKeptCredentialsSecretName(authMode, original) ?? '';

  if (!form || !authMode || !isSecretBackedAuthMode(authMode)) {
    return null;
  }

  return (
    <Stack sx={{ mt: 3 }}>
      <Typography variant="sectionHeading">{Messages.credentials}</Typography>
      {!source && !secretPermissionsLoading && (
        <Alert severity="error" sx={{ mt: 1 }}>
          {Messages.noSecretAccess(namespace)}
        </Alert>
      )}
      {sources.length > 1 && (
        <RadioGroup
          name={DestinationFormFields.credentialsSource}
          options={sources.map((option) => ({
            value: option,
            label: Messages.credentialsSource[option](keptSecret),
          }))}
          radioGroupFieldProps={{ value: source ?? '' }}
        />
      )}
      {sources.length === 1 && source === CredentialsSource.Keep && (
        <Typography variant="body2" sx={{ mt: 1 }}>
          {Messages.keptCredentials(keptSecret)}
        </Typography>
      )}
      {source === CredentialsSource.Update && (
        <Typography variant="caption" color="text.secondary">
          {Messages.updateCredentialsCaption(keptSecret)}
        </Typography>
      )}
      {source === CredentialsSource.New && (
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
      {source === CredentialsSource.Existing && (
        <ExistingSecretSelect
          namespace={namespace}
          className={className}
          authMode={authMode}
        />
      )}
      {takesCredentials(source) && (
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
