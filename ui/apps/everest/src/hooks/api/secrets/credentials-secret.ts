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

import { isAxiosError } from 'axios';
import {
  createSecretFn,
  deleteSecretFn,
  getSecretFn,
  patchSecretFn,
} from 'api/secrets';
import { Secret, SecretDataPatch } from 'shared-types/api.types';

// Whether a failed read is left unreported to the user; 'if-missing' leaves
// out only a Secret that doesn't exist, which the caller handles.
export type QuietSecretRead = boolean | 'if-missing';

export interface CredentialsSecretRequests {
  createSecret: (secret: Secret) => Promise<unknown>;
  getSecret: (name: string, quiet?: QuietSecretRead) => Promise<Secret>;
  patchSecret: (name: string, patch: SecretDataPatch) => Promise<unknown>;
  deleteSecret: (name: string) => Promise<unknown>;
}

export const getCredentialsSecretRequests = (
  cluster: string,
  namespace: string
): CredentialsSecretRequests => ({
  createSecret: (secret) => createSecretFn(cluster, namespace, secret),
  getSecret: (name, quiet = false) =>
    getSecretFn(
      cluster,
      namespace,
      name,
      quiet === 'if-missing' ? (error) => error.status === 404 : quiet
    ),
  patchSecret: (name, patch) => patchSecretFn(cluster, namespace, name, patch),
  deleteSecret: (name) => deleteSecretFn(cluster, namespace, name),
});

// create: a new Secret, labelled for the class and mode its credentials
// belong to; update: new credentials in the object's Secret, for every object
// that uses it, or that Secret created anew when it is gone.
export interface CredentialsSecretWrite {
  action: 'create' | 'update';
  name: string;
  labels: Record<string, string>;
  stringData: Record<string, string>;
}

// reusable: the Secret holds credentials of the object's class and mode, so
// the object can pick it as an existing Secret.
export class SecretAlreadyExistsError extends Error {
  readonly secretName: string;
  readonly reusable: boolean;

  constructor(secretName: string, reusable: boolean) {
    super(`Secret ${secretName} already exists`);
    this.secretName = secretName;
    this.reusable = reusable;
    this.name = 'SecretAlreadyExistsError';
  }
}

// The Secret changed between reading its version and writing the credentials.
export class SecretChangedMeanwhileError extends Error {
  readonly secretName: string;

  constructor(secretName: string) {
    super(`Secret ${secretName} changed meanwhile`);
    this.secretName = secretName;
    this.name = 'SecretChangedMeanwhileError';
  }
}

const hasStatus = (error: unknown, status: number) =>
  isAxiosError(error) && error.response?.status === status;

// One the user may not read can't be referenced either.
const isReusableSecret = async (
  name: string,
  labels: Record<string, string>,
  requests: CredentialsSecretRequests
) => {
  try {
    const { metadata } = await requests.getSecret(name, true);
    return Object.entries(labels).every(
      ([key, value]) => metadata?.labels?.[key] === value
    );
  } catch {
    return false;
  }
};

const createCredentialsSecret = async (
  { name, labels, stringData }: CredentialsSecretWrite,
  requests: CredentialsSecretRequests
) => {
  try {
    await requests.createSecret({ metadata: { name, labels }, stringData });
  } catch (error) {
    if (hasStatus(error, 409)) {
      throw new SecretAlreadyExistsError(
        name,
        await isReusableSecret(name, labels, requests)
      );
    }
    throw error;
  }
};

const findSecret = async (
  name: string,
  requests: CredentialsSecretRequests
): Promise<Secret | undefined> => {
  try {
    return await requests.getSecret(name, 'if-missing');
  } catch (error) {
    if (hasStatus(error, 404)) {
      return undefined;
    }
    throw error;
  }
};

// An object whose Secret is gone is not valid until it has one again, so the
// new credentials create it anew. Returns whether they did.
const updateCredentialsSecret = async (
  write: CredentialsSecretWrite,
  requests: CredentialsSecretRequests
): Promise<boolean> => {
  const { name, stringData } = write;
  const secret = await findSecret(name, requests);
  if (!secret) {
    await createCredentialsSecret(write, requests);
    return true;
  }
  try {
    await requests.patchSecret(name, {
      metadata: { resourceVersion: secret.metadata?.resourceVersion ?? '' },
      stringData,
    });
  } catch (error) {
    if (hasStatus(error, 409)) {
      throw new SecretChangedMeanwhileError(name);
    }
    throw error;
  }
  return false;
};

const writeCredentialsSecret = async (
  write: CredentialsSecretWrite,
  requests: CredentialsSecretRequests
): Promise<boolean> => {
  if (write.action === 'update') {
    return updateCredentialsSecret(write, requests);
  }
  await createCredentialsSecret(write, requests);
  return true;
};

// The Secret goes first, since the object references it. A Secret this call
// created is deleted again when the object can't be saved; updated
// credentials stay, as the old ones can't be read back. Secrets are never
// deleted otherwise: other objects or users may hold them.
export const saveWithCredentialsSecret = async <T>(
  {
    credentialsSecret,
    save,
  }: {
    credentialsSecret?: CredentialsSecretWrite;
    save: () => Promise<T>;
  },
  requests: CredentialsSecretRequests
): Promise<T> => {
  const created =
    !!credentialsSecret &&
    (await writeCredentialsSecret(credentialsSecret, requests));

  try {
    return await save();
  } catch (error) {
    if (credentialsSecret && created) {
      await requests
        .deleteSecret(credentialsSecret.name)
        .catch(() => undefined);
    }
    throw error;
  }
};
