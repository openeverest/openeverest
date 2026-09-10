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
  formValue: Record<string, unknown>;
};

type CreateInstanceSpec = NonNullable<CreateDbInstancePayload['spec']>;

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === 'object' && value !== null;

const parseDbWizardCore = (
  formValue: Record<string, unknown>
): { dbName: string; namespace: string } => {
  const dbName = formValue.dbName;
  const k8sNamespace = formValue.k8sNamespace;

  if (typeof dbName !== 'string' || dbName.length === 0) {
    throw new Error('Invalid create payload: dbName is required');
  }

  if (
    k8sNamespace !== undefined &&
    k8sNamespace !== null &&
    typeof k8sNamespace !== 'string'
  ) {
    throw new Error('Invalid create payload: k8sNamespace has invalid type');
  }

  return { dbName, namespace: k8sNamespace ?? '' };
};

export const buildCreateInstanceSpec = (
  formValue: Record<string, unknown>
): CreateInstanceSpec => {
  const { provider, dbName, k8sNamespace, spec, ...rest } = formValue;
  void dbName;
  void k8sNamespace;

  if (typeof provider !== 'string' || provider.length === 0) {
    throw new Error('Invalid create payload: provider is required');
  }

  const specRecord = isRecord(spec) ? spec : {};

  return {
    providerRef: { name: provider },
    ...deepMerge(rest, specRecord),
  };
};

export const useCreateDbInstance = (
  options?: UseMutationOptions<
    unknown,
    unknown,
    CreateInstanceHookArgType,
    unknown
  >
) =>
  useMutation({
    mutationFn: ({ formValue }: CreateInstanceHookArgType) => {
      const { dbName, namespace } = parseDbWizardCore(formValue);

      return createDbInstanceFn(
        'main',
        dbName,
        namespace,
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
