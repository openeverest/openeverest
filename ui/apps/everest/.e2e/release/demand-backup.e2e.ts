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

import { expect, test } from '@playwright/test';
import {
  deleteDbCluster,
  gotoDbClusterBackups,
  gotoDbClusterRestores,
} from '@e2e/utils/db-clusters-list';
import { getTokenFromLocalStorage } from '@e2e/utils/localStorage';
import { getClusterDetailedInfo } from '@e2e/utils/storage-class';
import {
  moveForward,
  submitWizard,
  populateBasicInformation,
  populateResources,
  populateAdvancedConfig,
} from '@e2e/utils/db-wizard';
import { EVEREST_CI_NAMESPACES } from '@e2e/constants';
import {
  waitForStatus,
  waitForDelete,
  findRowAndClickActions,
} from '@e2e/utils/table';
import { clickOnDemandBackup } from '@e2e/pr/db-cluster-details/utils';
import { prepareTestDB, dropTestDB, queryTestDB } from '@e2e/utils/db-cmd-line';
import { getDbClusterAPI } from '@e2e/utils/db-cluster';
import { shouldExecuteDBCombination } from '@e2e/utils/generic';

let token: string;

test.describe.configure({ retries: 0 });

const zephyrMap: Record<string, string> = {
  'backup-pxc': 'T101',
  'backup-psmdb': 'T102',
  'backup-postgresql': 'T103',
  'restore-pxc': 'T104',
  'restore-psmdb': 'T105',
  'restore-postgresql': 'T106',
};

