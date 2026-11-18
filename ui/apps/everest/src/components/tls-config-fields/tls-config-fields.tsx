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
import { SwitchInput, TextInput } from '@percona/ui-lib';
import { useFormContext, useWatch } from 'react-hook-form';
import TlsAlert from 'components/tls-alert';
import { TlsConfigFormFields } from './tls-config-fields.constants';
import { Messages } from './tls-config-fields.messages';

// How the object verifies the server it reaches.
export const TlsConfigFields = () => {
  const { control } = useFormContext();
  const skipVerification: unknown = useWatch({
    control,
    name: TlsConfigFormFields.insecureSkipVerify,
  });

  return (
    <Stack sx={{ mt: 3 }}>
      <Typography variant="sectionHeading">{Messages.tls}</Typography>
      <SwitchInput
        name={TlsConfigFormFields.insecureSkipVerify}
        label={Messages.skipTlsVerification}
      />
      {skipVerification ? (
        <TlsAlert sx={{ mt: 1 }} />
      ) : (
        <>
          <TextInput
            name={TlsConfigFormFields.caSecretName}
            label={Messages.caSecretName}
            textFieldProps={{ helperText: Messages.caHelper }}
          />
          <TextInput
            name={TlsConfigFormFields.caSecretKey}
            label={Messages.caSecretKey}
          />
        </>
      )}
    </Stack>
  );
};
