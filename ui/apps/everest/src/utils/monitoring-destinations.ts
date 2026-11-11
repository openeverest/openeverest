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
import { Section } from 'components/ui-generator/ui-generator.types';
import { SECRET_DEFINITION_LABEL } from 'components/ui-generator/secret-field/secret-create-modal/secret-create-modal.constants';
import { getByPath } from 'components/ui-generator/utils/object-path';
import { APIError, Instance, Secret } from 'shared-types/api.types';
import {
  MonitoringBinding,
  MonitoringClass,
  MonitoringConditionType,
  MonitoringDestination,
  MonitoringDestinationState,
  MonitoringDestinationStatus,
} from 'shared-types/monitoring-destinations.types';
import { getMonitoringClassUiSchema } from './monitoring-classes';

const ADDRESS_PARAMETER_PATH = 'parameters.url';
// The Ready reason a class controller reports before its first check.
const READY_PENDING_REASON = 'Pending';
// Reason on the 409 that refuses deleting a destination Instances still list.
const IN_USE_REASON = 'InUse';

export const isDestinationInUseError = (error: unknown) =>
  isAxiosError<APIError>(error) &&
  error.response?.status === 409 &&
  error.response.data?.reason === IN_USE_REASON;

// Core counts the Instances the user can't read; the readable ones cover a status lagging a new attachment.
export const getAttachedInstancesCount = (
  destination: MonitoringDestination,
  readableInstances: string[]
) =>
  Math.max(
    destination.status?.attachedInstances ?? 0,
    readableInstances.length
  );

export const getDestinationName = (destination: MonitoringDestination) =>
  destination.metadata?.name ?? '';

export const getInstanceMonitoringDestinationNames = (
  instance: Instance
): string[] =>
  (instance.spec?.monitoring?.destinations ?? []).map(
    (entry) => entry.destinationRef.name
  );

const CREDENTIALS_SECRET_PREFIX = 'monitoring-destination-';
const CREDENTIALS_SECRET_SUFFIX = '-credentials';

// The kind in the name keeps a destination's Secret apart from those of a
// backup storage or an Instance with the same name.
export const getDefaultCredentialsSecretName = (destinationName: string) =>
  `${CREDENTIALS_SECRET_PREFIX}${destinationName}${CREDENTIALS_SECRET_SUFFIX}`;

export const MONITORING_CLASS_SECRET_LABEL = 'openeverest.io/monitoring-class';

// The labels the Secrets API checks a credentials Secret against: the class
// that defines it and the authentication mode whose keys it holds.
export const getCredentialsSecretLabels = (
  className: string,
  mode: string
): Record<string, string> => ({
  [MONITORING_CLASS_SECRET_LABEL]: className,
  [SECRET_DEFINITION_LABEL]: mode,
});

// Whether an existing Secret holds the credentials of the class and mode, so
// that a destination can reuse it.
export const isCredentialsSecretFor = (
  secret: Secret,
  className: string,
  mode: string
) => {
  const labels = secret.metadata?.labels ?? {};
  return (
    labels[MONITORING_CLASS_SECRET_LABEL] === className &&
    labels[SECRET_DEFINITION_LABEL] === mode
  );
};

export const getCredentialsSecretName = (destination: MonitoringDestination) =>
  destination.spec.authentication?.credentialsSecretRef?.name;

// "Reason: message", or whichever of the two a condition reports.
export const describeCondition = ({
  reason,
  message,
}: {
  reason?: string;
  message?: string;
}) => [reason, message].filter(Boolean).join(': ');

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
  // Core holds the deletion until no Instance lists the destination.
  if (destination.metadata?.deletionTimestamp) {
    return { state: MonitoringDestinationState.Deleting, serverVersion };
  }
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

// Instance names per destination name, from one namespace's bindings.
export const groupInstancesByDestination = (
  bindings: MonitoringBinding[]
): Map<string, string[]> =>
  bindings.reduce((byDestination, binding) => {
    const destination = binding.spec.destinationRef.name;
    byDestination.set(destination, [
      ...(byDestination.get(destination) ?? []),
      binding.spec.instanceRef.name,
    ]);
    return byDestination;
  }, new Map<string, string[]>());
