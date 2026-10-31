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

import { z } from 'zod';
import { CA_SECRET_KEY } from 'api/backupStorage';
import { FieldType, Section } from 'components/ui-generator/ui-generator.types';
import {
  deleteByPath,
  getByPath,
  isEmptyFieldValue,
  resolvePath,
  setByPath,
} from 'components/ui-generator/utils/object-path/object-path';
import {
  BackupStorageClass,
  BackupStorageCRD,
  BackupStorageFormValues,
  CaSource,
  CredentialsSource,
  TlsMode,
} from 'shared-types/backupStorages.types';
import { getBackupStorageClassUiSchema } from 'utils/backup-storages';
import { rfc_123_schema } from 'utils/common-validation';
import { Messages } from '../storage-locations.messages';

const PARAMETERS_PATH = 'parameters';
const PEM_BUNDLE_REGEX =
  /^\s*(-----BEGIN CERTIFICATE-----[\s\S]+?-----END CERTIFICATE-----\s*)+$/;

export type ClassFieldType =
  | FieldType.Text
  | FieldType.Select
  | FieldType.Toggle;

export type ClassField = {
  key: string;
  uiType: ClassFieldType;
  // Relative to the spec for location and storage fields, the Secret key for credentials.
  path: string;
  label: string;
  placeholder?: string;
  helperText?: string;
  password: boolean;
  options: { label: string; value: string }[];
  defaultValue?: unknown;
  required: boolean;
};

type LooseComponent = {
  uiType?: string;
  path?: string | string[];
  components?: Section['components'];
  componentsOrder?: string[];
  fieldParams?: {
    label?: string;
    placeholder?: string;
    helperText?: string;
    type?: string;
    options?: { label: string; value: string }[];
    defaultValue?: unknown;
  };
  validation?: { required?: boolean };
};

const CLASS_FIELD_TYPES: string[] = [
  FieldType.Text,
  FieldType.Select,
  FieldType.Toggle,
];

export const getSectionFields = (
  section?: Pick<Section, 'components' | 'componentsOrder'>
): ClassField[] => {
  if (!section) {
    return [];
  }

  return (section.componentsOrder ?? Object.keys(section.components)).flatMap(
    (key): ClassField[] => {
      const item = section.components[key] as LooseComponent | undefined;
      if (!item) {
        return [];
      }
      if (item.components) {
        return getSectionFields({
          components: item.components,
          componentsOrder: item.componentsOrder,
        });
      }

      const path = resolvePath(item.path);
      if (!path || !item.uiType || !CLASS_FIELD_TYPES.includes(item.uiType)) {
        return [];
      }

      const params = item.fieldParams ?? {};
      return [
        {
          key,
          uiType: item.uiType as ClassFieldType,
          path,
          label: params.label ?? key,
          placeholder: params.placeholder,
          helperText: params.helperText,
          password: params.type === 'password',
          options: params.options ?? [],
          defaultValue: params.defaultValue,
          required:
            item.uiType !== FieldType.Toggle && !!item.validation?.required,
        },
      ];
    }
  );
};

export const findBackupStorageClass = (
  backupStorageClasses: BackupStorageClass[],
  name: string
) =>
  backupStorageClasses.find(
    (backupStorageClass) => backupStorageClass.metadata?.name === name
  );

export const getStorageFields = (backupStorageClass?: BackupStorageClass) => {
  const uiSchema = getBackupStorageClassUiSchema(backupStorageClass);
  return {
    location: getSectionFields(uiSchema.location),
    storage: getSectionFields(uiSchema.storage),
  };
};

export const getCredentialsFields = (
  backupStorageClass: BackupStorageClass | undefined,
  authMode: string
) =>
  getSectionFields(
    getBackupStorageClassUiSchema(backupStorageClass).credentials?.[authMode]
  );

export const authModeUsesSecret = (
  backupStorageClass: BackupStorageClass | undefined,
  authMode: string
) =>
  !!backupStorageClass?.spec.authModes.find((mode) => mode.name === authMode)
    ?.credentials ||
  getCredentialsFields(backupStorageClass, authMode).length > 0;

// On edit, blank credentials keep the Secret of an unchanged auth mode.
export const canKeepCredentials = (
  previous: BackupStorageCRD | undefined,
  authMode: string
) =>
  !!previous?.spec.authentication.credentialsSecretRef?.name &&
  previous.spec.authentication.mode === authMode;

const entersCredentials = (
  values: Pick<BackupStorageFormValues, 'credentialsSource'>,
  credentialsFields: ClassField[]
) =>
  credentialsFields.length > 0 &&
  values.credentialsSource === CredentialsSource.New;

const allBlank = (values: Record<string, unknown>, fields: ClassField[]) =>
  fields.every((field) => isEmptyFieldValue(values[field.path]));

export const storageLocationDefaultValues = (
  namespace: string
): BackupStorageFormValues => ({
  name: '',
  namespace,
  backupStorageClass: '',
  spec: { location: { bucket: '', prefix: '' }, parameters: {} },
  authMode: '',
  credentialsSource: CredentialsSource.New,
  credentialsSecretName: '',
  credentials: {},
  tlsMode: TlsMode.System,
  caSource: CaSource.New,
  caBundle: '',
  caSecretName: '',
  caSecretKey: CA_SECRET_KEY,
});

