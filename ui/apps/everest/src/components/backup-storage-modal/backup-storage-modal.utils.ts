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

import {
  getAuthenticationFormValues,
  getAuthenticationRequest,
  SecretNamespacePermissions,
  toCredentialsForm,
} from 'components/class-authentication';
import {
  getOpenApiParameterFields,
  OpenApiParameterFields,
  toOpenApiParameterFormValues,
  toOpenApiParametersUpdate,
} from 'components/openapi-parameters-fields';
import {
  getTlsConfigFormValues,
  toTlsConfig,
} from 'components/tls-config-fields';
import { FormMode, Section } from 'components/ui-generator/ui-generator.types';
import { FieldOverrides } from 'components/ui-generator/utils/field-overrides';
import {
  deepMerge,
  isPlainObject,
} from 'components/ui-generator/utils/object-path';
import { getComponentSourcePath } from 'components/ui-generator/utils/preprocess/normalized-component';
import { walkLeafComponents } from 'components/ui-generator/utils/schema-walker';
import {
  getSectionFormDefaults,
  getSectionFormUpdate,
  getSectionFormValues,
  toSectionForm,
} from 'components/ui-generator/utils/section-form';
import { SaveBackupStorageRequest } from 'hooks/api/backup-storages';
import {
  BackupStorageClass,
  BackupStorageCRD,
  BackupStorageSpec,
} from 'shared-types/backupStorages.types';
import {
  getBackupStorageClassName,
  getBackupStorageClassUiSchema,
  toBackupStorageClassContract,
} from 'utils/backup-storage-classes';
import { getDefaultStorageCredentialsSecretName } from 'utils/backup-storages';
import { toParametersContract } from 'utils/class-contract';
import {
  DEFAULT_LOCATION_SECTION,
  LOCATION_CONTRACT,
  LOCATION_SECTION_KEY,
  STORAGE_SECTION_KEY,
} from './backup-storage-modal.constants';
import { Messages } from './backup-storage-modal.messages';
import {
  BackupStorageFormValues,
  StorageClassForms,
} from './backup-storage-modal.types';

const NO_PARAMETER_FIELDS: OpenApiParameterFields = { fields: [], skipped: [] };

export const getFormMode = (original: BackupStorageCRD | undefined) =>
  original ? FormMode.Edit : FormMode.New;

// Engines treat the location as their catalog, so the API keeps it as the
// storage was added.
const getLocationOverrides = (
  section: Section,
  formMode: FormMode
): FieldOverrides => {
  const overrides: FieldOverrides = {};
  if (formMode !== FormMode.Edit) {
    return overrides;
  }
  walkLeafComponents(section.components, ({ component }) => {
    const path = getComponentSourcePath(component);
    if (path) {
      overrides[path] = {
        disabled: true,
        tooltip: Messages.fixedAfterCreation,
      };
    }
  });
  return overrides;
};

export const getStorageClassForms = (
  storageClass: BackupStorageClass | undefined,
  authMode: string,
  formMode: FormMode
): StorageClassForms => {
  const uiSchema = storageClass && getBackupStorageClassUiSchema(storageClass);
  const location = uiSchema?.location ?? DEFAULT_LOCATION_SECTION;
  const parametersSchema = storageClass?.spec.parametersSchema;

  return {
    location: toSectionForm(LOCATION_SECTION_KEY, location, {
      formMode,
      contract: LOCATION_CONTRACT,
      overrides: getLocationOverrides(location, formMode),
    }),
    storage:
      uiSchema?.storage &&
      toSectionForm(STORAGE_SECTION_KEY, uiSchema.storage, {
        formMode,
        contract: toParametersContract(parametersSchema),
      }),
    parameterFields: uiSchema?.storage
      ? NO_PARAMETER_FIELDS
      : getOpenApiParameterFields(parametersSchema?.openAPIV3Schema),
    credentials: toCredentialsForm(
      storageClass && toBackupStorageClassContract(storageClass),
      authMode
    ),
  };
};

// The class-dependent fields of a new storage of the class.
export const getNewStorageClassDefaults = (
  storageClass: BackupStorageClass | undefined
): Pick<
  BackupStorageFormValues,
  'authMode' | 'spec' | 'fallbackParameters'
