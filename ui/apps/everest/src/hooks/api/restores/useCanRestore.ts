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

// A user may restore an instance only if they can create restores and read the
// target's credentials (the restored database must be usable afterwards).
export const useCanRestore = (namespace: string, instanceName: string) => {
  const { canCreate } = useRBACPermissions(
    'database-cluster-restores',
    `${namespace}/${instanceName}`
  );
  const { canRead: canReadCredentials } = useRBACPermissions(
    'database-cluster-credentials',
    `${namespace}/${instanceName}`
  );

  return canCreate && canReadCredentials;
};
