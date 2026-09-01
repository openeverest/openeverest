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

import { expect, test } from '@playwright/test';
import { getCITokenFromLocalStorage } from '@e2e/utils/localStorage';
import { getClusterDetailedInfo } from '@e2e/utils/storage-class';
import { EVEREST_CI_NAMESPACES } from '@e2e/constants';
import {
  giveUserAdminPermissions,
  setRBACPermissionsK8S,
} from '@e2e/utils/rbac-cmd-line';
import { deleteDbClusterFn } from '@e2e/utils/db-cluster';
import { createDbWithParameters } from '@e2e/utils/rbac';
import { clickAddDbClusterBtn } from '@e2e/pr/db-cluster/db-wizard/db-wizard-utils';

let token: string;

test.describe.configure({ retries: 0 });

test.describe(
  'Create db clusters without backup schedules and monitoring',
  {
    tag: '@release',
  },
  () => {
    test.describe.configure({ timeout: 720000 });

    let storageClasses = [];
    const namespace2 = EVEREST_CI_NAMESPACES.EVEREST_UI;
    const namespace1 = EVEREST_CI_NAMESPACES.PXC_ONLY;
    const pxcDb = 'create-pxc-0';
    const psmdbDb = 'create-psmdb-0';
    const pgDb = 'create-pg-0';

    test.beforeAll(async ({ request }) => {
      token = await getCITokenFromLocalStorage();

      const { storageClassNames = [] } = await getClusterDetailedInfo(
        token,
        request
      );
      storageClasses = storageClassNames;
      await giveUserAdminPermissions();
    });

    test.beforeEach(async ({ page }) => {
      await page.goto('/databases');
    });

    // step 4
    [
      { db: 'psmdb', namespace: EVEREST_CI_NAMESPACES.PSMDB_ONLY },
      { db: 'pxc', namespace: EVEREST_CI_NAMESPACES.PXC_ONLY },
      { db: 'postgresql', namespace: EVEREST_CI_NAMESPACES.PG_ONLY },
    ].forEach(({ db, namespace }) => {
      test(`User can create ${db} databases in all available namespaces`, async ({
        page,
      }) => {
        await setRBACPermissionsK8S([
          ['namespaces', 'read', '*'],
          ['pod-scheduling-policies', 'read', '*'],
          ['database-engines', 'read', '*/*'],
          ['database-clusters', 'read', `*/*`],
          ['database-clusters', 'create', `*/*`],
        ]);
        expect(storageClasses.length).toBeGreaterThan(0);
        await clickAddDbClusterBtn(page, db);
        await page.getByTestId('text-input-k8s-namespace').click();

        const namespaceOptions = page.getByRole('option');
        expect(
          namespaceOptions.filter({ hasText: EVEREST_CI_NAMESPACES.EVEREST_UI })
        ).toBeVisible();
        expect(namespaceOptions.filter({ hasText: namespace })).toBeVisible();
      });
    });

    // step 5
    [
      { db: 'pxc', namespace: namespace1, dbName: pxcDb },
      { db: 'psmdb', namespace: namespace2, dbName: psmdbDb },
    ].forEach(({ db, namespace, dbName }) => {
      test(`Create a ${db} db without backup schedules or monitoring`, async ({
        page,
      }) => {
        expect(storageClasses.length).toBeGreaterThan(0);
        await createDbWithParameters({
          page: page,
          dbType: db,
          dbName: dbName,
          namespace: namespace,
          storageClasses: storageClasses,
        });
      });
    });

    // step 6
    test(`User can only access and create databases in ${namespace2} namespace`, async ({
      page,
    }) => {
      await setRBACPermissionsK8S([
        ['namespaces', 'read', '*'],
        ['pod-scheduling-policies', 'read', '*'],
        ['database-engines', 'read', '*/*'],
        ['database-clusters', 'read', `${namespace2}/*`],
        ['database-clusters', 'create', `${namespace2}/*`],
      ]);

      await page.reload({ waitUntil: 'networkidle' });

      await expect(page.getByText(pxcDb)).not.toBeVisible();
      await expect(page.getByText(psmdbDb)).toBeVisible();

      await clickAddDbClusterBtn(page, 'pxc');
      await page.getByTestId('text-input-k8s-namespace').click();

      const namespaceOptions = page.getByRole('option');
      expect(namespaceOptions.filter({ hasText: namespace2 })).toBeVisible();
      expect(
        namespaceOptions.filter({ hasText: namespace1 })
      ).not.toBeVisible();
    });

    // step 7
    test(`Create a pg db without schedules and monitoring in ${namespace2} namespace`, async ({
      page,
    }) => {
      expect(storageClasses.length).toBeGreaterThan(0);
      await createDbWithParameters({
        page: page,
        dbType: 'postgresql',
        dbName: pgDb,
        namespace: namespace2,
        storageClasses: storageClasses,
      });
    });

    test(`User cannot access databases in ${namespace1} namespace`, async ({
      page,
    }) => {
      await expect(page.getByText(pxcDb)).not.toBeVisible();
      await expect(page.getByText(psmdbDb)).toBeVisible();
      await expect(page.getByText(pgDb)).toBeVisible();
    });

    test('Delete databases', async ({ page, request }) => {
      await giveUserAdminPermissions();
      await deleteDbClusterFn(request, pxcDb, namespace1);
      await deleteDbClusterFn(request, psmdbDb, namespace2);
      await deleteDbClusterFn(request, pgDb, namespace2);

      await expect(page.getByText(pxcDb)).not.toBeVisible({ timeout: 70000 });
      await expect(page.getByText(psmdbDb)).not.toBeVisible({ timeout: 70000 });
      await expect(page.getByText(pgDb)).not.toBeVisible({ timeout: 70000 });
    });
  }
);

