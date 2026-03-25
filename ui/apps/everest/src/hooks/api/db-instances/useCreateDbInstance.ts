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
import { createInstanceFn, getInstanceConnectionFn } from 'api/instanceApi';
import { DbWizardType } from 'pages/database-form/database-form-schema';
import { PerconaQueryOptions } from 'shared-types/query.types';
import { InstanceConnectionDetails, GetInstanceConnectionPayload } from 'types/api';

type CreateInstanceHookArgType = {
  formValue: DbWizardType;
};

export const useCreateInstance = (
  options?: UseMutationOptions<
    DbWizardType,
    unknown,
    CreateInstanceHookArgType,
    unknown
  >
) =>
  useMutation({
    mutationFn: ({
      formValue: { provider, dbName, k8sNamespace, spec, ...rest },
    }: CreateInstanceHookArgType) => {
      return createInstanceFn('main', dbName, k8sNamespace || '', {
        provider: provider || '',
        ...rest,
        ...spec,
      });
    },
    ...options,
  });

export const useDbInstanceCredentials = (
  dbInstanceName: string,
  namespace: string,
  options?: PerconaQueryOptions<
    GetInstanceConnectionPayload,
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
    GetInstanceConnectionPayload,
    unknown,
    InstanceConnectionDetails
  >({
    queryKey: ['instance-credentials', dbInstanceName],
    queryFn: () =>
      getInstanceConnectionFn(clusterName, namespace, dbInstanceName),
    ...options,
    // select: canReadCredentials
    //   ? (creds) => creds
    //   : () => ({ username: '', password: '' }),
    // ...options,
    // enabled: (options?.enabled ?? true) && canReadCredentials,
  });
};

export default useCreateInstance;
