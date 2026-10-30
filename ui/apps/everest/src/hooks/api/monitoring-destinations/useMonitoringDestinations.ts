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
import { getMonitoringDestinationsFn } from 'api/monitoring-destinations';
import { useRBACPermissions } from 'hooks/rbac';
import { useClusterName } from 'hooks/api/useClusterName';
import {
  MonitoringDestination,
  MonitoringDestinationList,
} from 'shared-types/monitoring-destinations.types';
import { PerconaQueryOptions } from 'shared-types/query.types';

export const MONITORING_DESTINATIONS_QUERY_KEY = 'monitoring-destinations';

export const getMonitoringDestinationsQueryKey = (
  cluster: string,
  namespace: string
) => [MONITORING_DESTINATIONS_QUERY_KEY, cluster, namespace] as const;

export const useMonitoringDestinations = (
  namespace: string,
  options?: PerconaQueryOptions<
    MonitoringDestinationList,
    unknown,
    MonitoringDestination[]
  >
) => {
  const cluster = useClusterName();
  const { canRead } = useRBACPermissions(
    'monitoring-destinations',
    `${namespace}/*`
  );

  return useQuery<MonitoringDestinationList, unknown, MonitoringDestination[]>({
    queryKey: getMonitoringDestinationsQueryKey(cluster, namespace),
    queryFn: () => getMonitoringDestinationsFn(cluster, namespace),
    select: ({ items = [] }) => items,
    ...options,
    enabled: (options?.enabled ?? true) && canRead && !!namespace,
  });
};
