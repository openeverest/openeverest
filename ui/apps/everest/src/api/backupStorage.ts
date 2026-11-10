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
import { SecretDataPatch } from 'shared-types/api.types';
import { api } from './api';
import {
  createSecretFn,
  deleteSecretFn,
  getSecretFn,
  patchSecretFn,
} from './secrets';

export const CA_SECRET_KEY = 'ca.crt';

const BACKUP_STORAGE_CLASS_SECRET_LABEL = 'openeverest.io/backup-storage-class';
const DEFINITION_SECRET_LABEL = 'openeverest.io/definition';

// The labels the Secrets API checks a credentials Secret against: the class
// that defines it and the authentication mode whose keys it holds.
export const getCredentialsSecretLabels = (
  backupStorageClass: string,
  authMode: string
): Record<string, string> => ({
  [BACKUP_STORAGE_CLASS_SECRET_LABEL]: backupStorageClass,
  [DEFINITION_SECRET_LABEL]: authMode,
});

// A CA bundle belongs to no class: core defines it, so it has no source label.
export const CA_BUNDLE_DEFINITION = 'ca-bundle';
export const CA_BUNDLE_SECRET_LABELS: Record<string, string> = {
  [DEFINITION_SECRET_LABEL]: CA_BUNDLE_DEFINITION,
};

type BackupStorageSpec = BackupStorageCRD['spec'];

type StorageSecretRefs = {
  credentialsSecretName?: string;
  caSecretKeyRef?: { name: string; key: string };
};

type SecretToCreate = {
  kind: BackupStorageSecretKind;
  name: string;
  labels: Record<string, string>;
  stringData: Record<string, string>;
};

type SecretToPatch = {
  name: string;
  set: Record<string, string>;
  remove: string[];
};

