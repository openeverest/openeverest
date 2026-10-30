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
} from 'utils/monitoring-destinations';
import { MonitoringDestinationRow } from './monitoring-destinations.types';

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

const intersect = (...lists: string[][]) =>
  lists.reduce((common, list) => common.filter((item) => list.includes(item)));

// Writing a destination may write its credentials Secret too, and needs the
// class.
export const toWritableNamespaces = ({
  destinationNamespaces,
  secretCreateNamespaces,
  secretReadNamespaces,
  canReadClasses,
}: {
  destinationNamespaces: string[];
  secretCreateNamespaces: string[];
  secretReadNamespaces: string[];
  canReadClasses: boolean;
}): string[] =>
  canReadClasses
    ? intersect(
        destinationNamespaces,
        secretCreateNamespaces,
        secretReadNamespaces
      )
    : [];
