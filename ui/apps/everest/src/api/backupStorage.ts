// Copyright (C) 2023 Percona LLC
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

import { AxiosError } from 'axios';
import {
  BackupStorageFormValues,
  BackupStorageCRD,
  BackupStorageListCRD,
  BackupStorageListFilter,
  BackupStorageSecretKind,
  CaSource,
  CredentialsSource,
  TlsMode,
} from 'shared-types/backupStorages.types';
import { generateShortUID } from 'utils/generateShortUID';
import { api } from './api';
import { createSecretFn, deleteSecretFn } from './secrets';

export const CA_SECRET_KEY = 'ca.crt';

type BackupStorageSpec = BackupStorageCRD['spec'];

type StorageSecretRefs = {
  credentialsSecretName?: string;
  caSecretKeyRef?: { name: string; key: string };
};

export class BackupStorageSecretConflictError extends Error {
  kind: BackupStorageSecretKind;
  secretName: string;

  constructor(kind: BackupStorageSecretKind, secretName: string) {
    super(`Secret "${secretName}" already exists`);
    this.kind = kind;
    this.secretName = secretName;
  }
}

// Secrets can't be updated, so an edit writes new ones under a unique name.
export const backupStorageSecretName = (
  storageName: string,
  kind: BackupStorageSecretKind,
  unique = false
) =>
  `backup-storage-${storageName}-${kind}${unique ? `-${generateShortUID()}` : ''}`;

const toStringData = (values: Record<string, unknown>) =>
  Object.fromEntries(
    Object.entries(values)
      .filter(([, value]) => value !== undefined && value !== null)
      .map(([key, value]) => [
        key,
        typeof value === 'string' ? value : JSON.stringify(value),
      ])
  );

const createStorageSecret = async (
  cluster: string,
  namespace: string,
  name: string,
  kind: BackupStorageSecretKind,
  stringData: Record<string, string>
): Promise<string> => {
  try {
    await createSecretFn(cluster, namespace, {
      metadata: { name },
      type: 'Opaque',
      stringData,
    });
  } catch (error) {
    if ((error as AxiosError)?.status === 409) {
      throw new BackupStorageSecretConflictError(kind, name);
    }
    throw error;
  }
  return name;
};

// Creates the Secrets the form asks for, then runs `write`; the Secrets
// created here are deleted again if any step fails.
const withStorageSecrets = async <T>(
  cluster: string,
  values: BackupStorageFormValues,
  isEdit: boolean,
  write: (refs: StorageSecretRefs) => Promise<T>
): Promise<T> => {
  const { name, namespace } = values;
  const created: string[] = [];
  const create = async (
    kind: BackupStorageSecretKind,
    stringData: Record<string, string>
  ) => {
    const secretName = await createStorageSecret(
      cluster,
      namespace,
      backupStorageSecretName(name, kind, isEdit),
      kind,
      stringData
    );
    created.push(secretName);
    return secretName;
  };

  try {
    const refs: StorageSecretRefs = {};
    if (values.credentialsSource === CredentialsSource.New) {
      refs.credentialsSecretName = await create(
        BackupStorageSecretKind.Credentials,
        toStringData(values.credentials)
      );
    } else if (values.credentialsSecretName) {
      refs.credentialsSecretName = values.credentialsSecretName;
    }

    if (values.tlsMode === TlsMode.CustomCA) {
      refs.caSecretKeyRef =
        values.caSource === CaSource.New
          ? {
              name: await create(BackupStorageSecretKind.CA, {
                [CA_SECRET_KEY]: values.caBundle,
              }),
              key: CA_SECRET_KEY,
            }
          : { name: values.caSecretName, key: values.caSecretKey };
    }

    return await write(refs);
  } catch (error) {
    await Promise.allSettled(
      created.map((secretName) =>
        deleteSecretFn(cluster, namespace, secretName)
      )
    );
    throw error;
  }
};

const tls = (
  tlsMode: TlsMode,
  caSecretKeyRef?: { name: string; key: string }
): BackupStorageSpec['tls'] => {
  switch (tlsMode) {
    case TlsMode.SkipVerify:
      return { insecureSkipVerify: true };
    case TlsMode.CustomCA:
      return { ca: { secretKeyRef: caSecretKeyRef } };
    default:
      return undefined;
  }
};

export const getBackupStoragesFn = async (
  cluster: string,
  namespace: string,
  filter?: BackupStorageListFilter
): Promise<BackupStorageCRD[]> => {
  const response = await api.get<BackupStorageListCRD>(
    `clusters/${cluster}/namespaces/${namespace}/backup-storages`,
    { params: filter }
  );
  return response.data?.items ?? [];
};

export const createBackupStorageFn = async (
  cluster: string,
  formData: BackupStorageFormValues
) =>
  withStorageSecrets(
    cluster,
    formData,
    false,
    async ({ credentialsSecretName, caSecretKeyRef }) => {
      const { name, namespace } = formData;
      const storageTls = tls(formData.tlsMode, caSecretKeyRef);
      const storage: BackupStorageCRD = {
        metadata: { name, namespace },
        spec: {
          backupStorageClassRef: { name: formData.backupStorageClass },
          location: {
            bucket: formData.spec.location?.bucket ?? '',
            ...(formData.spec.location?.prefix && {
              prefix: formData.spec.location.prefix,
            }),
          },
          parameters: (formData.spec.parameters ??
            {}) as BackupStorageSpec['parameters'],
          authentication: {
            mode: formData.authMode,
            ...(credentialsSecretName && {
              credentialsSecretRef: { name: credentialsSecretName },
            }),
          },
          ...(storageTls && { tls: storageTls }),
        },
      };
      const response = await api.post<BackupStorageCRD>(
        `clusters/${cluster}/namespaces/${namespace}/backup-storages`,
        storage
      );
      return response.data;
    }
  );

// A merge patch: parameters and TLS settings the form dropped are set to null.
export const editBackupStorageFn = async (
  cluster: string,
  formData: BackupStorageFormValues,
  previous: BackupStorageCRD
) =>
  withStorageSecrets(
    cluster,
    formData,
    true,
    async ({ credentialsSecretName, caSecretKeyRef }) => {
      const { name, namespace } = formData;
      const storageTls = tls(formData.tlsMode, caSecretKeyRef);
      const droppedParameters = Object.fromEntries(
        Object.keys(previous.spec.parameters ?? {}).map((key) => [key, null])
      );
      const patch = {
        spec: {
          parameters: { ...droppedParameters, ...formData.spec.parameters },
          authentication: {
            mode: formData.authMode,
            credentialsSecretRef: credentialsSecretName
              ? { name: credentialsSecretName }
              : null,
          },
          tls: storageTls
            ? {
                ca: storageTls.ca ?? null,
                insecureSkipVerify: storageTls.insecureSkipVerify ?? null,
              }
            : null,
        },
      };
      const response = await api.patch<BackupStorageCRD>(
        `clusters/${cluster}/namespaces/${namespace}/backup-storages/${name}`,
        patch,
        { headers: { 'Content-Type': 'application/merge-patch+json' } }
      );
      return response.data;
    }
  );

export const deleteBackupStorageFn = async (
  cluster: string,
  backupStorageId: string,
  namespace: string
) => {
  const response = await api.delete(
    `clusters/${cluster}/namespaces/${namespace}/backup-storages/${backupStorageId}`
  );
  return response.data;
};
