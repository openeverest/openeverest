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

import { MenuItem, Typography } from '@mui/material';
import { SelectInput } from '@percona/ui-lib';
import { useFormContext, useWatch } from 'react-hook-form';
import {
  MonitoringClass,
  MonitoringDestination,
} from 'shared-types/monitoring-destinations.types';
import { CredentialsFields } from '../credentials-fields/credentials-fields';
import { useDestinationClassForms } from '../hooks/use-destination-class-forms';
import { DestinationFormFields } from '../monitoring-destination-modal.constants';
import { Messages } from '../monitoring-destination-modal.messages';
import { MonitoringDestinationFormValues } from '../monitoring-destination-modal.types';
import { getFormMode } from '../monitoring-destination-modal.utils';
import { ParametersFields } from '../parameters-fields/parameters-fields';

interface ClassFieldsProps {
  classes: MonitoringClass[];
  original?: MonitoringDestination;
}

// Everything the picked class defines: its auth modes, credentials and
// parameters.
export const ClassFields = ({ classes, original }: ClassFieldsProps) => {
  const { control } = useFormContext<MonitoringDestinationFormValues>();
  const namespace = useWatch({
    control,
    name: DestinationFormFields.namespace,
  });
  const { monitoringClass, forms } = useDestinationClassForms(
    classes,
    original
  );

  if (!monitoringClass) {
    return null;
  }

  return (
    <>
      {monitoringClass.spec.description && (
        <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
          {monitoringClass.spec.description}
        </Typography>
      )}
      <SelectInput
        name={DestinationFormFields.authMode}
        label={Messages.authMode}
        isRequired
      >
        {monitoringClass.spec.authModes.map(({ name }) => (
          <MenuItem key={name} value={name}>
            {name}
          </MenuItem>
        ))}
      </SelectInput>
      <ParametersFields
        forms={forms}
        namespace={namespace}
        formMode={getFormMode(original)}
      />
      <CredentialsFields
        form={forms.credentials}
        original={original}
        namespace={namespace}
      />
    </>
  );
};
