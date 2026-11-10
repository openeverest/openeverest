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
import { getAuthModeCredentialsSection } from 'utils/auth-mode-credentials';
import { getBackupStorageClassUiSchema } from 'utils/backup-storages';
import { rfc_123_schema } from 'utils/common-validation';
import { Messages } from '../storage-locations.messages';

const PARAMETERS_PATH = 'parameters';
const MAX_SECRET_NAME_LENGTH = 253;
const secretNameSchema = rfc_123_schema({
  fieldName: 'name',
  maxLength: MAX_SECRET_NAME_LENGTH,
});
// Certificate bodies are base64 (RFC 7468): without '-', a body cannot run
// past its END line, which keeps matching linear.
const PEM_BUNDLE_REGEX =
  /^\s*(?:-----BEGIN CERTIFICATE-----[A-Za-z0-9+/=\s]+-----END CERTIFICATE-----\s*)+$/;

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

// A credentials key of the auth mode, as the form renders it.
export type CredentialsField = ClassField & {
  // Declared required by the mode, so a saved Secret is known to hold it.
  requiredByMode: boolean;
};

const findAuthMode = (
  backupStorageClass: BackupStorageClass | undefined,
  authMode: string
) => backupStorageClass?.spec.authModes.find((mode) => mode.name === authMode);

const toMaskedField = (key: string): ClassField => ({
  key,
  uiType: FieldType.Text,
  path: key,
  label: key,
  password: true,
  options: [],
  required: false,
});

// The class's credentials form for the mode or, when it ships none, one masked
// input per credential key the mode declares. Keys the mode declares and the
// form leaves out are added as masked inputs: the keys come from the mode.
export const getCredentialsFields = (
  backupStorageClass: BackupStorageClass | undefined,
  authMode: string
): CredentialsField[] => {
  const declared = findAuthMode(backupStorageClass, authMode)?.credentials;
  const requiredKeys = declared?.required ?? [];
  const fields = getSectionFields(
    getAuthModeCredentialsSection(
      backupStorageClass?.spec.authModes ?? [],
      getBackupStorageClassUiSchema(backupStorageClass).credentials ?? {},
      authMode
    )
  );
  const missing = [...requiredKeys, ...(declared?.optional ?? [])]
    .filter((key) => !fields.some((field) => field.path === key))
    .map(toMaskedField);

  return [...fields, ...missing].map((field) => ({
    ...field,
    required: field.required || requiredKeys.includes(field.path),
    requiredByMode: requiredKeys.includes(field.path),
  }));
};

export const authModeUsesSecret = (
  backupStorageClass: BackupStorageClass | undefined,
  authMode: string
) =>
  !!findAuthMode(backupStorageClass, authMode)?.credentials ||
  getCredentialsFields(backupStorageClass, authMode).length > 0;

// On edit, the storage keeps its Secret, and its keys can be replaced in place,
// as long as the auth mode is unchanged.
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

// The keys whose values are sent when replacing keys in place: the required
// keys being replaced and the optional keys not being removed, where a blank
// value keeps the stored one.
const touchedCredentials = (
  values: Pick<
    BackupStorageFormValues,
    'replacedCredentialKeys' | 'removedCredentialKeys'
  >,
  credentialsFields: CredentialsField[]
) =>
  credentialsFields.filter((field) =>
    field.requiredByMode
      ? values.replacedCredentialKeys.includes(field.path)
      : !values.removedCredentialKeys.includes(field.path)
  );

export const storageLocationDefaultValues = (
  namespace: string
): BackupStorageFormValues => ({
  name: '',
  namespace,
  backupStorageClass: '',
  spec: {
    location: { objectStore: { bucket: '', prefix: '' } },
    parameters: {},
  },
  authMode: '',
  credentialsSource: CredentialsSource.New,
  credentialsSecretName: '',
  credentialsSaveAs: '',
  credentials: {},
  replacedCredentialKeys: [],
  removedCredentialKeys: [],
  tlsMode: TlsMode.System,
  caSource: CaSource.New,
  caBundle: '',
  caSecretName: '',
  caSecretKey: CA_SECRET_KEY,
});

