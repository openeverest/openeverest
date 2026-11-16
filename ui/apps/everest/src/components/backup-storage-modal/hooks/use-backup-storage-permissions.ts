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
  useNamespacePermissionsForResource,
  useRBACPermissions,
} from 'hooks/rbac';
import { BackupStoragePermissions } from '../backup-storage-modal.types';

// Adding a storage needs its class, so a user who can't read storage classes
// adds none. Secret permissions are checked by the form.
export const useBackupStoragePermissions = (): BackupStoragePermissions => {
  const { canCreate, isPermissionsLoading: isCreateLoading } =
    useNamespacePermissionsForResource('backup-storages');
  const { canRead: canReadClasses, isPermissionsLoading: isClassesLoading } =
    useRBACPermissions('backup-storage-classes');

  return {
    createNamespaces: canReadClasses ? canCreate : [],
    isLoading: isCreateLoading || isClassesLoading,
  };
};
