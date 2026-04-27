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
import { getMonitoringConfigsFn } from 'api/monitoringConfigs';

export const MONITORING_CONFIGS_QUERY_KEY = 'monitoringConfigs';

export const useMonitoringConfigsList = (
  cluster: string,
  namespace: string,
  options?: { refetchInterval?: number; enabled?: boolean }
) => {
  return useQuery({
    queryKey: [MONITORING_CONFIGS_QUERY_KEY, cluster, namespace],
    queryFn: () => getMonitoringConfigsFn(cluster, namespace),
    refetchInterval: options?.refetchInterval ?? 5_000,
    enabled: options?.enabled ?? (!!namespace && !!cluster),
  });
};