// A labelled CA bundle holds its certificates under ca.crt, so a reference to
// any other key names a Secret from elsewhere, e.g. cert-manager.
const getCaSource = (caSecretKeyRef?: { key: string }) => {
  if (!caSecretKeyRef) {
    return CaSource.New;
  }
  return caSecretKeyRef.key === CA_SECRET_KEY
    ? CaSource.Existing
    : CaSource.External;
};

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
        objectStore: {
          bucket: spec.location.objectStore.bucket,
          prefix: spec.location.objectStore.prefix ?? '',
        },
      },
      parameters: { ...spec.parameters },
    },
    authMode: spec.authentication.mode,
    credentialsSource: spec.authentication.credentialsSecretRef
      ? CredentialsSource.Current
      : CredentialsSource.New,
    credentialsSecretName: spec.authentication.credentialsSecretRef?.name ?? '',
    credentialsSaveAs: '',
    credentials: {},
    replacedCredentialKeys: [],
    removedCredentialKeys: [],
    tlsMode,
    caSource: getCaSource(caSecretKeyRef),
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
      credentialsSaveAs: z.string(),
      credentials: z.record(z.unknown()),
      replacedCredentialKeys: z.array(z.string()),
      removedCredentialKeys: z.array(z.string()),
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
        if (
          values.credentialsSource === CredentialsSource.Current &&
          canKeepCredentials(previous, values.authMode)
        ) {
          touchedCredentials(values, credentialsFields).forEach((field) => {
            if (
              field.requiredByMode &&
              isEmptyFieldValue(values.credentials[field.path])
            ) {
              required(['credentials', field.path]);
            }
          });
        } else if (!entersCredentials(values, credentialsFields)) {
          if (!values.credentialsSecretName) {
            required(['credentialsSecretName']);
          }
        } else {
          credentialsFields.forEach((field) => {
            if (
              field.required &&
              isEmptyFieldValue(values.credentials[field.path])
            ) {
              required(['credentials', field.path]);
            }
          });
          if (values.credentialsSaveAs) {
            secretNameSchema
              .safeParse(values.credentialsSaveAs)
              .error?.issues.forEach((issue) =>
                ctx.addIssue({ ...issue, path: ['credentialsSaveAs'] })
              );
          }
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
          if (values.caSource === CaSource.External && !values.caSecretKey) {
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
  const bucket = values.spec.location?.objectStore?.bucket?.trim() ?? '';
  const prefix = values.spec.location?.objectStore?.prefix?.trim();

  const credentialsFields = getCredentialsFields(
    backupStorageClass,
    values.authMode
  );
  const nonEmptyCredentials = (fields: ClassField[]) =>
    Object.fromEntries(
      fields
        .map((field) => [field.path, values.credentials[field.path]] as const)
        .filter(([, value]) => !isEmptyFieldValue(value))
    );

  let credentialsSource = CredentialsSource.Existing;
  let credentialsSecretName = '';
  let credentials: Record<string, unknown> = {};
  let removedCredentialKeys: string[] = [];
  if (authModeUsesSecret(backupStorageClass, values.authMode)) {
    if (
      values.credentialsSource === CredentialsSource.Current &&
      canKeepCredentials(previous, values.authMode)
    ) {
      credentialsSource = CredentialsSource.Current;
      credentialsSecretName =
        previous?.spec.authentication.credentialsSecretRef?.name ?? '';
      credentials = nonEmptyCredentials(
        touchedCredentials(values, credentialsFields)
      );
      removedCredentialKeys = values.removedCredentialKeys.filter((key) =>
        credentialsFields.some(
          (field) => field.path === key && !field.requiredByMode
        )
      );
    } else if (entersCredentials(values, credentialsFields)) {
      credentialsSource = CredentialsSource.New;
      credentials = nonEmptyCredentials(credentialsFields);
    } else {
      credentialsSecretName = values.credentialsSecretName;
    }
  }

  return {
    ...values,
    spec: {
      location: { objectStore: { bucket, ...(prefix && { prefix }) } },
      parameters,
    },
    credentialsSource,
    credentialsSecretName,
    credentialsSaveAs:
      credentialsSource === CredentialsSource.New
        ? values.credentialsSaveAs.trim()
        : '',
    credentials,
    replacedCredentialKeys: [],
    removedCredentialKeys,
  };
};
