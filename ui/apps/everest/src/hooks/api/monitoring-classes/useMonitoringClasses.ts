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
import { getMonitoringClassesFn } from 'api/monitoring-classes';
import { useRBACPermissions } from 'hooks/rbac';
import { useClusterName } from 'hooks/api/useClusterName';
import {
  MonitoringClass,
  MonitoringClassList,
} from 'shared-types/monitoring-destinations.types';
import { PerconaQueryOptions } from 'shared-types/query.types';
import { isMonitoringClassAccepted } from 'utils/monitoring-classes';

export const MONITORING_CLASSES_QUERY_KEY = 'monitoring-classes';

export const getMonitoringClassesQueryKey = (
  cluster: string,
  provider: string
) => [MONITORING_CLASSES_QUERY_KEY, cluster, provider] as const;

// Classes an Instance of the provider can send monitoring data to.
export const useMonitoringClassesForProvider = (
  provider: string,
  options?: PerconaQueryOptions<MonitoringClassList, unknown, MonitoringClass[]>
) => {
  const cluster = useClusterName();
  const { canRead } = useRBACPermissions('monitoring-classes');

  return useQuery<MonitoringClassList, unknown, MonitoringClass[]>({
    queryKey: getMonitoringClassesQueryKey(cluster, provider),
    queryFn: () => getMonitoringClassesFn(cluster, provider),
    select: ({ items = [] }) => items.filter(isMonitoringClassAccepted),
    ...options,
    enabled: (options?.enabled ?? true) && canRead && !!provider,
  });
};

// Every installed class, e.g. to name the class of each destination.
export const useMonitoringClasses = (
  options?: PerconaQueryOptions<MonitoringClassList, unknown, MonitoringClass[]>
) => {
  const cluster = useClusterName();
  const { canRead } = useRBACPermissions('monitoring-classes');

  return useQuery<MonitoringClassList, unknown, MonitoringClass[]>({
    queryKey: getMonitoringClassesQueryKey(cluster, ''),
    queryFn: () => getMonitoringClassesFn(cluster),
    select: ({ items = [] }) => items,
    enabled: (options?.enabled ?? true) && canRead,
    ...options,
  });
};
