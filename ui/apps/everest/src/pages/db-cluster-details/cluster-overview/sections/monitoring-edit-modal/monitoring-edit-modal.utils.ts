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
  deepClone,
  deleteByPathAndEmptyParents,
  isPlainObject,
  setByPath,
} from 'components/ui-generator/utils/object-path';
import { Instance, Provider } from 'shared-types/api.types';
import {
  InstanceMonitoringDestination,
  InstanceMonitoringDestinationPayload,
  MonitoringClass,
  MonitoringDestination,
} from 'shared-types/monitoring-destinations.types';
import {
  getMonitoringClassName,
  getMonitoringClassTitle,
  restartsPods,
} from 'utils/monitoring-classes';
import { getDestinationName } from 'utils/monitoring-destinations';
import {
  MonitoringChange,
  MonitoringDisruption,
} from './monitoring-edit-modal.types';

const INSTANCE_MONITORING_DESTINATIONS_PATH = 'spec.monitoring.destinations';

type MonitoringEntry = Pick<InstanceMonitoringDestination, 'destinationRef'> & {
  parameters?: unknown;
};

const withSortedKeys = (_key: string, value: unknown) =>
  isPlainObject(value)
    ? Object.fromEntries(
        Object.entries(value).sort(([a], [b]) => a.localeCompare(b))
      )
    : value;

const isSameParameters = (a: unknown, b: unknown) =>
  JSON.stringify(a ?? {}, withSortedKeys) ===
  JSON.stringify(b ?? {}, withSortedKeys);

const toChanges = (before: MonitoringEntry[], after: MonitoringEntry[]) => {
  const toParameters = (entries: MonitoringEntry[]) =>
    new Map(entries.map((entry) => [entry.destinationRef.name, entry.parameters]));
  const was = toParameters(before);
  const now = toParameters(after);
  return [
    ...[...now.keys()]
      .filter((name) => !was.has(name))
      .map((name) => ({ name, change: MonitoringChange.Enable })),
    ...[...was.keys()]
      .filter((name) => !now.has(name))
      .map((name) => ({ name, change: MonitoringChange.Remove })),
    ...[...now]
      .filter(
        ([name, parameters]) =>
          was.has(name) && !isSameParameters(was.get(name), parameters)
      )
      .map(([name]) => ({ name, change: MonitoringChange.Update })),
  ];
};

// A rolling restart takes a component down when it runs a single replica.
export const hasSingleReplicaMonitoredComponent = (
  instance: Instance,
  provider: Provider | undefined
) => {
  const monitoredTypes = Object.keys(
    provider?.spec?.monitoring?.componentTypes ?? {}
  );
  return Object.values(instance.spec?.components ?? {}).some(
    ({ type, replicas }) =>
      !!type && monitoredTypes.includes(type) && replicas === 1
  );
};

// Any change to an entry of a class that rolls the pods for this provider
// restarts the database: adding, removing or changing its parameters.
export const getMonitoringDisruption = ({
  before,
  after,
  destinations,
  classes,
  provider,
  hasSingleReplica,
}: {
  before: MonitoringEntry[];
  after: MonitoringEntry[];
  destinations: MonitoringDestination[];
  classes: MonitoringClass[];
  provider: string;
  hasSingleReplica: boolean;
}): MonitoringDisruption => {
  const restarts = toChanges(before, after).flatMap(({ name, change }) => {
    const className = destinations.find(
      (destination) => getDestinationName(destination) === name
    )?.spec.classRef.name;
    const monitoringClass = classes.find(
      (c) => getMonitoringClassName(c) === className
    );
    return monitoringClass && restartsPods(monitoringClass, provider)
      ? [{ change, classTitle: getMonitoringClassTitle(monitoringClass) }]
      : [];
  });
  const uniqueRestarts = restarts.filter(
    (restart, index) =>
      restarts.findIndex(
        (other) =>
          other.change === restart.change &&
          other.classTitle === restart.classTitle
      ) === index
  );

  return {
    restarts: uniqueRestarts,
    singleReplica: uniqueRestarts.length > 0 && hasSingleReplica,
  };
};

// The generated Instance types keep entry parameters opaque, so the
// class-defined objects are written onto a copy of the Instance by path.
export const withMonitoringDestinations = (
  instance: Instance,
  destinations: InstanceMonitoringDestinationPayload[]
): Instance => {
  const updated = deepClone(instance);
  if (destinations.length > 0) {
    setByPath(updated, INSTANCE_MONITORING_DESTINATIONS_PATH, destinations);
  } else {
    deleteByPathAndEmptyParents(updated, INSTANCE_MONITORING_DESTINATIONS_PATH);
  }
  return updated;
};
