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
import { SECRET_DEFINITION_LABEL } from 'components/ui-generator/secret-field/secret-create-modal/secret-create-modal.constants';
import { getByPath } from 'components/ui-generator/utils/object-path';
import { Instance, Secret } from 'shared-types/api.types';
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

export const getInstanceMonitoringDestinationNames = (
  instance: Instance
): string[] =>
  (instance.spec?.monitoring?.destinations ?? []).map(
    (entry) => entry.destinationRef.name
  );

const CREDENTIALS_SECRET_PREFIX = 'monitoring-destination-';
const CREDENTIALS_SECRET_SUFFIX = '-credentials';
const MAX_SECRET_NAME_LENGTH = 253;
// Names the credentials Secret the UI created for the destination. A Secret
// the user brought is never recorded here, so it is never deleted.
const UI_CREATED_SECRET_ANNOTATION =
  'monitoring.openeverest.io/ui-created-credentials-secret';

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

// A fresh name for rotated credentials, so the destination can move to the
// new Secret before the old one goes.
export const getRotatedCredentialsSecretName = (
  destinationName: string,
  suffix = `${Date.now().toString(36)}${generateShortUID()}`
) => {
  const base = getDefaultCredentialsSecretName(destinationName);
  return `${base.slice(0, MAX_SECRET_NAME_LENGTH - suffix.length - 1)}-${suffix}`;
};

export const getCredentialsSecretName = (destination: MonitoringDestination) =>
  destination.spec.authentication?.credentialsSecretRef?.name;

// The destination's credentials Secret, when the UI created it.
export const getUiCreatedCredentialsSecretName = (
  destination: MonitoringDestination
): string | undefined => {
  const secretName = getCredentialsSecretName(destination);
  return secretName &&
    destination.metadata?.annotations?.[UI_CREATED_SECRET_ANNOTATION] ===
      secretName
    ? secretName
    : undefined;
};

// Records the Secret the UI created for the destination, or clears the record.
export const withUiCreatedCredentialsSecret = <
  Metadata extends { annotations?: Record<string, string> },
>(
  metadata: Metadata,
  secretName: string | undefined
): Metadata => {
  const annotations = Object.fromEntries(
    Object.entries(metadata.annotations ?? {}).filter(
      ([key]) => key !== UI_CREATED_SECRET_ANNOTATION
    )
  );
  if (secretName) {
    annotations[UI_CREATED_SECRET_ANNOTATION] = secretName;
  }
  return {
    ...metadata,
    annotations: Object.keys(annotations).length > 0 ? annotations : undefined,
  };
};

export const isSecretUsedByOtherDestination = (
  destinationName: string,
  secretName: string,
  namespaceDestinations: MonitoringDestination[]
) =>
  namespaceDestinations.some(
    (other) =>
      getDestinationName(other) !== destinationName &&
      getCredentialsSecretName(other) === secretName
  );

// The credentials Secret that goes with the destination on delete: one the UI
// created that no other destination of the namespace references.
export const getSecretDeletedWithDestination = (
  destination: MonitoringDestination,
  namespaceDestinations: MonitoringDestination[]
): string | undefined => {
  const secretName = getUiCreatedCredentialsSecretName(destination);
  return secretName &&
    !isSecretUsedByOtherDestination(
      getDestinationName(destination),
      secretName,
      namespaceDestinations
    )
    ? secretName
    : undefined;
};

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
