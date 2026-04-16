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
  useMutation,
  useQuery,
  UseMutationOptions,
} from '@tanstack/react-query';
import { createDbInstanceFn, getDbInstanceConnectionFn } from 'api/instanceApi';
import { DbWizardType } from 'pages/database-form/database-form-schema';
import { PerconaQueryOptions } from 'shared-types/query.types';
import {
  InstanceConnectionDetails,
  CreateDbInstancePayload,
  GetDbInstanceConnectionPayload,
} from 'shared-types/api.types';
import { deepMerge } from 'components/ui-generator/utils/object-path/object-path';

export const getDbInstanceCredentialsQueryKey = (
  dbInstanceName: string,
  namespace: string,
  clusterName: string
) => ['instance-credentials', namespace, clusterName, dbInstanceName] as const;

type CreateInstanceHookArgType = {
  formValue: DbWizardType;
};

type CreateInstanceSpec = NonNullable<CreateDbInstancePayload['spec']>;

export const buildCreateInstanceSpec = (
  formValue: DbWizardType
): CreateInstanceSpec => {
  const { provider, dbName, k8sNamespace, spec, ...rest } = formValue;
  void dbName;
  void k8sNamespace;

  return {
    provider,
    ...(deepMerge(
      rest as Record<string, unknown>,
      spec as unknown as Record<string, unknown>
    ) as Record<string, unknown>),
  };
};

export const useCreateDbInstance = (
  options?: UseMutationOptions<
    DbWizardType,
    unknown,
    CreateInstanceHookArgType,
    unknown
  >
) =>
  useMutation({
    mutationFn: ({ formValue }: CreateInstanceHookArgType) => {
      const { dbName, k8sNamespace } = formValue;

      return createDbInstanceFn(
        'main',
        dbName,
        k8sNamespace || '',
        buildCreateInstanceSpec(formValue)
      );
    },
    ...options,
  });

export const useDbInstanceCredentials = (
  dbInstanceName: string,
  namespace: string,
  options?: PerconaQueryOptions<
    GetDbInstanceConnectionPayload,
    unknown,
    InstanceConnectionDetails
  >
) => {
  // TODO implement RBAC
  // const { canRead: canReadCredentials } = useRBACPermissions(
  //     'database-instance-credentials',
  //     `${namespace}/${dbInstanceName}`
  //   );
  // TODO change to global use of cluster name during implementing multicluster feature
  const clusterName = 'main';

  return useQuery<
    GetDbInstanceConnectionPayload,
    unknown,
    InstanceConnectionDetails
  >({
    queryKey: getDbInstanceCredentialsQueryKey(
      dbInstanceName,
      namespace,
      clusterName
    ),
    queryFn: () =>
      getDbInstanceConnectionFn(clusterName, namespace, dbInstanceName),
    ...options,
    // select: canReadCredentials
    //   ? (creds) => creds
    //   : () => ({ username: '', password: '' }),
    // ...options,
    // enabled: (options?.enabled ?? true) && canReadCredentials,
  });
};

export default useCreateDbInstance;
