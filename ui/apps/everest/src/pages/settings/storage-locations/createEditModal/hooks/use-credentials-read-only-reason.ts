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

import { useSecret } from 'hooks/api/secrets';
import { useClusterName } from 'hooks/api/useClusterName';
import { useRBACPermissions } from 'hooks/rbac';
import { Messages } from '../../storage-locations.messages';

// The API patches only the Secrets it created (spec 020 §4.3).
const MANAGED_SECRET_LABEL = 'openeverest.io/managed';

// Why the keys of the storage's credentials can't be replaced in place, if
// they can't: the Secret can't be read, was saved outside OpenEverest, or the
// user may not update it.
export const useCredentialsReadOnlyReason = (
  namespace: string,
  secretName: string,
  enabled: boolean
): string | undefined => {
  const cluster = useClusterName();
  const { data: secret, isError } = useSecret(cluster, namespace, secretName, {
    enabled,
  });
  const { canUpdate, isPermissionsLoading } = useRBACPermissions(
    'secrets',
    `${namespace}/${secretName}`
  );

  if (!enabled) {
    return undefined;
  }
  if (isError) {
    return Messages.createEditModal.credentialsUnreadable(secretName);
  }
  if (secret && secret.metadata?.labels?.[MANAGED_SECRET_LABEL] !== 'true') {
    return Messages.createEditModal.credentialsManagedOutside;
  }
  if (!isPermissionsLoading && !canUpdate) {
    return Messages.createEditModal.credentialsNotUpdatable(secretName);
  }
  return undefined;
};
