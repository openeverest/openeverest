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
import { useImportBackupClasses } from 'hooks/api/backup-classes/useBackupClasses';
import { useClusterName } from 'hooks/api/useClusterName';
import { useNamespacePermissionsForResource } from 'hooks/rbac';

// Namespaces where the user can scan, see the found backups and create an instance from one.
export const useImportNamespaces = () => {
  const imports = useNamespacePermissionsForResource('backup-imports');
  const backups = useNamespacePermissionsForResource('backups');
  const instances = useNamespacePermissionsForResource('instances');
  const restores = useNamespacePermissionsForResource('restores');

  const namespaces = useMemo(
    () =>
      imports.canCreate.filter(
        (namespace) =>
          imports.canRead.includes(namespace) &&
          backups.canRead.includes(namespace) &&
          instances.canCreate.includes(namespace) &&
          restores.canCreate.includes(namespace)
      ),
    [
      imports.canCreate,
      imports.canRead,
      backups.canRead,
      instances.canCreate,
      restores.canCreate,
    ]
  );

  return {
    namespaces,
    isLoading:
      imports.isLoading ||
      imports.isPermissionsLoading ||
      backups.isPermissionsLoading ||
      instances.isPermissionsLoading ||
      restores.isPermissionsLoading,
  };
};

export const useImportAvailability = () => {
  const clusterName = useClusterName();
  const { namespaces, isLoading: isNamespacesLoading } = useImportNamespaces();
  const { data: importClasses, isLoading: isClassesLoading } =
    useImportBackupClasses(clusterName);
  const hasNamespaces = namespaces.length > 0;

  return {
    canImport: hasNamespaces && importClasses.length > 0,
    // Without a namespace to import into, the classes don't matter.
    isLoading: isNamespacesLoading || (hasNamespaces && isClassesLoading),
  };
};
