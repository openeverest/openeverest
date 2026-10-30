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
  setByPath,
} from 'components/ui-generator/utils/object-path';
import { Instance } from 'shared-types/api.types';
import {
  InstanceMonitoringDestinationPayload,
  MonitoringClass,
  MonitoringDestination,
} from 'shared-types/monitoring-destinations.types';
import {
  getMonitoringClassName,
  getMonitoringClassTitle,
} from 'utils/monitoring-classes';
import { getDestinationName } from 'utils/monitoring-destinations';
import {
  MonitoringChange,
  MonitoringDisruption,
} from './monitoring-edit-modal.types';

const INSTANCE_MONITORING_DESTINATIONS_PATH = 'spec.monitoring.destinations';

const restartsPods = (monitoringClass: MonitoringClass, provider: string) =>
  monitoringClass.status?.providers?.some(
    (entry) => entry.name === provider && entry.disruption === 'RollingRestart'
  ) ?? false;

// Adding or removing a destination whose class rolls the pods for this
// provider restarts the database.
export const getMonitoringDisruption = ({
  before,
  after,
  destinations,
  classes,
  provider,
  engineReplicas,
}: {
  before: string[];
  after: string[];
  destinations: MonitoringDestination[];
  classes: MonitoringClass[];
  provider: string;
  engineReplicas: number | undefined;
}): MonitoringDisruption => {
  const changes = [
    ...after
      .filter((name) => !before.includes(name))
      .map((name) => ({ name, change: MonitoringChange.Enable })),
    ...before
      .filter((name) => !after.includes(name))
      .map((name) => ({ name, change: MonitoringChange.Remove })),
  ];

  const restarts = changes.flatMap(({ name, change }) => {
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
    singleReplica: uniqueRestarts.length > 0 && engineReplicas === 1,
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