export const storageLocationEditValues = (
  storage: BackupStorageCRD
): BackupStorageFormValues => {
  const { spec } = storage;
  const caSecretKeyRef = spec.tls?.ca?.secretKeyRef;
  let tlsMode = TlsMode.System;
  if (spec.tls?.insecureSkipVerify) {
    tlsMode = TlsMode.SkipVerify;
  } else if (caSecretKeyRef) {
    tlsMode = TlsMode.CustomCA;
  }

  return {
    name: storage.metadata?.name ?? '',
    namespace: storage.metadata?.namespace ?? '',
    backupStorageClass: spec.backupStorageClassRef.name,
    spec: {
      location: {
        bucket: spec.location.bucket,
        prefix: spec.location.prefix ?? '',
      },
      parameters: { ...spec.parameters },
    },
    authMode: spec.authentication.mode,
    credentialsSource: CredentialsSource.New,
    credentialsSecretName: spec.authentication.credentialsSecretRef?.name ?? '',
    credentials: {},
    tlsMode,
    caSource: caSecretKeyRef ? CaSource.Existing : CaSource.New,
    caBundle: '',
    caSecretName: caSecretKeyRef?.name ?? '',
    caSecretKey: caSecretKeyRef?.key ?? CA_SECRET_KEY,
  };
};

export const storageLocationSchema = (
  backupStorageClasses: BackupStorageClass[],
  previous?: BackupStorageCRD
) =>
  z
    .object({
      name: rfc_123_schema({ fieldName: 'storage name', maxLength: 22 }),
      namespace: z.string().min(1),
      backupStorageClass: z.string().min(1),
      spec: z.object({
        location: z.record(z.unknown()).optional(),
        parameters: z.record(z.unknown()).optional(),
      }),
      authMode: z.string(),
      credentialsSource: z.nativeEnum(CredentialsSource),
      credentialsSecretName: z.string(),
      credentials: z.record(z.unknown()),
      tlsMode: z.nativeEnum(TlsMode),
      caSource: z.nativeEnum(CaSource),
      caBundle: z.string(),
      caSecretName: z.string(),
      caSecretKey: z.string(),
    })
    .superRefine((values, ctx) => {
      const required = (
        path: string[],
        message = Messages.createEditModal.required
      ) => ctx.addIssue({ code: z.ZodIssueCode.custom, path, message });

      const backupStorageClass = findBackupStorageClass(
        backupStorageClasses,
        values.backupStorageClass
      );
      if (!backupStorageClass) {
        return;
      }

      const { location, storage } = getStorageFields(backupStorageClass);
      [...location, ...storage].forEach((field) => {
        if (
          field.required &&
          isEmptyFieldValue(getByPath(values.spec, field.path))
        ) {
          required(['spec', ...field.path.split('.')]);
        }
      });

      if (!values.authMode) {
        required(['authMode']);
        return;
      }

      if (authModeUsesSecret(backupStorageClass, values.authMode)) {
        const credentialsFields = getCredentialsFields(
          backupStorageClass,
          values.authMode
        );
        if (!entersCredentials(values, credentialsFields)) {
          if (!values.credentialsSecretName) {
            required(['credentialsSecretName']);
          }
        } else if (
          !(
            canKeepCredentials(previous, values.authMode) &&
            allBlank(values.credentials, credentialsFields)
          )
        ) {
          credentialsFields.forEach((field) => {
            if (
              field.required &&
              isEmptyFieldValue(values.credentials[field.path])
            ) {
              required(['credentials', field.path]);
            }
          });
        }
      }

      if (values.tlsMode === TlsMode.CustomCA) {
        if (values.caSource === CaSource.New) {
          if (!PEM_BUNDLE_REGEX.test(values.caBundle)) {
            required(['caBundle'], Messages.createEditModal.invalidCaBundle);
          }
        } else {
          if (!values.caSecretName) {
            required(['caSecretName']);
          }
          if (!values.caSecretKey) {
            required(['caSecretKey']);
          }
        }
      }
    });

// Keeps only what the selected class and auth mode render; on edit, parameters
// the class doesn't render are kept as they are.
export const prepareStorageSubmitValues = (
  values: BackupStorageFormValues,
  backupStorageClass: BackupStorageClass | undefined,
  previous?: BackupStorageCRD
): BackupStorageFormValues => {
  const spec = values.spec as Record<string, unknown>;
  const { location, storage } = getStorageFields(backupStorageClass);
  const parameters: Record<string, unknown> = {
    ...previous?.spec.parameters,
  };
  [...location, ...storage]
    .filter((field) => field.path.startsWith(`${PARAMETERS_PATH}.`))
    .forEach((field) => {
      const key = field.path.slice(PARAMETERS_PATH.length + 1);
      const value = getByPath(spec, field.path);
      if (isEmptyFieldValue(value)) {
        deleteByPath(parameters, key);
      } else {
        setByPath(parameters, key, value);
      }
    });
  const bucket = values.spec.location?.bucket?.trim() ?? '';
  const prefix = values.spec.location?.prefix?.trim();

  const credentialsFields = getCredentialsFields(
    backupStorageClass,
    values.authMode
  );
  const credentials = Object.fromEntries(
    credentialsFields
      .map((field) => [field.path, values.credentials[field.path]] as const)
      .filter(([, value]) => !isEmptyFieldValue(value))
  );

  let credentialsSource = CredentialsSource.Existing;
  let credentialsSecretName = '';
  if (authModeUsesSecret(backupStorageClass, values.authMode)) {
    if (!entersCredentials(values, credentialsFields)) {
      credentialsSecretName = values.credentialsSecretName;
    } else if (
      canKeepCredentials(previous, values.authMode) &&
      Object.keys(credentials).length === 0
    ) {
      credentialsSecretName =
        previous?.spec.authentication.credentialsSecretRef?.name ?? '';
    } else {
      credentialsSource = CredentialsSource.New;
    }
  }

  return {
    ...values,
    spec: {
      location: { bucket, ...(prefix && { prefix }) },
      parameters,
    },
    credentialsSource,
    credentialsSecretName,
    credentials: credentialsSource === CredentialsSource.New ? credentials : {},
  };
};