// What the form does to Secrets: the references the storage gets, the Secrets
// it creates, and the Secrets whose keys it changes in place.
type StorageSecretsPlan = {
  refs: StorageSecretRefs;
  create: SecretToCreate[];
  patch: SecretToPatch[];
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

// The name the form saves a storage's credentials or pasted CA under.
export const backupStorageSecretName = (
  storageName: string,
  kind: BackupStorageSecretKind
) => `backup-storage-${storageName}-${kind}`;

// Whether the storage trusts the CA the form saved for it from a pasted PEM,
// which a newly pasted PEM then replaces in place.
export const usesOwnCaSecret = (storage: BackupStorageCRD | undefined) =>
  !!storage &&
  storage.spec.tls?.ca?.secretKeyRef?.name ===
    backupStorageSecretName(
      storage.metadata?.name ?? '',
      BackupStorageSecretKind.CA
    );

const toStringData = (values: Record<string, unknown>) =>
  Object.fromEntries(
    Object.entries(values)
      .filter(([, value]) => value !== undefined && value !== null)
      .map(([key, value]) => [
        key,
        typeof value === 'string' ? value : JSON.stringify(value),
      ])
  );

const planStorageSecrets = (
  values: BackupStorageFormValues,
  previous?: BackupStorageCRD
): StorageSecretsPlan => {
  const plan: StorageSecretsPlan = { refs: {}, create: [], patch: [] };

  if (values.credentialsSource === CredentialsSource.New) {
    const name =
      values.credentialsSaveAs ||
      backupStorageSecretName(values.name, BackupStorageSecretKind.Credentials);
    plan.create.push({
      kind: BackupStorageSecretKind.Credentials,
      name,
      labels: getCredentialsSecretLabels(
        values.backupStorageClass,
        values.authMode
      ),
      stringData: toStringData(values.credentials),
    });
    plan.refs.credentialsSecretName = name;
  } else if (values.credentialsSecretName) {
    plan.refs.credentialsSecretName = values.credentialsSecretName;
    const set = toStringData(values.credentials);
    if (
      values.credentialsSource === CredentialsSource.Current &&
      (Object.keys(set).length > 0 || values.removedCredentialKeys.length > 0)
    ) {
      plan.patch.push({
        name: values.credentialsSecretName,
        set,
        remove: values.removedCredentialKeys,
      });
    }
  }

  if (values.tlsMode !== TlsMode.CustomCA) {
    return plan;
  }
  switch (values.caSource) {
    case CaSource.New: {
      const name = backupStorageSecretName(
        values.name,
        BackupStorageSecretKind.CA
      );
      const stringData = { [CA_SECRET_KEY]: values.caBundle };
      if (usesOwnCaSecret(previous)) {
        plan.patch.push({ name, set: stringData, remove: [] });
      } else {
        plan.create.push({
          kind: BackupStorageSecretKind.CA,
          name,
          labels: CA_BUNDLE_SECRET_LABELS,
          stringData,
        });
      }
      plan.refs.caSecretKeyRef = { name, key: CA_SECRET_KEY };
      break;
    }
    case CaSource.Existing:
      plan.refs.caSecretKeyRef = {
        name: values.caSecretName,
        key: CA_SECRET_KEY,
      };
      break;
    case CaSource.External:
      plan.refs.caSecretKeyRef = {
        name: values.caSecretName,
        key: values.caSecretKey,
      };
      break;
  }
  return plan;
};

const toDataPatch = (
  { set, remove }: SecretToPatch,
  resourceVersion: string
): SecretDataPatch => ({
  metadata: { resourceVersion },
  ...(Object.keys(set).length > 0 && { stringData: set }),
  ...(remove.length > 0 && {
    data: Object.fromEntries(remove.map((key) => [key, null])),
  }),
});

// Reads the resourceVersion of each Secret to patch and dry-runs the patch, so
// a patch that would be refused stops the edit before anything is written.
const dryRunSecretPatches = (
  cluster: string,
  namespace: string,
  patches: SecretToPatch[]
) =>
  Promise.all(
    patches.map(async (patch) => {
      const secret = await getSecretFn(cluster, namespace, patch.name);
      const body = toDataPatch(patch, secret.metadata?.resourceVersion ?? '');
      await patchSecretFn(cluster, namespace, patch.name, body, {
        dryRun: true,
      });
      return { name: patch.name, body };
    })
  );

const deleteSecrets = (cluster: string, namespace: string, names: string[]) =>
  Promise.allSettled(
    names.map((name) => deleteSecretFn(cluster, namespace, name))
  );

const createSecrets = async (
  cluster: string,
  namespace: string,
  secrets: SecretToCreate[]
): Promise<string[]> => {
  const created: string[] = [];
  try {
    for (const { kind, name, labels, stringData } of secrets) {
      try {
        await createSecretFn(cluster, namespace, {
          metadata: { name, labels },
          type: 'Opaque',
          stringData,
        });
      } catch (error) {
        if ((error as AxiosError)?.status === 409) {
          throw new BackupStorageSecretConflictError(kind, name);
        }
        throw error;
      }
      created.push(name);
    }
  } catch (error) {
    await deleteSecrets(cluster, namespace, created);
    throw error;
  }
  return created;
};

// Dry-runs the Secret patches, creates the Secrets the form asks for, runs
// `write`, then applies the patches. The Secrets created here are deleted
// again if a step fails before the storage is written, and a rejected storage
// leaves the patched Secrets untouched: the storage write can't be dry-run.
const writeWithStorageSecrets = async <T>(
  cluster: string,
  values: BackupStorageFormValues,
  previous: BackupStorageCRD | undefined,
  write: (refs: StorageSecretRefs) => Promise<T>
): Promise<T> => {
  const { namespace } = values;
  const plan = planStorageSecrets(values, previous);
  const patches = await dryRunSecretPatches(cluster, namespace, plan.patch);
  const created = await createSecrets(cluster, namespace, plan.create);

  let result: T;
  try {
    result = await write(plan.refs);
  } catch (error) {
    await deleteSecrets(cluster, namespace, created);
    throw error;
  }

  for (const { name, body } of patches) {
    await patchSecretFn(cluster, namespace, name, body);
  }
  return result;
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
  writeWithStorageSecrets(
    cluster,
    formData,
    undefined,
    async ({ credentialsSecretName, caSecretKeyRef }) => {
      const { name, namespace } = formData;
      const storageTls = tls(formData.tlsMode, caSecretKeyRef);
      const storage: BackupStorageCRD = {
        metadata: { name, namespace },
        spec: {
          backupStorageClassRef: { name: formData.backupStorageClass },
          location: {
            objectStore: {
              bucket: formData.spec.location?.objectStore?.bucket ?? '',
              ...(formData.spec.location?.objectStore?.prefix && {
                prefix: formData.spec.location.objectStore.prefix,
              }),
            },
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
  writeWithStorageSecrets(
    cluster,
    formData,
    previous,
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
