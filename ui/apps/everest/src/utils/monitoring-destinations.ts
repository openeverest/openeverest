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

import { Section } from 'components/ui-generator/ui-generator.types';
import { getByPath } from 'components/ui-generator/utils/object-path';
import {
  MonitoringClass,
  MonitoringConditionType,
  MonitoringDestination,
  MonitoringDestinationState,
  MonitoringDestinationStatus,
} from 'shared-types/monitoring-destinations.types';
import { generateShortUID } from './generateShortUID';
import { getMonitoringClassUiSchema } from './monitoring-classes';

const ADDRESS_PARAMETER_PATH = 'parameters.url';
// The Ready reason a class controller reports before its first check.
const READY_PENDING_REASON = 'Pending';

export const getDestinationName = (destination: MonitoringDestination) =>
  destination.metadata?.name ?? '';

const CREDENTIALS_SECRET_SUFFIX = '-credentials';
const MAX_SECRET_NAME_LENGTH = 253;

export const getDefaultCredentialsSecretName = (destinationName: string) =>
  `${destinationName}${CREDENTIALS_SECRET_SUFFIX}`;

// A fresh name for rotated credentials, so the destination can move to the
// new Secret before the old one goes.
export const getRotatedCredentialsSecretName = (
  destinationName: string,
  suffix = `${Date.now().toString(36)}${generateShortUID()}`
) => {
  const base = getDefaultCredentialsSecretName(destinationName);
  return `${base.slice(0, MAX_SECRET_NAME_LENGTH - suffix.length - 1)}-${suffix}`;
};

// Secrets the form names: <destination>-credentials, or with a rotation
// suffix. Any other Secret was brought by the user and is never deleted.
export const isFormCreatedCredentialsSecret = (
  destinationName: string,
  secretName: string
) => {
  const base = getDefaultCredentialsSecretName(destinationName);
  return (
    secretName === base ||
    (secretName.startsWith(`${base}-`) &&
      /^[a-z0-9]+$/.test(secretName.slice(base.length + 1)))
  );
};

export const getCredentialsSecretName = (destination: MonitoringDestination) =>
  destination.spec.authentication?.credentialsSecretRef?.name;

// A Secret the form created for the destination may go once no other
// destination of the namespace references it.
export const isCredentialsSecretRemovable = (
  destinationName: string,
  secretName: string,
  namespaceDestinations: MonitoringDestination[]
) =>
  isFormCreatedCredentialsSecret(destinationName, secretName) &&
  !namespaceDestinations.some(
    (other) =>
      getDestinationName(other) !== destinationName &&
      getCredentialsSecretName(other) === secretName
  );

// The credentials Secret that goes with the destination on delete.
export const getSecretDeletedWithDestination = (
  destination: MonitoringDestination,
  namespaceDestinations: MonitoringDestination[]
): string | undefined => {
  const secretName = getCredentialsSecretName(destination);
  return secretName &&
    isCredentialsSecretRemovable(
      getDestinationName(destination),
      secretName,
      namespaceDestinations
    )
    ? secretName
    : undefined;
};

export const getDestinationStatus = (
  destination: MonitoringDestination
): MonitoringDestinationStatus => {
  const conditions = destination.status?.conditions ?? [];
  const accepted = conditions.find(
    (c) => c.type === MonitoringConditionType.Accepted
  );
  const ready = conditions.find(
    (c) => c.type === MonitoringConditionType.Ready
  );
  const serverVersion = destination.status?.serverVersion || undefined;
  const explainedBy = (
    state: MonitoringDestinationState,
    condition: (typeof conditions)[number] | undefined
  ): MonitoringDestinationStatus => ({
    state,
    reason: condition?.reason || undefined,
    message: condition?.message || undefined,
    serverVersion,
  });

  if (accepted?.status === 'False') {
    return explainedBy(MonitoringDestinationState.Failing, accepted);
  }
  if (ready?.status === 'True') {
    return { state: MonitoringDestinationState.Ready, serverVersion };
  }
  if (ready?.status === 'False' && ready.reason !== READY_PENDING_REASON) {
    return explainedBy(MonitoringDestinationState.Failing, ready);
  }
  return explainedBy(MonitoringDestinationState.Pending, ready ?? accepted);
};

const toPaths = (path: string | string[] | undefined): string[] => {
  if (path === undefined) {
    return [];
  }
  return Array.isArray(path) ? path : [path];
};

const hasAddressComponent = (components: Section['components']): boolean =>
  Object.values(components).some((component) =>
    'components' in component
      ? hasAddressComponent(component.components)
      : toPaths(component.path).includes(ADDRESS_PARAMETER_PATH)
  );

// The backend address, when the class's destination form keeps it in
// parameters.url; classes such as Prometheus have none.
export const getDestinationAddress = (
  destination: MonitoringDestination,
  monitoringClass: MonitoringClass | undefined
): string | undefined => {
  const section = monitoringClass
    ? getMonitoringClassUiSchema(monitoringClass).destination
    : undefined;
  if (!section || !hasAddressComponent(section.components)) {
    return undefined;
  }
  const url = getByPath(destination.spec, ADDRESS_PARAMETER_PATH);
  return typeof url === 'string' && url ? url : undefined;
};