test.describe(
  'Create db clusters with backup schedules and monitoring',
  {
    tag: '@release',
  },
  () => {
    test.describe.configure({ timeout: 720000 });

    let storageClasses = [];
    const namespace2 = EVEREST_CI_NAMESPACES.EVEREST_UI;
    const namespace1 = EVEREST_CI_NAMESPACES.PXC_ONLY;
    const pxcDb = 'create-pxc-1';
    const psmdbDb = 'create-psmdb-1';

    test.beforeAll(async ({ request }) => {
      token = await getCITokenFromLocalStorage();

      const { storageClassNames = [] } = await getClusterDetailedInfo(
        token,
        request
      );
      storageClasses = storageClassNames;
    });

    test.beforeEach(async ({ page }) => {
      await page.goto('/databases');
    });

    // step 5
    [
      { db: 'psmdb', namespace: EVEREST_CI_NAMESPACES.PSMDB_ONLY },
      { db: 'pxc', namespace: EVEREST_CI_NAMESPACES.PXC_ONLY },
      { db: 'postgresql', namespace: EVEREST_CI_NAMESPACES.PG_ONLY },
    ].forEach(({ db, namespace }) => {
      test(`User can create ${db} databases in all available namespaces`, async ({
        page,
      }) => {
        await setRBACPermissionsK8S([
          ['namespaces', 'read', '*'],
          ['pod-scheduling-policies', 'read', '*'],
          ['database-engines', 'read', '*/*'],
          ['database-clusters', 'read', `*/*`],
          ['database-clusters', 'create', `*/*`],
          ['backup-storages', 'read', `*/*`],
          ['database-cluster-backups', 'create', `*/*`],
          ['monitoring-instances', 'read', `*/*`],
        ]);
        expect(storageClasses.length).toBeGreaterThan(0);
        await clickAddDbClusterBtn(page, db);
        await page.getByTestId('text-input-k8s-namespace').click();

        const namespaceOptions = page.getByRole('option');
        expect(
          namespaceOptions.filter({ hasText: EVEREST_CI_NAMESPACES.EVEREST_UI })
        ).toBeVisible();
        expect(namespaceOptions.filter({ hasText: namespace })).toBeVisible();
      });
    });

    // step 6
    test(`Create a pxc db with backup schedules or monitoring in ${namespace1}`, async ({
      page,
    }) => {
      const db = 'pxc';
      expect(storageClasses.length).toBeGreaterThan(0);
      await createDbWithParameters({
        page: page,
        dbType: db,
        dbName: pxcDb,
        namespace: namespace1,
        storageClasses: storageClasses,
        addBackupSchedule: true,
        addMonitoring: true,
      });
    });

    // step 10
    test(`User can only access and create databases in ${namespace2} namespace`, async ({
      page,
    }) => {
      await setRBACPermissionsK8S([
        ['namespaces', 'read', '*'],
        ['pod-scheduling-policies', 'read', '*'],
        ['database-engines', 'read', '*/*'],
        ['database-clusters', 'read', `${namespace2}/*`],
        ['database-clusters', 'create', `${namespace2}/*`],
        ['backup-storages', 'read', `${namespace2}/*`],
        ['database-cluster-backups', 'create', `${namespace2}/*`],
        ['monitoring-instances', 'read', `${namespace2}/*`],
      ]);
      const dbType = 'psmdb';
      const dbName = psmdbDb;

      await test.step(`Create a psmdb test with backup schedules and monitoring in ${namespace2}`, async () => {
        await createDbWithParameters({
          page: page,
          dbType: dbType,
          dbName: dbName,
          namespace: namespace2,
          storageClasses: storageClasses,
          addBackupSchedule: true,
          addMonitoring: true,
        });
      });

      await test.step(`User should not be able to add a backup storage and monitoring instance in ${namespace2} namespace`, async () => {
        await page.goto('/settings/storage-locations');
        await expect(page.getByRole('table')).toBeVisible();
        await expect(page.getByTestId('add-backup-storage')).not.toBeVisible();
        await expect(
          page.getByRole('row').filter({ hasText: 'bucket-1' })
        ).toBeVisible();

        await page.goto('/settings/monitoring-endpoints');
        await expect(page.getByRole('table')).toBeVisible();
        await expect(
          page.getByRole('row').filter({ hasText: 'e2e-endpoint-0' })
        ).toBeVisible();
        await expect(
          page.getByTestId('add-monitoring-endpoint')
        ).not.toBeVisible();
      });

      await test.step(`User should not be able to access databases in ${namespace1} namespace`, async () => {
        await page.goto('/databases');
        await expect(page.getByText(`${namespace1}`)).not.toBeVisible();
      });
    });

    test('Delete databases', async ({ page, request }) => {
      await giveUserAdminPermissions();
      await deleteDbClusterFn(request, pxcDb, namespace1);
      await deleteDbClusterFn(request, psmdbDb, namespace2);

      await expect(page.getByText(pxcDb)).not.toBeVisible({ timeout: 70000 });
      await expect(page.getByText(psmdbDb)).not.toBeVisible({
        timeout: 70000,
      });
    });
  }
);
