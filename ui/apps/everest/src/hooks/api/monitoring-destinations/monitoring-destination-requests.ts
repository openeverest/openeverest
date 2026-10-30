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
import { Secret } from 'shared-types/api.types';
import {
  MonitoringDestination,
  MonitoringDestinationPayload,
} from 'shared-types/monitoring-destinations.types';
import {
  getDestinationName,
  getSecretDeletedWithDestination,
} from 'utils/monitoring-destinations';

export interface MonitoringDestinationRequests {
  createSecret: (secret: Secret) => Promise<unknown>;
  deleteSecret: (name: string) => Promise<unknown>;
  createDestination: (
    destination: MonitoringDestinationPayload
  ) => Promise<MonitoringDestination>;
  updateDestination: (
    destination: MonitoringDestinationPayload
  ) => Promise<MonitoringDestination>;
  deleteDestination: (name: string) => Promise<unknown>;
}

// create: a new Secret; replace: the referenced Secret recreated with new
// values (there is no Secret update); reuse: an existing Secret as it is.
export type CredentialsSecretWrite =
  | {
      action: 'create' | 'replace';
      name: string;
      stringData: Record<string, string>;
    }
  | { action: 'reuse'; name: string };

export interface SaveMonitoringDestinationRequest {
  namespace: string;
  isEdit: boolean;
  destination: MonitoringDestinationPayload;
  credentialsSecret?: CredentialsSecretWrite;
}

export class SecretAlreadyExistsError extends Error {
  readonly secretName: string;

  constructor(secretName: string) {
    super(`Secret ${secretName} already exists`);
    this.secretName = secretName;
    this.name = 'SecretAlreadyExistsError';
  }
}

const hasStatus = (error: unknown, status: number) =>
  isAxiosError(error) && error.response?.status === status;

const ignoreNotFound = (error: unknown) => {
  if (!hasStatus(error, 404)) {
    throw error;
  }
};

// The Secret goes first, since the destination references it; a Secret this
// call created is deleted again when the destination can't be saved.
export const saveMonitoringDestination = async (
  { isEdit, destination, credentialsSecret }: SaveMonitoringDestinationRequest,
  requests: MonitoringDestinationRequests
): Promise<MonitoringDestination> => {
  let createdSecretName: string | undefined;

  if (credentialsSecret && credentialsSecret.action !== 'reuse') {
    const { action, name, stringData } = credentialsSecret;
    if (action === 'replace') {
      await requests.deleteSecret(name).catch(ignoreNotFound);
    }
    try {
      await requests.createSecret({ metadata: { name }, stringData });
    } catch (error) {
      if (action === 'create' && hasStatus(error, 409)) {
        throw new SecretAlreadyExistsError(name);
      }
      throw error;
    }
    if (action === 'create') {
      createdSecretName = name;
    }
  }

  try {
    return isEdit
      ? await requests.updateDestination(destination)
      : await requests.createDestination(destination);
  } catch (error) {
    if (createdSecretName) {
      await requests.deleteSecret(createdSecretName).catch(() => undefined);
    }
    throw error;
  }
};

export const deleteMonitoringDestination = async (
  {
    destination,
    namespaceDestinations,
  }: {
    destination: MonitoringDestination;
    namespaceDestinations: MonitoringDestination[];
  },
  requests: Pick<
    MonitoringDestinationRequests,
    'deleteDestination' | 'deleteSecret'
  >
): Promise<void> => {
  await requests.deleteDestination(getDestinationName(destination));

  const secretName = getSecretDeletedWithDestination(
    destination,
    namespaceDestinations
  );
  if (secretName) {
    await requests.deleteSecret(secretName).catch(ignoreNotFound);
  }
};
