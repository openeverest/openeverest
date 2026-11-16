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

import { useMemo } from 'react';
import { useFormContext, useWatch } from 'react-hook-form';
import {
  AuthModeSelect,
  CredentialsFields,
  SecretNamespacePermissions,
} from 'components/class-authentication';
import {
  MonitoringClass,
  MonitoringDestination,
} from 'shared-types/monitoring-destinations.types';
import { toMonitoringClassContract } from 'utils/monitoring-classes';
import { getDefaultCredentialsSecretName } from 'utils/monitoring-destinations';
import { useDestinationClassForms } from '../hooks/use-destination-class-forms';
import { DestinationFormFields } from '../monitoring-destination-modal.constants';
import { Messages } from '../monitoring-destination-modal.messages';
import { MonitoringDestinationFormValues } from '../monitoring-destination-modal.types';
import { getFormMode } from '../monitoring-destination-modal.utils';
import { ParametersFields } from '../parameters-fields/parameters-fields';

interface ClassFieldsProps {
  classes: MonitoringClass[];
  original?: MonitoringDestination;
  secretPermissions: SecretNamespacePermissions;
  secretPermissionsLoading: boolean;
}

// Everything the picked class defines: its auth modes, credentials and
// parameters.
export const ClassFields = ({
  classes,
  original,
  secretPermissions,
  secretPermissionsLoading,
}: ClassFieldsProps) => {
  const { control } = useFormContext<MonitoringDestinationFormValues>();
  const [name, namespace] = useWatch({
    control,
    name: [DestinationFormFields.name, DestinationFormFields.namespace],
  });
  const { monitoringClass, forms } = useDestinationClassForms(
    classes,
    original
  );
  const contract = useMemo(
    () => monitoringClass && toMonitoringClassContract(monitoringClass),
    [monitoringClass]
  );

  if (!contract) {
    return null;
  }

  return (
    <>
      <AuthModeSelect contract={contract} />
      <ParametersFields
        forms={forms}
        namespace={namespace}
        formMode={getFormMode(original)}
      />
      <CredentialsFields
        contract={contract}
        form={forms.credentials}
        saved={original?.spec.authentication}
        namespace={namespace}
        defaultSecretName={getDefaultCredentialsSecretName(name)}
        nouns={Messages.authenticationNouns}
        secretPermissions={secretPermissions}
        secretPermissionsLoading={secretPermissionsLoading}
      />
    </>
  );
};
