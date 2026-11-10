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
  MonitoringBinding,
  MonitoringClass,
  MonitoringDestination,
} from 'shared-types/monitoring-destinations.types';
import {
  getMonitoringClassName,
  getMonitoringClassTitle,
} from 'utils/monitoring-classes';
import {
  getDestinationAddress,
  getDestinationName,
  getDestinationStatus,
  groupInstancesByDestination,
} from 'utils/monitoring-destinations';
import { MonitoringDestinationRow } from './monitoring-destinations.types';

export const toMonitoringDestinationRows = (
  namespaces: {
    namespace: string;
    destinations: MonitoringDestination[];
    bindings: MonitoringBinding[];
  }[],
  classes: MonitoringClass[]
): MonitoringDestinationRow[] =>
  namespaces.flatMap(({ namespace, destinations, bindings }) => {
    const instancesByDestination = groupInstancesByDestination(bindings);
    return destinations.map((destination) => {
      const name = getDestinationName(destination);
      const className = destination.spec.classRef.name;
      const monitoringClass = classes.find(
        (c) => getMonitoringClassName(c) === className
      );
      return {
        name,
        namespace,
        type: monitoringClass
          ? getMonitoringClassTitle(monitoringClass)
          : className,
        address: getDestinationAddress(destination, monitoringClass),
        status: getDestinationStatus(destination),
        usedBy: instancesByDestination.get(name) ?? [],
        raw: destination,
      };
    });
  });
