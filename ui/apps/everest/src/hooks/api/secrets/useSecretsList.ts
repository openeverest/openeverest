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

import { useQuery } from '@tanstack/react-query';
import { getSecretsFn } from 'api/secretApi';
import { useRBACPermissions } from 'hooks/rbac';
import { SecretList } from 'shared-types/api.types';

export const SECRETS_QUERY_KEY = 'secrets';

export const getSecretsQueryKey = (clusterName: string, namespace: string) =>
  [SECRETS_QUERY_KEY, clusterName, namespace] as const;

// Lists OpenEverest-managed Secrets only, and only their metadata.
export const useSecretsList = (clusterName: string, namespace: string) => {
  const { canRead } = useRBACPermissions('secrets', `${namespace}/*`);

  return useQuery<SecretList, unknown, SecretList['items']>({
    queryKey: getSecretsQueryKey(clusterName, namespace),
    queryFn: () => getSecretsFn(clusterName, namespace),
    select: ({ items }) => items,
    enabled: canRead,
  });
};
