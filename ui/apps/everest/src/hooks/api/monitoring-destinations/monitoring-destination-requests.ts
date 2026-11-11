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
import { Secret, SecretDataPatch } from 'shared-types/api.types';
import {
  MonitoringDestination,
  MonitoringDestinationPayload,
} from 'shared-types/monitoring-destinations.types';
import {
  getCredentialsSecretLabels,
  isCredentialsSecretFor,
} from 'utils/monitoring-destinations';

export interface MonitoringDestinationRequests {
  createSecret: (secret: Secret) => Promise<unknown>;
  getSecret: (name: string, quiet?: boolean) => Promise<Secret>;
  patchSecret: (name: string, patch: SecretDataPatch) => Promise<unknown>;
  deleteSecret: (name: string) => Promise<unknown>;
  createDestination: (
    destination: MonitoringDestinationPayload
  ) => Promise<MonitoringDestination>;
  updateDestination: (
    destination: MonitoringDestinationPayload
  ) => Promise<MonitoringDestination>;
}

// create: a new Secret; update: new credentials in the destination's Secret,
// for every destination that uses it.
export interface CredentialsSecretWrite {
  action: 'create' | 'update';
  name: string;
  stringData: Record<string, string>;
}

export interface SaveMonitoringDestinationRequest {
  namespace: string;
  isEdit: boolean;
  destination: MonitoringDestinationPayload;
  credentialsSecret?: CredentialsSecretWrite;
}

// reusable: the Secret holds credentials of the destination's class and mode,
// so the destination can pick it as an existing Secret.
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

// Whether the Secret that took the name holds credentials of the class and
// mode. One the user may not read can't be referenced either.
const isReusableSecret = async (
  name: string,
  className: string,
  mode: string,
  requests: MonitoringDestinationRequests
) => {
  try {
    return isCredentialsSecretFor(
      await requests.getSecret(name, true),
      className,
      mode
    );
  } catch {
    return false;
  }
};

const createCredentialsSecret = async (
  { name, stringData }: CredentialsSecretWrite,
  destination: MonitoringDestinationPayload,
  requests: MonitoringDestinationRequests
) => {
  const className = destination.spec.classRef.name;
  const mode = destination.spec.authentication?.mode ?? '';
  try {
    await requests.createSecret({
      metadata: { name, labels: getCredentialsSecretLabels(className, mode) },
      stringData,
    });
  } catch (error) {
    if (hasStatus(error, 409)) {
      throw new SecretAlreadyExistsError(
        name,
        await isReusableSecret(name, className, mode, requests)
      );
    }
    throw error;
  }
};

const updateCredentialsSecret = async (
  { name, stringData }: CredentialsSecretWrite,
  requests: MonitoringDestinationRequests
) => {
  const { metadata } = await requests.getSecret(name);
  try {
    await requests.patchSecret(name, {
      metadata: { resourceVersion: metadata?.resourceVersion ?? '' },
      stringData,
    });
  } catch (error) {
    if (hasStatus(error, 409)) {
      throw new SecretChangedMeanwhileError(name);
    }
    throw error;
  }
};

// The Secret goes first, since the destination references it. A Secret this
// call created is deleted again when the destination can't be saved; updated
// credentials stay, as the old ones can't be read back. Secrets are never
// deleted otherwise: other destinations or users may hold them.
export const saveMonitoringDestination = async (
  { isEdit, destination, credentialsSecret }: SaveMonitoringDestinationRequest,
  requests: MonitoringDestinationRequests
): Promise<MonitoringDestination> => {
  if (credentialsSecret?.action === 'create') {
    await createCredentialsSecret(credentialsSecret, destination, requests);
  } else if (credentialsSecret?.action === 'update') {
    await updateCredentialsSecret(credentialsSecret, requests);
  }

  try {
    return isEdit
      ? await requests.updateDestination(destination)
      : await requests.createDestination(destination);
  } catch (error) {
    if (credentialsSecret?.action === 'create') {
      await requests
        .deleteSecret(credentialsSecret.name)
        .catch(() => undefined);
    }
    throw error;
  }
};
