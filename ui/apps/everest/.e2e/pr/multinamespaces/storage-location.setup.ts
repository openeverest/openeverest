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

import { test as setup } from '@playwright/test';
import 'dotenv/config';
import { createDbClusterFn } from '@e2e/utils/db-cluster';
import { EVEREST_CI_NAMESPACES } from '@e2e/constants';
import { waitForInitializingState } from '@e2e/utils/table';

const pgDbName = 'pr-mul-ns-db-pg';
const pxcDbName = 'pr-mul-ns-db-pxc';

setup.describe.parallel('Storage Locations setup', () => {
  setup('Create PG cluster', async ({ page, request }) => {
    await createDbClusterFn(
      request,
      {
        dbName: pgDbName,
        dbType: 'postgresql',
        numberOfNodes: '1',
      },
      EVEREST_CI_NAMESPACES.PG_ONLY
    );
    await page.goto('/databases');
    await waitForInitializingState(page, pgDbName);
  });

  setup('Create PXC cluster', async ({ request }) => {
    await createDbClusterFn(
      request,
      {
        dbName: pxcDbName,
        dbType: 'mysql',
        numberOfNodes: '1',
        backup: {
          enabled: false,
        },
      },
      EVEREST_CI_NAMESPACES.PXC_ONLY
    );
  });
});
