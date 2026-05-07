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
  UseMutationOptions,
  useQuery,
} from '@tanstack/react-query';
import { deleteRestore, getInstanceRestores } from 'api/restores';
import { PerconaQueryOptions } from 'shared-types/query.types';
import { Restore, RestoreList } from 'shared-types/restores.types';

export const RESTORES_QUERY_KEY = 'restores';

export const getRestoreListQueryKey = (
  clusterName: string,
  namespace: string,
  instanceName: string
) => [RESTORES_QUERY_KEY, clusterName, namespace, instanceName] as const;

export const useDbClusterRestores = (
  clusterName: string,
  namespace: string,
  instanceName: string,
  options?: PerconaQueryOptions<RestoreList, unknown, Restore[]>
) => {
  return useQuery<RestoreList, unknown, Restore[]>({
    queryKey: getRestoreListQueryKey(clusterName, namespace, instanceName),
    queryFn: () => getInstanceRestores(clusterName, namespace, instanceName),
    select: ({ items = [] }) => items,
    refetchInterval: 5 * 1000,
    ...options,
    enabled: options?.enabled ?? true,
  });
};

export const useDeleteRestore = (
  clusterName: string,
  namespace: string,
  instanceName: string,
  options?: UseMutationOptions<unknown, unknown, string, unknown>
) =>
  useMutation({
    mutationFn: (restoreName: string) =>
      deleteRestore(clusterName, namespace, instanceName, restoreName),
    ...options,
  });
