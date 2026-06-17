// everest
// Copyright (C) 2023 Percona LLC
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

import { test as teardown } from '@playwright/test';
import { getCITokenFromLocalStorage } from '../utils/localStorage';
import { getBucketNamespacesMap } from '../constants';
import {
  deleteMonitoringConfig,
  listMonitoringConfigs,
} from '@e2e/utils/monitoring-config';

teardown.describe.serial('Monitoring Configs teardown', () => {
  teardown('Delete Monitoring Configs', async ({ request }) => {
    const token = await getCITokenFromLocalStorage();
    const bucketNamespacesMap = getBucketNamespacesMap();
    const allNamespaces = Array.from(
      new Set(bucketNamespacesMap.map(([, namespaces]) => namespaces).flat())
    );
    const promises: Promise<any>[] = [];

    for (const [idx, namespace] of allNamespaces.entries()) {
      const monitoringResponse = await listMonitoringConfigs(
        request,
        namespace,
        token!
      );
      const monitoringConfigs = monitoringResponse?.items ?? [];
      for (const config of monitoringConfigs) {
        const name = config.metadata?.name;
        if (name) {
          promises.push(
            deleteMonitoringConfig(request, namespace, name, token!)
          );
        }
      }
    }
    await Promise.all(promises);
  });
});
