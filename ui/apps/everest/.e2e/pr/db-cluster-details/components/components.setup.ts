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

import { expect, test, test as setup } from '@playwright/test';
import { createDbClusterFn, getDbClusterAPI } from '@e2e/utils/db-cluster';
import { EVEREST_CI_NAMESPACES, TIMEOUTS } from '@e2e/constants';
import { getCITokenFromLocalStorage } from '@e2e/utils/localStorage';

const namespace = EVEREST_CI_NAMESPACES.PG_ONLY,
  dbClusterName = 'pr-db-det-comp';
let token: string;

setup.describe.serial('DB Cluster Overview setup', () => {
  test.describe.configure({ timeout: TIMEOUTS.FiveMinutes });

  setup(`Get token`, async ({}) => {
    token = await getCITokenFromLocalStorage();
    expect(token).not.toHaveLength(0);
  });

  setup(`Create DB cluster`, async ({ request }) => {
    await createDbClusterFn(
      request,
      {
        dbName: dbClusterName,
        dbType: 'postgresql',
        numberOfNodes: '3',
        numberOfProxies: '2',
      },
      namespace
    );
  });

  setup(`Wait for DB cluster creation`, async ({ request }) => {
    await expect(async () => {
      const dbCluster = await getDbClusterAPI(
        dbClusterName,
        namespace,
        request,
        token
      );
      expect(dbCluster.status.status === 'ready').toBeTruthy();
    }).toPass({
      intervals: [1000],
      timeout: TIMEOUTS.FiveMinutes,
    });
  });
});
