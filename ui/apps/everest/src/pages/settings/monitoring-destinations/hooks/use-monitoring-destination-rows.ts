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

import { useMonitoringBindingsForNamespaces } from 'hooks/api/monitoring-bindings';
import { useMonitoringClasses } from 'hooks/api/monitoring-classes';
import { useMonitoringDestinationsForNamespaces } from 'hooks/api/monitoring-destinations';
import { useNamespacePermissionsForResource } from 'hooks/rbac';
import { MonitoringClass } from 'shared-types/monitoring-destinations.types';
import { toMonitoringDestinationRows } from '../monitoring-destinations.utils';

const NO_CLASSES: MonitoringClass[] = [];

// The destinations of every namespace the user can read, with the databases
// using each where the user can read the bindings.
export const useMonitoringDestinationRows = () => {
  const destinationPermissions = useNamespacePermissionsForResource(
    'monitoring-destinations'
  );
  const bindingPermissions = useNamespacePermissionsForResource(
    'monitoring-bindings'
  );
  const { data: classes = NO_CLASSES } = useMonitoringClasses();

  const readNamespaces = destinationPermissions.canRead;
  const destinationResults =
    useMonitoringDestinationsForNamespaces(readNamespaces);
  const bindingResults = useMonitoringBindingsForNamespaces(
    bindingPermissions.canRead
  );

  const rows = toMonitoringDestinationRows(
    readNamespaces.map((namespace, index) => ({
      namespace,
      destinations: destinationResults[index]?.data ?? [],
      bindings:
        bindingResults[bindingPermissions.canRead.indexOf(namespace)]?.data ??
        [],
    })),
    classes
  );

  return {
    rows,
    classes,
    isLoading:
      destinationPermissions.isPermissionsLoading ||
      destinationResults.some((result) => result.isLoading),
  };
};
