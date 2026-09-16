// everest
// Copyright (C) 2023 Percona LLC
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

import { expect, test as setup } from '@playwright/test';
import 'dotenv/config';
import { getCITokenFromLocalStorage } from '../utils/localStorage';
import {
  EVEREST_CI_CLUSTER,
  EVEREST_CI_NAMESPACES,
  getBucketNamespacesMap,
} from '../constants';

const { MONITORING_URL, MONITORING_USER, MONITORING_PASSWORD } = process.env;

const postMonitoringConfig = async (
  request: any,
  token: string,
  namespace: string,
  name: string
) =>
  request.post(
    `/v1/clusters/${EVEREST_CI_CLUSTER}/namespaces/${namespace}/monitoring-configs`,
    {
      data: {
        name,
        type: 'pmm',
        url: MONITORING_URL,
        verifyTLS: false,
        pmm: {
          user: MONITORING_USER,
          password: MONITORING_PASSWORD,
        },
      },
      headers: {
        Authorization: `Bearer ${token}`,
      },
    }
  );

setup.describe.serial('Monitoring Instance setup', () => {
  setup('Create Monitoring instances', async ({ request }) => {
    const token = await getCITokenFromLocalStorage();
    const bucketNamespacesMap = getBucketNamespacesMap();
    const allNamespaces = Array.from(
      new Set(bucketNamespacesMap.map(([, namespaces]) => namespaces).flat())
    );
    const promises: Promise<any>[] = [];

    // For the sake of simplicity, we will create a monitoring endpoint for all namespaces in the buckets we defined
    for (const [idx, namespace] of allNamespaces.entries()) {
      promises.push(
        (async () => {
          const name = `e2e-endpoint-${idx}`;
          let response = await postMonitoringConfig(
            request,
            token,
            namespace,
            name
          );

          if (response.status() === 404) {
            const body = await response.json();
            const isMissingNamespace =
              body?.message && body.message.includes('not found');

            if (isMissingNamespace) {
              response = await postMonitoringConfig(
                request,
                token,
                EVEREST_CI_NAMESPACES.EVEREST_UI,
                name
              );
            }
          }

          expect([200, 201, 409]).toContain(response.status());
        })()
      );
    }
    await Promise.all(promises);
  });
});
