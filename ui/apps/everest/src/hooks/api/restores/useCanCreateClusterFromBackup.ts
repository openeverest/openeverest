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

import { useMemo } from 'react';
import { useRBACAction, useRBACPermissions } from 'hooks/rbac';
import { useCanRestore } from './useCanRestore';

interface CanCreateClusterFromBackupOptions {
  hasSchedules?: boolean;
  // Destinations the source sends monitoring data to; the clone lists them
  // too, which needs read on each.
  monitoringDestinations?: string[];
}

const NO_DESTINATIONS: string[] = [];

// A user may clone an instance into a new database (restore-to-new-DB) when they
// can restore the source, read its connection details and create instances in
// its namespace. Single source of truth for this combined gate, shared by the
// instance and backup-row action menus.
export const useCanCreateClusterFromBackup = (
  namespace: string,
  instanceName: string,
  options: CanCreateClusterFromBackupOptions = {}
) => {
  const { hasSchedules = false, monitoringDestinations = NO_DESTINATIONS } =
    options;
  // Keyed by the names so a new array with the same destinations keeps the
  // permission check stable.
  const destinationsKey = monitoringDestinations.join(',');
  const destinationObjects = useMemo(
    () =>
      destinationsKey
        ? destinationsKey.split(',').map((name) => `${namespace}/${name}`)
        : NO_DESTINATIONS,
    [namespace, destinationsKey]
  );
  const canRestore = useCanRestore(namespace, instanceName);
  // The provider copies the source's users secret into the clone, so cloning
  // reveals the source credentials.
  const { isAllowed: canReadSourceConnection } = useRBACAction(
    'read-connection',
    'instances',
    `${namespace}/${instanceName}`
  );
  const { canCreate: canCreateClusters } = useRBACPermissions(
    'instances',
    `${namespace}/*`
  );
  const { canCreate: canCreateBackups } = useRBACPermissions(
    'backups',
    `${namespace}/${instanceName}`
  );
  const { canRead: canReadMonitoring } = useRBACPermissions(
    'monitoring-destinations',
    destinationObjects
  );

  let canCreateClusterFromBackup =
    canRestore && canReadSourceConnection && canCreateClusters;
  if (hasSchedules) {
    canCreateClusterFromBackup = canCreateClusterFromBackup && canCreateBackups;
  }
  if (destinationObjects.length > 0) {
    canCreateClusterFromBackup =
      canCreateClusterFromBackup && canReadMonitoring;
  }

  return canCreateClusterFromBackup;
};
