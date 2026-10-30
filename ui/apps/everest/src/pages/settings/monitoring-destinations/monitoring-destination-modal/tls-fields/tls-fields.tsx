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
import TlsAlert from 'components/tls-alert';
import { DestinationFormFields } from '../monitoring-destination-modal.constants';
import { Messages } from '../monitoring-destination-modal.messages';
import { MonitoringDestinationFormValues } from '../monitoring-destination-modal.types';

export const TlsFields = () => {
  const { control } = useFormContext<MonitoringDestinationFormValues>();
  const skipVerification = useWatch({
    control,
    name: DestinationFormFields.insecureSkipVerify,
  });

  return (
    <Stack sx={{ mt: 3 }}>
      <Typography variant="sectionHeading">{Messages.tls}</Typography>
      <CheckboxInput
        name={DestinationFormFields.insecureSkipVerify}
        label={Messages.skipTlsVerification}
      />
      {skipVerification ? (
        <TlsAlert sx={{ mt: 1 }} />
      ) : (
        <>
          <TextInput
            name={DestinationFormFields.caSecretName}
            label={Messages.caSecretName}
            textFieldProps={{ helperText: Messages.caHelper }}
          />
          <TextInput
            name={DestinationFormFields.caSecretKey}
            label={Messages.caSecretKey}
          />
        </>
      )}
    </Stack>
  );
};
