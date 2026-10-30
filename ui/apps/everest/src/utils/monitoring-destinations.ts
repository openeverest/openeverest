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
import {
  MonitoringClass,
  MonitoringDestination,
  MonitoringDestinationState,
  MonitoringDestinationStatus,
} from 'shared-types/monitoring-destinations.types';
import { getMonitoringClassUiSchema } from './monitoring-classes';

const ADDRESS_PARAMETER_PATH = 'parameters.url';
// The Ready reason a class controller reports before its first check.
const READY_PENDING_REASON = 'Pending';

export const getDestinationName = (destination: MonitoringDestination) =>
  destination.metadata?.name ?? '';

export const getDefaultCredentialsSecretName = (destinationName: string) =>
  `${destinationName}-credentials`;

export const getCredentialsSecretName = (destination: MonitoringDestination) =>
  destination.spec.authentication?.credentialsSecretRef?.name;

// The credentials Secret that goes with the destination on delete: the one
// the form creates by default, unless another destination of the namespace
// uses it too.
export const getSecretDeletedWithDestination = (
  destination: MonitoringDestination,
  namespaceDestinations: MonitoringDestination[]
): string | undefined => {
  const name = getDestinationName(destination);
  const secretName = getCredentialsSecretName(destination);
  const sharedWithAnother = namespaceDestinations.some(
    (other) =>
      getDestinationName(other) !== name &&
      getCredentialsSecretName(other) === secretName
  );
  return secretName === getDefaultCredentialsSecretName(name) &&
    !sharedWithAnother
    ? secretName
    : undefined;
};

export const getDestinationStatus = (
  destination: MonitoringDestination
): MonitoringDestinationStatus => {
  const conditions = destination.status?.conditions ?? [];
  const accepted = conditions.find((c) => c.type === 'Accepted');
  const ready = conditions.find((c) => c.type === 'Ready');
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
  const parameters: unknown = destination.spec.parameters;
  const url =
    typeof parameters === 'object' && parameters !== null
      ? Reflect.get(parameters, 'url')
      : undefined;
  return typeof url === 'string' && url ? url : undefined;
};
