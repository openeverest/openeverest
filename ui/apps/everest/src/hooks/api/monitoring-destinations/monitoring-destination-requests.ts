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
  getCredentialsSecretLabels,
  getDestinationName,
  getSecretDeletedWithDestination,
  isCredentialsSecretFor,
  isSecretUsedByOtherDestination,
} from 'utils/monitoring-destinations';

export interface MonitoringDestinationRequests {
  createSecret: (secret: Secret) => Promise<unknown>;
  getSecret: (name: string) => Promise<Secret>;
  deleteSecret: (name: string) => Promise<unknown>;
  createDestination: (
    destination: MonitoringDestinationPayload
  ) => Promise<MonitoringDestination>;
  updateDestination: (
    destination: MonitoringDestinationPayload
  ) => Promise<MonitoringDestination>;
  deleteDestination: (name: string) => Promise<unknown>;
  listDestinations: () => Promise<MonitoringDestination[]>;
}

// create: a new Secret; replace: a new Secret under a fresh name that takes
// over from previousName (there is no Secret update, and switching names
// leaves no moment without credentials); reuse: an existing Secret as it is.
// previousName is set only for a Secret the UI created, never one the user
// brought.
export type CredentialsSecretWrite =
  | {
      action: 'create';
      name: string;
      stringData: Record<string, string>;
    }
  | {
      action: 'replace';
      name: string;
      stringData: Record<string, string>;
      previousName?: string;
    }
  | { action: 'reuse'; name: string };

export interface SaveMonitoringDestinationRequest {
  namespace: string;
  isEdit: boolean;
  destination: MonitoringDestinationPayload;
  credentialsSecret?: CredentialsSecretWrite;
}

// reusable: the Secret holds credentials of the destination's class and mode,
// so the destination can reference it as it is.
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

const hasStatus = (error: unknown, status: number) =>
  isAxiosError(error) && error.response?.status === status;

const ignoreNotFound = (error: unknown) => {
  if (!hasStatus(error, 404)) {
    throw error;
  }
};

// The replaced Secret goes once the destination no longer references it, if
// no other destination uses it. Best effort: the save has already succeeded.
const deleteReplacedSecret = async (
  destinationName: string,
  secretName: string,
  requests: MonitoringDestinationRequests
) => {
  try {
    const namespaceDestinations = await requests.listDestinations();
    if (
      !isSecretUsedByOtherDestination(
        destinationName,
        secretName,
        namespaceDestinations
      )
    ) {
      await requests.deleteSecret(secretName);
    }
  } catch {
    // A leftover Secret is harmless: the destination already uses the new one.
  }
};

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
      await requests.getSecret(name),
      className,
      mode
    );
  } catch {
    return false;
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
    const className = destination.spec.classRef.name;
    const mode = destination.spec.authentication?.mode ?? '';
    try {
      await requests.createSecret({
        metadata: { name, labels: getCredentialsSecretLabels(className, mode) },
        stringData,
      });
    } catch (error) {
      if (action === 'create' && hasStatus(error, 409)) {
        throw new SecretAlreadyExistsError(
          name,
          await isReusableSecret(name, className, mode, requests)
        );
      }
      throw error;
    }
    createdSecretName = name;
  }

  let saved: MonitoringDestination;
  try {
    saved = isEdit
      ? await requests.updateDestination(destination)
      : await requests.createDestination(destination);
  } catch (error) {
    if (createdSecretName) {
      await requests.deleteSecret(createdSecretName).catch(() => undefined);
    }
    throw error;
  }

  if (
    credentialsSecret?.action === 'replace' &&
    credentialsSecret.previousName &&
    credentialsSecret.previousName !== credentialsSecret.name
  ) {
    await deleteReplacedSecret(
      destination.metadata?.name ?? '',
      credentialsSecret.previousName,
      requests
    );
  }
  return saved;
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
