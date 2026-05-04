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

import { MonitoringConfigForNamespaceResult } from 'hooks/api/monitoring/useMonitoringConfigsList';
import { MonitoringConfigTableElement } from './monitoring-endpoints.types';

export const convertMonitoringConfigsToTableFormat = (
  data: MonitoringConfigForNamespaceResult[]
): MonitoringConfigTableElement[] => {
  const result: MonitoringConfigTableElement[] = [];
  data.forEach((item) => {
    if (!item.queryResult.isSuccess) return;
    const rows = item.queryResult.data.map(
      (config): MonitoringConfigTableElement => ({
        name: config.metadata?.name ?? '',
        namespace: item.namespace,
        type: config.spec.type,
        url: config.spec.url,
        inUse: config.status?.inUse ?? false,
        pmmServerVersion: config.status?.pmmServerVersion ?? '-',
        raw: config,
      })
    );
    result.push(...rows);
  });
  return result;
};
