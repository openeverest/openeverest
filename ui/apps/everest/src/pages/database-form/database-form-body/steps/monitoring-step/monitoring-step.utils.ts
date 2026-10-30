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

import {
  InstanceMonitoringDestination,
  MonitoringClass,
  MonitoringDestination,
} from 'shared-types/monitoring-destinations.types';
import {
  MAX_MONITORING_DESTINATIONS,
  MONITORING_ENTRY_NAME_MAX_LENGTH,
} from './monitoring-step.constants';
import {
  DestinationDisabledReason,
  DestinationReadiness,
  MonitoringClassGroup,
} from './monitoring-step.types';

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === 'object' && value !== null && !Array.isArray(value);

const isInstanceMonitoringDestination = (
  value: unknown
): value is InstanceMonitoringDestination => {
  if (!isRecord(value) || typeof value.name !== 'string') {
    return false;
  }
  const ref = value.destinationRef;
  return isRecord(ref) && typeof ref.name === 'string';
};

export const parseMonitoringDestinations = (
  value: unknown
): InstanceMonitoringDestination[] =>
  Array.isArray(value) ? value.filter(isInstanceMonitoringDestination) : [];

export const getDestinationName = (destination: MonitoringDestination) =>
  destination.metadata?.name ?? '';

export const getMonitoringClassName = (monitoringClass: MonitoringClass) =>
  monitoringClass.metadata?.name ?? '';

export const getMonitoringClassTitle = (monitoringClass: MonitoringClass) =>
  monitoringClass.spec.displayName || getMonitoringClassName(monitoringClass);

// Keeps the order of the classes and lists every class, including those
// without a destination in the namespace yet.
export const groupDestinationsByClass = (
  classes: MonitoringClass[],
  destinations: MonitoringDestination[]
): MonitoringClassGroup[] =>
  classes.map((monitoringClass) => ({
    monitoringClass,
    destinations: destinations.filter(
      (destination) =>
        destination.spec.classRef.name ===
        getMonitoringClassName(monitoringClass)
    ),
  }));

export const getDestinationReadiness = (
  destination: MonitoringDestination
): DestinationReadiness => {
  const conditions = destination.status?.conditions ?? [];
  const accepted = conditions.find((c) => c.type === 'Accepted');
  const ready = conditions.find((c) => c.type === 'Ready');

  if (accepted?.status === 'False' || ready?.status === 'False') {
    return DestinationReadiness.NotReady;
  }
  if (ready?.status === 'True') {
    return DestinationReadiness.Ready;
  }
  return DestinationReadiness.Pending;
};

// The backend address, when the class keeps it in a url parameter.
export const getDestinationSummary = (
  destination: MonitoringDestination
): string | undefined => {
  const parameters: unknown = destination.spec.parameters;
  if (!isRecord(parameters)) {
    return undefined;
  }
  return typeof parameters.url === 'string' ? parameters.url : undefined;
};

const toDnsLabel = (name: string) =>
  name
    .toLowerCase()
    .replace(/[^a-z0-9-]+/g, '-')
    .slice(0, MONITORING_ENTRY_NAME_MAX_LENGTH)
    .replace(/^-+|-+$/g, '');

// Entry names default to the destination name, made a DNS label and unique
// among the Instance's entries.
export const toMonitoringEntryName = (
  destinationName: string,
  takenNames: string[]
): string => {
  const base = toDnsLabel(destinationName) || 'destination';
  if (!takenNames.includes(base)) {
    return base;
  }
  for (let index = 2; ; index++) {
    const suffix = `-${index}`;
    const candidate = `${base.slice(0, MONITORING_ENTRY_NAME_MAX_LENGTH - suffix.length).replace(/-+$/, '')}${suffix}`;
    if (!takenNames.includes(candidate)) {
      return candidate;
    }
  }
};

export const isDestinationSelected = (
  selected: InstanceMonitoringDestination[],
  destinationName: string
) => selected.some((entry) => entry.destinationRef.name === destinationName);

export const toggleMonitoringDestination = (
  selected: InstanceMonitoringDestination[],
  destinationName: string
): InstanceMonitoringDestination[] => {
  if (isDestinationSelected(selected, destinationName)) {
    return selected.filter(
      (entry) => entry.destinationRef.name !== destinationName
    );
  }
  return [
    ...selected,
    {
      name: toMonitoringEntryName(
        destinationName,
        selected.map((entry) => entry.name)
      ),
      destinationRef: { name: destinationName },
    },
  ];
};

// The integration a class is rendered through for the provider, when the
// provider renders it natively (e.g. PMM through spec.pmm of the engine CR).
const getProviderManagedIntegration = (
  monitoringClass: MonitoringClass,
  provider: string
): string | undefined => {
  const compatibility = monitoringClass.status?.providers?.find(
    (entry) => entry.name === provider
  );
  return compatibility?.mode === 'ProviderManaged'
    ? monitoringClass.spec.providerManaged?.integration
    : undefined;
};

// Why a destination cannot be added to the selection, if it cannot. Core
// keeps one ProviderManaged entry per integration, because the engine
// operator runs one agent of each kind.
export const getDestinationDisabledReason = ({
  destination,
  selected,
  destinations,
  classes,
  provider,
}: {
  destination: MonitoringDestination;
  selected: InstanceMonitoringDestination[];
  destinations: MonitoringDestination[];
  classes: MonitoringClass[];
  provider: string;
}): DestinationDisabledReason | undefined => {
  const name = getDestinationName(destination);
  if (isDestinationSelected(selected, name)) {
    return undefined;
  }
  if (selected.length >= MAX_MONITORING_DESTINATIONS) {
    return DestinationDisabledReason.LimitReached;
  }

  const integrationOf = (classRefName: string) => {
    const monitoringClass = classes.find(
      (c) => getMonitoringClassName(c) === classRefName
    );
    return monitoringClass
      ? getProviderManagedIntegration(monitoringClass, provider)
      : undefined;
  };

  const integration = integrationOf(destination.spec.classRef.name);
  if (!integration) {
    return undefined;
  }
  const integrationTaken = selected.some((entry) => {
    const selectedDestination = destinations.find(
      (d) => getDestinationName(d) === entry.destinationRef.name
    );
    return (
      !!selectedDestination &&
      integrationOf(selectedDestination.spec.classRef.name) === integration
    );
  });
  return integrationTaken
    ? DestinationDisabledReason.IntegrationTaken
    : undefined;
};

// Drops entries whose destination is not offered, e.g. after the namespace
// changed on an earlier step.
export const pruneMonitoringDestinations = (
  selected: InstanceMonitoringDestination[],
  offeredDestinationNames: string[]
): InstanceMonitoringDestination[] =>
  selected.filter((entry) =>
    offeredDestinationNames.includes(entry.destinationRef.name)
  );
