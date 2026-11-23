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

import { useFormContext, useWatch } from 'react-hook-form';
import {
  AuthModeSelect,
  CredentialsFields,
  SecretNamespacePermissions,
} from 'components/class-authentication';
import { OpenApiParametersFields } from 'components/openapi-parameters-fields';
import { UIGenerator } from 'components/ui-generator/ui-generator';
import {
  BackupStorageClass,
  BackupStorageCRD,
} from 'shared-types/backupStorages.types';
import { getDefaultStorageCredentialsSecretName } from 'utils/backup-storages';
import { StorageFormFields } from '../backup-storage-modal.constants';
import { Messages } from '../backup-storage-modal.messages';
import { BackupStorageFormValues } from '../backup-storage-modal.types';
import { getFormMode } from '../backup-storage-modal.utils';
import { useStorageClassForms } from '../hooks/use-storage-class-forms';

interface ClassFieldsProps {
  classes: BackupStorageClass[];
  original?: BackupStorageCRD;
  secretPermissions: SecretNamespacePermissions;
  secretPermissionsLoading: boolean;
}

// Everything the picked class defines: where backups go, how the store is
// reached, and how the storage authenticates to it.
export const ClassFields = ({
  classes,
  original,
  secretPermissions,
  secretPermissionsLoading,
}: ClassFieldsProps) => {
  const { control } = useFormContext<BackupStorageFormValues>();
  const [name, namespace] = useWatch({
    control,
    name: [StorageFormFields.name, StorageFormFields.namespace],
  });
  const { contract, forms } = useStorageClassForms(classes, original);
  const formMode = getFormMode(original);

  if (!contract) {
    return null;
  }

  return (
    <>
      <UIGenerator
        sectionKey={forms.location.sectionKey}
        sections={forms.location.sections}
        root={StorageFormFields.spec}
        namespace={namespace}
        formMode={formMode}
      />
      {forms.storage ? (
        <UIGenerator
          sectionKey={forms.storage.sectionKey}
          sections={forms.storage.sections}
          root={StorageFormFields.spec}
          namespace={namespace}
          formMode={formMode}
        />
      ) : (
        <OpenApiParametersFields
          name={StorageFormFields.fallbackParameters}
          parameterFields={forms.parameterFields}
        />
      )}
      <AuthModeSelect contract={contract} />
      <CredentialsFields
        contract={contract}
        form={forms.credentials}
        saved={original?.spec.authentication}
        namespace={namespace}
        defaultSecretName={getDefaultStorageCredentialsSecretName(name)}
        nouns={Messages.authenticationNouns}
        secretPermissions={secretPermissions}
        secretPermissionsLoading={secretPermissionsLoading}
      />
    </>
  );
};
