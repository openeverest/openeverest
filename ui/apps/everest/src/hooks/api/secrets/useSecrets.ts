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

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { AxiosError } from 'axios';
import { createSecretFn, deleteSecretFn, getSecretsFn } from 'api/secrets';
import { Secret, SecretListFilter } from 'shared-types/api.types';

export const SECRETS_QUERY_KEY = 'secrets';

// An in-use secret stays listed with a deletionTimestamp until its finalizer clears.
const dropTerminating = (secrets: Secret[]) =>
  secrets.filter((secret) => !secret.metadata?.deletionTimestamp);

export const useSecrets = (
  cluster: string,
  namespace: string,
  filter: SecretListFilter,
  options?: { enabled?: boolean }
) =>
  useQuery({
    queryKey: [SECRETS_QUERY_KEY, cluster, namespace, filter],
    queryFn: () => getSecretsFn(cluster, namespace, filter),
    select: dropTerminating,
    enabled: (options?.enabled ?? true) && !!cluster && !!namespace,
  });

export const useCreateSecret = (cluster: string, namespace: string) => {
  const queryClient = useQueryClient();

  return useMutation<Secret, AxiosError, Secret>({
    mutationFn: (secret) => createSecretFn(cluster, namespace, secret),
    // Don't keep the submitted secret values in the mutation cache.
    gcTime: 0,
    onSuccess: () =>
      queryClient.invalidateQueries({
        queryKey: [SECRETS_QUERY_KEY, cluster, namespace],
      }),
  });
};

export const useDeleteSecret = (cluster: string, namespace: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (name: string) => deleteSecretFn(cluster, namespace, name),
    onSuccess: () =>
      queryClient.invalidateQueries({
        queryKey: [SECRETS_QUERY_KEY, cluster, namespace],
      }),
  });
};