> => {
  const authModes = storageClass
    ? toBackupStorageClassContract(storageClass).authModes
    : [];
  const authMode = authModes.length === 1 ? authModes[0].name : '';
  const forms = getStorageClassForms(storageClass, authMode, FormMode.New);
  return {
    authMode,
    spec: deepMerge(
      getSectionFormDefaults(forms.location),
      forms.storage ? getSectionFormDefaults(forms.storage) : {}
    ),
    fallbackParameters: toOpenApiParameterFormValues(
      undefined,
      forms.parameterFields
    ),
  };
};

export const getFormDefaults = ({
  original,
  namespace,
  classes,
}: {
  original?: BackupStorageCRD;
  namespace: string;
  classes: BackupStorageClass[];
}): BackupStorageFormValues => {
  const empty: BackupStorageFormValues = {
    name: '',
    namespace,
    className: '',
    ...getAuthenticationFormValues(undefined),
    spec: {},
    fallbackParameters: {},
    ...getTlsConfigFormValues(undefined),
  };
  if (!original) {
    return empty;
  }

  const originalClassName = original.spec.backupStorageClassRef.name;
  const forms = getStorageClassForms(
    classes.find((c) => getBackupStorageClassName(c) === originalClassName),
    original.spec.authentication.mode,
    FormMode.Edit
  );

  return {
    ...empty,
    name: original.metadata?.name ?? '',
    namespace: original.metadata?.namespace ?? namespace,
    className: originalClassName,
    ...getAuthenticationFormValues(
      original.spec.authentication,
      forms.credentials
    ),
    spec: deepMerge(
      getSectionFormValues(forms.location, original.spec),
      forms.storage ? getSectionFormValues(forms.storage, original.spec) : {}
    ),
    fallbackParameters: toOpenApiParameterFormValues(
      original.spec.parameters,
      forms.parameterFields
    ),
    ...getTlsConfigFormValues(original.spec.tls),
  };
};

const toLocation = (value: unknown): BackupStorageSpec['location'] => {
  const objectStore = isPlainObject(value) ? value.objectStore : undefined;
  const bucket =
    isPlainObject(objectStore) && typeof objectStore.bucket === 'string'
      ? objectStore.bucket
      : '';
  const prefix =
    isPlainObject(objectStore) && typeof objectStore.prefix === 'string'
      ? objectStore.prefix
      : '';
  return { objectStore: { bucket, ...(prefix && { prefix }) } };
};

// An edit keeps the saved parameters the class's form doesn't own as they
// were.
const toParameters = (
  values: BackupStorageFormValues,
  forms: StorageClassForms,
  original: BackupStorageCRD | undefined
): Record<string, unknown> => {
  if (!forms.storage) {
    return toOpenApiParametersUpdate(
      values.fallbackParameters,
      forms.parameterFields,
      original?.spec.parameters
    );
  }
  const { parameters } = getSectionFormUpdate(
    forms.storage,
    values.spec,
    original?.spec ?? {}
  );
  return isPlainObject(parameters) ? parameters : {};
};

// The requests a submit makes. Credentials go to the Secret only; the
// storage merely references it.
export const buildSaveRequest = ({
  values,
  storageClass,
  original,
  secretPermissions,
}: {
  values: BackupStorageFormValues;
  storageClass: BackupStorageClass | undefined;
  original?: BackupStorageCRD;
  secretPermissions: SecretNamespacePermissions;
}): SaveBackupStorageRequest => {
  const forms = getStorageClassForms(
    storageClass,
    values.authMode,
    getFormMode(original)
  );
  const { location } = getSectionFormUpdate(
    forms.location,
    values.spec,
    original?.spec ?? {}
  );
  const parameters = toParameters(values, forms, original);
  const tls = toTlsConfig(values);
  const { authentication, credentialsSecret } = getAuthenticationRequest({
    values,
    contract: storageClass && toBackupStorageClassContract(storageClass),
    credentialsForm: forms.credentials,
    saved: original?.spec.authentication,
    permissions: secretPermissions,
    defaultSecretName: getDefaultStorageCredentialsSecretName(values.name),
  });

  return {
    namespace: values.namespace,
    isEdit: !!original,
    credentialsSecret,
    storage: {
      ...(original && { apiVersion: original.apiVersion, kind: original.kind }),
      metadata: original ? { ...original.metadata } : { name: values.name },
      spec: {
        backupStorageClassRef: { name: values.className },
        authentication,
        location: toLocation(location),
        ...(tls && { tls }),
        ...(Object.keys(parameters).length > 0 && { parameters }),
      },
    },
  };
};