[
  { db: 'psmdb', size: 3 },
  { db: 'pxc', size: 3 },
  { db: 'postgresql', size: 3 },
].forEach(({ db, size }) => {
  test.describe(
    'Demand backup',
    {
      tag: '@release',
    },
    () => {
      test.skip(!shouldExecuteDBCombination(db, size));
      test.describe.configure({ timeout: 720000 });

      const clusterName = `${db}-${size}-dembkp`;
      let zephyrId: string;

      let storageClasses = [];
      const namespace = EVEREST_CI_NAMESPACES.EVEREST_UI;
      const monitoringName = 'e2e-endpoint-0';
      const baseBackupName = `dembkp-${db}-${size}`;

      test.beforeAll(async ({ request }) => {
        token = await getTokenFromLocalStorage();

        const { storageClassNames = [] } = await getClusterDetailedInfo(
          token,
          request
        );
        storageClasses = storageClassNames;
      });

      test(`Cluster creation [${db} size ${size}]`, async ({
        page,
        request,
      }) => {
        expect(storageClasses.length).toBeGreaterThan(0);

        await page.goto('/databases');
        await page.getByTestId('add-db-cluster-button').waitFor();
        await page.getByTestId('add-db-cluster-button').click();
        await page.getByTestId(`add-db-cluster-button-${db}`).click();

        await test.step('Populate basic information', async () => {
          await populateBasicInformation(
            page,
            namespace,
            clusterName,
            db,
            storageClasses[0],
            false,
            null
          );
          await moveForward(page);
        });

        await test.step('Populate resources', async () => {
          await page
            .getByRole('button')
            .getByText(size + ' node')
            .click();
          await expect(page.getByText('Nodes (' + size + ')')).toBeVisible();
          await populateResources(page, 0.6, 1, 1, size);
          await moveForward(page);
        });

        await test.step('Populate backups', async () => {
          await moveForward(page);
        });

        await test.step('Populate advanced db config', async () => {
          await populateAdvancedConfig(page, db, false, '', true, '');
          await moveForward(page);
        });

        await test.step('Populate monitoring', async () => {
          await page.getByTestId('switch-input-monitoring').click();
          await page
            .getByTestId('text-input-monitoring-instance')
            .fill(monitoringName);
          await expect(
            page.getByTestId('text-input-monitoring-instance')
          ).toHaveValue(monitoringName);
        });

        await test.step('Submit wizard', async () => {
          await submitWizard(page);
        });

        // go to db list and check status
        await test.step('Check db list and status', async () => {
          await page.goto('/databases');
          await waitForStatus(page, clusterName, 'Initializing', 30000);
          await waitForStatus(page, clusterName, 'Up', 720000);
        });

        await test.step('Check db cluster k8s object options', async () => {
          const addedCluster = await getDbClusterAPI(
            clusterName,
            EVEREST_CI_NAMESPACES.EVEREST_UI,
            request,
            token
          );

          expect(addedCluster?.spec.engine.type).toBe(db);
          expect(addedCluster?.spec.engine.replicas).toBe(size);
          expect(['600m', '0.6']).toContain(
            addedCluster?.spec.engine.resources?.cpu.toString()
          );
          expect(addedCluster?.spec.engine.resources?.memory.toString()).toBe(
            '1G'
          );
          expect(addedCluster?.spec.engine.storage.size.toString()).toBe('1Gi');
          expect(addedCluster?.spec.proxy.expose.type).toBe('ClusterIP');
          if (db != 'psmdb') {
            expect(addedCluster?.spec.proxy.replicas).toBe(size);
          }
        });
      });

      test(`Add data [${db} size ${size}]`, async () => {
        await prepareTestDB(clusterName, namespace);
      });

      zephyrId = zephyrMap[`backup-${db}`];
      test(`${zephyrId} - Create demand backup [${db} size ${size}]`, async ({
        page,
      }) => {
        await gotoDbClusterBackups(page, clusterName);
        await clickOnDemandBackup(page);
        await page.getByTestId('text-input-name').fill(baseBackupName + '-1');
        await expect(page.getByTestId('text-input-name')).not.toBeEmpty();
        await expect(
          page.getByTestId('text-input-storage-location')
        ).not.toBeEmpty();
        await page.getByTestId('form-dialog-create').click();

        await waitForStatus(page, baseBackupName + '-1', 'Succeeded', 360000);
      });

      test(`Delete data [${db} size ${size}]`, async () => {
        await dropTestDB(clusterName, namespace);
      });

      zephyrId = zephyrMap[`restore-${db}`];
      test(`${zephyrId} - Restore cluster [${db} size ${size}]`, async ({
        page,
      }) => {
        await test.step('Navigate to backups and restore', async () => {
          await gotoDbClusterBackups(page, clusterName);
          await findRowAndClickActions(
            page,
            baseBackupName + '-1',
            'Restore to this DB'
          );
          await expect(
            page.getByTestId('select-input-backup-name')
          ).not.toBeEmpty();
          await page.getByTestId('form-dialog-restore').click();

          await page.goto('/databases');
          await waitForStatus(page, clusterName, 'Restoring', 30000);
          await waitForStatus(page, clusterName, 'Up', 660000);

          await gotoDbClusterRestores(page, clusterName);
          // we select based on backup source since restores cannot be named and we don't know
          // in advance what will be the name
          await waitForStatus(page, baseBackupName + '-1', 'Succeeded', 120000);
        });

        await test.step(`Check data after restore [${db} size ${size}]`, async () => {
          const result = await queryTestDB(clusterName, namespace);
          switch (db) {
            case 'pxc':
              expect(result.trim()).toBe('1\n2\n3');
              break;
            case 'psmdb':
              expect(result.trim()).toBe('[{"a":1},{"a":2},{"a":3}]');
              break;
            case 'postgresql':
              expect(result.trim()).toBe('1\n 2\n 3');
              break;
          }
        });
      });

      test(`Delete restore [${db} size ${size}]`, async ({ page }) => {
        await gotoDbClusterRestores(page, clusterName);
        await findRowAndClickActions(page, baseBackupName + '-1', 'Delete');
        await expect(page.getByLabel('Delete restore')).toBeVisible();
        await page.getByTestId('confirm-dialog-delete').click();
        await waitForDelete(page, baseBackupName + '-1', 15000);
      });

      test(`Delete backup [${db} size ${size}]`, async ({ page }) => {
        await gotoDbClusterBackups(page, clusterName);
        await findRowAndClickActions(page, baseBackupName + '-1', 'Delete');
        await expect(page.getByLabel('Delete backup')).toBeVisible();
        await page.getByTestId('form-dialog-delete').click();
        await waitForDelete(page, baseBackupName + '-1', 30000);
      });

      test(`Delete cluster [${db} size ${size}]`, async ({ page }) => {
        await deleteDbCluster(page, clusterName);
        await waitForStatus(page, clusterName, 'Deleting', 15000);
        await waitForDelete(page, clusterName, 240000);
      });
    }
  );
});
