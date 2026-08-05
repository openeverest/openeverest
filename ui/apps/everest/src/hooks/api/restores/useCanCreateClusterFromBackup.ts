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

import { useRBACPermissions } from 'hooks/rbac';
import { useCanRestore } from './useCanRestore';

// A user may clone an instance into a new database (restore-to-new-DB) when they
// can restore the source and create database clusters in its namespace. Single
// source of truth for this combined gate, shared by the instance and backup-row
// action menus.
export const useCanCreateClusterFromBackup = (
  namespace: string,
  instanceName: string
) => {
  const canRestore = useCanRestore(namespace, instanceName);
  const { canCreate: canCreateClusters } = useRBACPermissions(
    'database-clusters',
    `${namespace}/*`
  );

  return canRestore && canCreateClusters;
};
