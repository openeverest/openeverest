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
import {
  getCredentialsSecretFilter,
  isSecretBackedAuthMode,
} from 'utils/class-contract';
import { AuthenticationFormFields } from '../class-authentication.constants';
import { Messages } from '../class-authentication.messages';
import {
  AuthenticationFormValues,
  CredentialsFieldsProps,
  CredentialsSource,
} from '../class-authentication.types';
import {
  getCredentialsSource,
  getCredentialsSources,
  getKeptCredentialsSecretName,
  takesCredentials,
} from '../class-authentication.utils';
import { useResetStaleCredentials } from '../hooks/use-reset-stale-credentials';
import { ExistingSecretSelect } from './existing-secret-select/existing-secret-select';

// Where the credentials of a mode that keeps them in a Secret come from, and
// the credentials themselves when the Secret is new or updated.
export const CredentialsFields = ({
  contract,
  form,
  saved,
  namespace,
  defaultSecretName,
  nouns,
  secretPermissions,
  secretPermissionsLoading,
}: CredentialsFieldsProps) => {
  const { control } = useFormContext<AuthenticationFormValues>();
  const [authMode, credentialsSource] = useWatch({
    control,
    name: [
      AuthenticationFormFields.authMode,
      AuthenticationFormFields.credentialsSource,
    ],
  });
  const values = { authMode, namespace, credentialsSource };
  const sources = getCredentialsSources(values, saved, secretPermissions);
  const source = getCredentialsSource(values, saved, secretPermissions);
  const keptSecret = getKeptCredentialsSecretName(authMode, saved) ?? '';
  useResetStaleCredentials({
    namespace,
    className: contract.name,
    authMode,
    form,
  });

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
          name={AuthenticationFormFields.credentialsSource}
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
          {Messages.updateCredentialsCaption(keptSecret, nouns.object)}
        </Typography>
      )}
      {source === CredentialsSource.New && (
        <TextInput
          name={AuthenticationFormFields.credentialsSecretName}
          label={Messages.credentialsSecretName}
          textFieldProps={{
            placeholder: defaultSecretName,
            helperText: Messages.credentialsSecretNameHelper(defaultSecretName),
          }}
        />
      )}
      {source === CredentialsSource.Existing && (
        <ExistingSecretSelect
          namespace={namespace}
          filter={getCredentialsSecretFilter(contract, authMode)}
        />
      )}
      {takesCredentials(source) && (
        <UIGenerator
          sectionKey={form.sectionKey}
          sections={form.sections}
          root={AuthenticationFormFields.credentials}
          namespace={namespace}
          formMode={FormMode.New}
        />
      )}
    </Stack>
  );
};
