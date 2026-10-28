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

// Mirrors the server's CreateRestore check: restores are keyed by the target
// instance, and an in-place restore reads that instance's own backups. It keeps
// the instance's own credentials, unlike a clone (useCanCreateClusterFromBackup).
export const useCanRestore = (namespace: string, instanceName: string) => {
  const { canCreate } = useRBACPermissions(
    'restores',
    `${namespace}/${instanceName}`
  );
  const { canRead: canReadBackups } = useRBACPermissions(
    'backups',
    `${namespace}/${instanceName}`
  );

  return canCreate && canReadBackups;
};
