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

import { useCallback } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import {
  createMonitoringDestinationFn,
  deleteMonitoringDestinationFn,
  updateMonitoringDestinationFn,
} from 'api/monitoring-destinations';
import {
  createSecretFn,
  deleteSecretFn,
  getSecretFn,
  patchSecretFn,
} from 'api/secrets';
import { MONITORING_BINDINGS_QUERY_KEY } from 'hooks/api/monitoring-bindings';
import { SECRETS_QUERY_KEY } from 'hooks/api/secrets';
import { useClusterName } from 'hooks/api/useClusterName';
import { MonitoringDestination } from 'shared-types/monitoring-destinations.types';
import { isDestinationInUseError } from 'utils/monitoring-destinations';
import {
  MonitoringDestinationRequests,
  saveMonitoringDestination,
  SaveMonitoringDestinationRequest,
} from './monitoring-destination-requests';
import { MONITORING_DESTINATIONS_QUERY_KEY } from './useMonitoringDestinations';

const toRequests = (
  cluster: string,
  namespace: string
): MonitoringDestinationRequests => ({
  createSecret: (secret) => createSecretFn(cluster, namespace, secret),
  getSecret: (name, quiet = false) =>
    getSecretFn(cluster, namespace, name, quiet),
  patchSecret: (name, patch) => patchSecretFn(cluster, namespace, name, patch),
  deleteSecret: (name) => deleteSecretFn(cluster, namespace, name),
  createDestination: (destination) =>
    createMonitoringDestinationFn(cluster, namespace, destination),
  updateDestination: (destination) =>
    updateMonitoringDestinationFn(cluster, namespace, destination),
});

// Settled, not only successful: a failed save may still have touched the
// Secret or the destination.
const useInvalidateNamespace = () => {
  const cluster = useClusterName();
  const queryClient = useQueryClient();

  return (namespace: string) =>
    Promise.all(
      [
        MONITORING_DESTINATIONS_QUERY_KEY,
        MONITORING_BINDINGS_QUERY_KEY,
        SECRETS_QUERY_KEY,
      ].map((key) =>
        queryClient.invalidateQueries({ queryKey: [key, cluster, namespace] })
      )
    );
};

export const useSaveMonitoringDestination = () => {
  const cluster = useClusterName();
  const invalidateNamespace = useInvalidateNamespace();

  return useMutation<
    MonitoringDestination,
    unknown,
    SaveMonitoringDestinationRequest
  >({
    mutationFn: (request) =>
      saveMonitoringDestination(
        request,
        toRequests(cluster, request.namespace)
      ),
    // Don't keep the submitted credentials in the mutation cache.
    gcTime: 0,
    onSettled: (_data, _error, { namespace }) => invalidateNamespace(namespace),
  });
};

interface DeleteMonitoringDestinationVariables {
  namespace: string;
  name: string;
}

// The credentials Secret stays: other destinations or users may hold it.
export const useDeleteMonitoringDestination = () => {
  const cluster = useClusterName();
  const invalidateNamespace = useInvalidateNamespace();

  return useMutation<void, unknown, DeleteMonitoringDestinationVariables>({
    mutationFn: async ({ namespace, name }) => {
      await deleteMonitoringDestinationFn(cluster, namespace, name);
    },
    onSettled: (_data, _error, { namespace }) => invalidateNamespace(namespace),
  });
};

// Whether the server would delete the destination now: a dry run, so nothing
// is deleted. Instances still listing it make the answer false.
export const useCanDeleteMonitoringDestination = () => {
  const cluster = useClusterName();

  return useCallback(
    (namespace: string, name: string): Promise<boolean> =>
      deleteMonitoringDestinationFn(cluster, namespace, name, true).then(
        () => true,
        (error: unknown) => {
          if (isDestinationInUseError(error)) {
            return false;
          }
          throw error;
        }
      ),
    [cluster]
  );
};
