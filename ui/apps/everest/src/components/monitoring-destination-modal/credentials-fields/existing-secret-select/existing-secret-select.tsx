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

import { MenuItem } from '@mui/material';
import { SelectInput } from '@percona/ui-lib';
import { useSecrets } from 'hooks/api/secrets';
import { useClusterName } from 'hooks/api/useClusterName';
import { DestinationFormFields } from '../../monitoring-destination-modal.constants';
import { Messages } from '../../monitoring-destination-modal.messages';

interface ExistingSecretSelectProps {
  namespace: string;
  className: string;
  authMode: string;
}

// Secrets labelled with the class and mode, as the Secrets API requires of the
// destination's credentials.
export const ExistingSecretSelect = ({
  namespace,
  className,
  authMode,
}: ExistingSecretSelectProps) => {
  const cluster = useClusterName();
  const {
    data: secrets = [],
    isLoading,
    isError,
  } = useSecrets(cluster, namespace, {
    monitoringClass: className,
    definition: authMode,
  });
  const names = secrets.flatMap(({ metadata }) =>
    metadata?.name ? [metadata.name] : []
  );

  const helperText = isLoading
    ? Messages.loadingSecrets
    : isError
      ? Messages.secretsLoadFailed
      : names.length === 0
        ? Messages.noExistingSecrets(namespace)
        : Messages.existingSecretHelper;

  return (
    <SelectInput
      name={DestinationFormFields.existingSecretName}
      label={Messages.existingSecret}
      helperText={helperText}
      loading={isLoading}
      selectFieldProps={{ disabled: names.length === 0 }}
      isRequired
    >
      {names.map((name) => (
        <MenuItem key={name} value={name}>
          {name}
        </MenuItem>
      ))}
    </SelectInput>
  );
};
