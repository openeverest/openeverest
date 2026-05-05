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
import {
  queryPSMDB,
  prepareMongoDBTestDB,
  validateMongoDBSharding,
  configureMongoDBSharding,
  dropTestDB,
  queryTestDB,
} from '@e2e/utils/db-cmd-line';
import { getDbClusterAPI } from '@e2e/utils/db-cluster';
import { checkDBMetrics, checkQAN } from '@e2e/utils/monitoring-instance';
import { shouldExecuteDBCombination } from '@e2e/utils/generic';

let token: string;

const db = 'psmdb';
const size = 3;

test.describe.configure({ retries: 0 });

test.describe(
  'PSMDB sharding demand backup',
  {
    tag: '@release',
  },
  () => {
    test.skip(!shouldExecuteDBCombination(db, size));
    test.describe.configure({ timeout: 720000 });

    const clusterName = `${db}-${size}-shard`;

    let storageClasses = [];
    const namespace = EVEREST_CI_NAMESPACES.EVEREST_UI;
    const monitoringName = 'e2e-endpoint-0';
    const baseBackupName = `shard-${db}-${size}`;

    test.beforeAll(async ({ request }) => {
      token = await getTokenFromLocalStorage();

      const { storageClassNames = [] } = await getClusterDetailedInfo(
        token,
        request
      );
      storageClasses = storageClassNames;
    });

    test(`Cluster creation [${db} size ${size}]`, async ({ page, request }) => {
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
      });

      await test.step('Activate sharding', async () => {
        const shardingCheckbox = page.getByTestId('switch-input-sharding');
        await shardingCheckbox.click();
        await page.getByTestId('db-wizard-continue-button').isEnabled();
        await expect(
          page.getByTestId('db-wizard-continue-button')
        ).not.toBeDisabled();

        await moveForward(page);
      });

      await test.step('Populate resources', async () => {
        await page
          .getByRole('button')
          .getByText(size + ' nodes')
          .click();

        await expect(
          page.getByText('Nodes Per Shard (' + size + ')')
        ).toBeVisible();
        await expect(page.getByText('Routers (3)')).toBeVisible();
        await expect(page.getByText('2 shards')).toBeVisible();
        await expect(
          page.getByText(
            '6 nodes - CPU - 6.00 CPU; Memory - 24.00 GB; Disk - 150.00 Gi'
          )
        ).toBeVisible();
        await expect(page.getByText('3 configuration servers')).toBeVisible();
        await expect(
          page.getByText('3 routers - CPU - 3.00 CPU; Memory - 6.00 GB')
        ).toBeVisible();
        await populateResources(page, 0.6, 1, 1, size, 2, 0.6, 1, 2, 3);
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
        await waitForStatus(page, clusterName, 'Up', 660000);
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

        expect(addedCluster?.spec.sharding.enabled).toBe(true);
        expect(addedCluster?.spec.sharding.shards).toBe(2);
        expect(addedCluster?.spec.sharding.configServer.replicas).toBe(3);
      });
    });

    test(`Add data [${db} size ${size}]`, async () => {
      await prepareMongoDBTestDB(clusterName, namespace);
    });

    test(`Setup MongoDB-specific sharding [${db} size ${size}]`, async () => {
      await configureMongoDBSharding(clusterName, namespace);
    });

    test(`Validate Sharding for MongoDB [${db} size ${size}]`, async () => {
      await validateMongoDBSharding(clusterName, namespace, 't1');
      await validateMongoDBSharding(clusterName, namespace, 't2');
    });

    test(`Check PMM DB metrics [${db} size ${size}]`, async () => {
      const replicas = ['rs0', 'rs1', 'cfg'];

      for (const replica of replicas) {
        for (let i = 0; i < size; i++) {
          await checkDBMetrics(
            'node_boot_time_seconds',
            `everest-ui-${clusterName}-${replica}-${i}`,
            'admin:admin'
          );
          await checkDBMetrics(
            'mongodb_connections',
            `everest-ui-${clusterName}-${replica}-${i}`,
            'admin:admin'
          );
        }
      }

      // check mongos metrics
      for (let i = 0; i < 2; i++) {
        await checkDBMetrics(
          'node_boot_time_seconds',
          `everest-ui-${clusterName}-mongos-${i}`,
          'admin:admin'
        );
      }
    });

    test(`Check PMM QAN [${db} size ${size}]`, async () => {
      // Wait for 90 seconds for QAN to get data
      await new Promise((resolve) => setTimeout(resolve, 90000));

      const replicas = ['rs0', 'rs1'];

      for (const replica of replicas) {
        await checkQAN(
          'mongodb',
          `everest-ui-${clusterName}-${replica}-0`,
          'admin:admin'
        );
      }
    });

    test(`Create demand backup [${db} size ${size}]`, async ({ page }) => {
      await gotoDbClusterBackups(page, clusterName);
      await clickOnDemandBackup(page);
      await page.getByTestId('text-input-name').fill(baseBackupName + '-1');
      await expect(page.getByTestId('text-input-name')).not.toBeEmpty();
      await expect(
        page.getByTestId('text-input-storage-location')
      ).not.toBeEmpty();
      await page.getByTestId('form-dialog-create').click();

      await waitForStatus(page, baseBackupName + '-1', 'Succeeded', 300000);
    });

    test(`Delete data [${db} size ${size}]`, async () => {
      await dropTestDB(clusterName, namespace);
    });

    test(`Restore cluster [${db} size ${size}]`, async ({ page }) => {
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

    test(`Check data after restore [${db} size ${size}]`, async () => {
      await test.step('Validate the data in t1', async () => {
        const t1Data = await queryTestDB(clusterName, namespace, 't1');
        expect(t1Data.trim()).toBe('[{"a":1},{"a":2},{"a":3}]');
      });

      await test.step('Validate the data in t2', async () => {
        const t2Data = await queryTestDB(clusterName, namespace, 't2');
        expect(t2Data.trim()).toBe('[{"a":1},{"a":2},{"a":3}]');
      });

      await test.step('Validate sharding', async () => {
        const shardingStatus = await queryPSMDB(
          clusterName,
          namespace,
          'admin',
          'sh.status();'
        );
        expect(shardingStatus).toContain('test.t1');
        expect(shardingStatus).toContain('test.t2');
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
