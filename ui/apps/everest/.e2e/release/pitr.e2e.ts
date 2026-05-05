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
  findDbAndClickActions,
} from '@e2e/utils/db-clusters-list';
import { getTokenFromLocalStorage } from '@e2e/utils/localStorage';
import { getClusterDetailedInfo } from '@e2e/utils/storage-class';
import {
  moveForward,
  submitWizard,
  populateBasicInformation,
  populateResources,
  populateAdvancedConfig,
  goToLastStepByStepAndSubmit,
} from '@e2e/utils/db-wizard';
import { EVEREST_CI_NAMESPACES, getBucketNamespacesMap } from '@e2e/constants';
import {
  waitForStatus,
  waitForDelete,
  findRowAndClickActions,
} from '@e2e/utils/table';
import { clickOnDemandBackup } from '@e2e/pr/db-cluster-details/utils';
import {
  prepareTestDB,
  dropTestDB,
  queryTestDB,
  insertTestDB,
} from '@e2e/utils/db-cmd-line';
import { addFirstScheduleInDBWizard } from '@e2e/pr/db-cluster/db-wizard/db-wizard-utils';
import { getDbClusterAPI, updateDbClusterAPI } from '@e2e/utils/db-cluster';
import { shouldExecuteDBCombination } from '@e2e/utils/generic';

type pitrTime = {
  day: string;
  month: string;
  year: string;
  hour: string;
  minute: string;
  second: string;
};

let token: string;
let backupStorage: string;
let pitrRestoreTime: pitrTime = {
  day: '',
  month: '',
  year: '',
  hour: '',
  minute: '',
  second: '',
};

function getCurrentPITRTime(): pitrTime {
  const time = {} as pitrTime;
  const now: Date = new Date();

  // Get date parts
  time.day = now.getDate().toString();
  time.month = (now.getMonth() + 1).toString(); // Months are zero-indexed, so add 1
  time.year = now.getFullYear().toString();

  // Get time parts
  time.hour = now.getHours().toString();
  time.minute = now.getMinutes().toString();
  time.second = now.getSeconds().toString();

  return time;
}

function getFormattedPITRTime(time: pitrTime): string {
  const formattedDateTime: string = `${time.day.padStart(2, '0')}/${time.month.padStart(2, '0')}/${time.year} at ${time.hour.padStart(2, '0')}:${time.minute.padStart(2, '0')}:${time.second.padStart(2, '0')}`;

  return formattedDateTime;
}

test.describe.configure({ retries: 0 });

const zephyrMap: Record<string, string> = {
  'backup-pxc': 'T101',
  'backup-psmdb': 'T102',
  'backup-postgresql': 'T103',
  'restore-pxc': 'T104',
  'restore-psmdb': 'T105',
  'restore-postgresql': 'T106',
  'restore-newdb-pxc': 'T125',
  'restore-newdb-psmdb': 'T126',
  'restore-newdb-postgresql': 'T127',
};

function getBackupStorage(): string {
  const bucketNamespacesMap = getBucketNamespacesMap();

  // Check if map includes ["everest-testing","everest-ui"]
  const hasEverestTesting = bucketNamespacesMap.some(
    ([bucket, ns]) => bucket === 'everest-testing' && ns === 'everest-ui'
  );

  if (
    process.env.EVEREST_S3_ACCESS_KEY &&
    process.env.EVEREST_S3_SECRET_KEY &&
    hasEverestTesting
  ) {
    backupStorage = 'everest-testing';
    console.log(
      `AWS credentials present and EVEREST_BUCKETS_NAMESPACES_MAP includes ["everest-testing","everest-ui"], so using bucket ${backupStorage}`
    );
  } else {
    backupStorage = 'bucket-1';
    console.log(
      `AWS credentials missing or EVEREST_BUCKETS_NAMESPACES_MAP does not include ["everest-testing","everest-ui"], so using MinIO bucket ${backupStorage}`
    );
  }

  return backupStorage;
}

[
  { db: 'psmdb', size: 3 },
  { db: 'pxc', size: 3 },
  { db: 'postgresql', size: 3 },
].forEach(({ db, size }) => {
  test.describe(
    'PITR',
    {
      tag: '@release',
    },
    () => {
      test.skip(!shouldExecuteDBCombination(db, size));
      test.describe.configure({ timeout: 1_200_000 }); // 20 minutes

      const clusterName = `${db}-${size}-pitr`;
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

        backupStorage = await getBackupStorage();
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

        await test.step('Populate backups, enable PITR', async () => {
          await addFirstScheduleInDBWizard(page, backupStorage);
          const pitrCheckbox = page
            .getByTestId('switch-input-pitr-enabled')
            .getByRole('checkbox');

          if (db !== 'postgresql') {
            await expect(pitrCheckbox).not.toBeChecked();
            await pitrCheckbox.setChecked(true);

            if (db === 'pxc') {
              const pitrStorageLocation = page.getByTestId(
                'text-input-pitr-storage-location'
              );
              await expect(pitrStorageLocation).toBeVisible();
              await expect(pitrStorageLocation).not.toBeEmpty();
              await pitrStorageLocation.click();
              await page.getByRole('option', { name: backupStorage }).click();
            } else {
              await expect(
                page.getByText(`Storage: ${backupStorage}`)
              ).toHaveCount(2);
            }
          }

          await expect(pitrCheckbox).toBeChecked();
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

        await test.step('Check db list and status', async () => {
          await page.goto('/databases');
          if (db !== 'postgresql') {
            await waitForStatus(page, clusterName, 'Initializing', 30000);
          }
          await waitForStatus(page, clusterName, 'Up', 900000);
        });

        await test.step('Update PSMDB cluster PITR uploadIntervalSec', async () => {
          if (db !== 'psmdb') {
            return;
          }

          const psmdbCluster = await getDbClusterAPI(
            clusterName,
            EVEREST_CI_NAMESPACES.EVEREST_UI,
            request,
            token
          );
          psmdbCluster.spec.backup.pitr.uploadIntervalSec = 60;
          await updateDbClusterAPI(
            clusterName,
            EVEREST_CI_NAMESPACES.EVEREST_UI,
            psmdbCluster,
            request,
            token
          );
        });

        await test.step('Check db cluster k8s object options', async () => {
          const addedCluster = await getDbClusterAPI(
            clusterName,
            EVEREST_CI_NAMESPACES.EVEREST_UI,
            request,
            token
          );

          expect(addedCluster?.spec.backup.pitr.enabled).toBe(true);
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

      test(`Create demand backup [${db} size ${size}]`, async ({ page }) => {
        await gotoDbClusterBackups(page, clusterName);
        await clickOnDemandBackup(page);
        await page.getByTestId('text-input-name').fill(baseBackupName + '-1');
        await expect(page.getByTestId('text-input-name')).not.toBeEmpty();
        if (db !== 'psmdb') {
          // PSMDB uses the same storage location for PITR and backups
          await page.getByTestId('text-input-storage-location').click();
          await page.getByRole('option', { name: `${backupStorage}` }).click();
        }
        await expect(
          page.getByTestId('text-input-storage-location')
        ).toHaveValue(backupStorage);
        await page.getByTestId('form-dialog-create').click();

        await waitForStatus(page, baseBackupName + '-1', 'Succeeded', 360000);
      });

      test(`Add more data [${db} size ${size}]`, async () => {
        await insertTestDB(
          clusterName,
          namespace,
          ['4', '5', '6'],
          ['1', '2', '3', '4', '5', '6']
        );
        pitrRestoreTime = getCurrentPITRTime();
        await insertTestDB(
          clusterName,
          namespace,
          ['7', '8', '9'],
          ['1', '2', '3', '4', '5', '6', '7', '8', '9']
        );
      });

      test(`Wait 1 min for binlogs to be uploaded [${db} size ${size}]`, async () => {
        const delay = (ms: number) =>
          new Promise((resolve) => setTimeout(resolve, ms));
        await delay(65000);
      });

      test(`Delete data [${db} size ${size}]`, async () => {
        await dropTestDB(clusterName, namespace);
      });

      test(`Restore cluster [${db} size ${size}]`, async ({ page }) => {
        await page.goto('databases');
        await findDbAndClickActions(page, clusterName, 'Restore from a backup');
        await page
          .getByTestId('radio-option-fromPITR')
          .click({ timeout: 5000 });
        await expect(page.getByTestId('radio-option-fromPITR')).toBeChecked({
          timeout: 5000,
        });
        await expect(
          page.getByPlaceholder('DD/MM/YYYY at hh:mm:ss')
        ).toBeVisible({ timeout: 5000 });
        await expect(
          page.getByPlaceholder('DD/MM/YYYY at hh:mm:ss')
        ).not.toBeEmpty({ timeout: 5000 });
        await page.getByTestId('CalendarIcon').click({ timeout: 5000 });
        await page
          .getByLabel(pitrRestoreTime.hour + ' hours', { exact: true })
          .click({ timeout: 5000 });
        await page
          .getByLabel(pitrRestoreTime.minute + ' minutes', { exact: true })
          .click({ timeout: 5000 });
        await page
          .getByLabel(pitrRestoreTime.second + ' seconds', { exact: true })
          .click({ timeout: 5000 });
        await expect(
          page.getByPlaceholder('DD/MM/YYYY at hh:mm:ss')
        ).toHaveValue(getFormattedPITRTime(pitrRestoreTime));

        await page.getByTestId('form-dialog-restore').click({ timeout: 5000 });

        await page.goto('/databases');
        await waitForStatus(page, clusterName, 'Restoring', 30000);
        await waitForStatus(page, clusterName, 'Up', 900000);

        await gotoDbClusterRestores(page, clusterName);
        await waitForStatus(page, 'restore-', 'Succeeded', 120000);
      });

      test(`Check data after restore [${db} size ${size}]`, async () => {
        const result = await queryTestDB(clusterName, namespace);
        switch (db) {
          case 'pxc':
            expect(result.trim()).toBe('1\n2\n3\n4\n5\n6');
            break;
          case 'psmdb':
            expect(result.trim()).toBe(
              '[{"a":1},{"a":2},{"a":3},{"a":4},{"a":5},{"a":6}]'
            );
            break;
          case 'postgresql':
            expect(result.trim()).toBe('1\n 2\n 3\n 4\n 5\n 6');
            break;
        }
      });

      test(`Delete first restore [${db} size ${size}]`, async ({ page }) => {
        // we delete first restore since they have random names and we don't
        // want it to mess later with the test
        await gotoDbClusterRestores(page, clusterName);
        await findRowAndClickActions(page, baseBackupName + `-1`, 'Delete');
        await expect(page.getByLabel('Delete restore')).toBeVisible();
        await page.getByTestId('confirm-dialog-delete').click();
        await waitForDelete(page, baseBackupName + `-1`, 15000);
      });

      test(`Create second demand backup [${db} size ${size}]`, async ({
        page,
      }) => {
        await gotoDbClusterBackups(page, clusterName);
        await clickOnDemandBackup(page);
        await page.getByTestId('text-input-name').fill(baseBackupName + '-2');
        await expect(page.getByTestId('text-input-name')).not.toBeEmpty();
        if (db !== 'psmdb') {
          // PSMDB uses the same storage location for PITR and backups
          await page.getByTestId('text-input-storage-location').click();
          await page.getByRole('option', { name: `${backupStorage}` }).click();
        }
        await expect(
          page.getByTestId('text-input-storage-location')
        ).toHaveValue(backupStorage);
        await page.getByTestId('form-dialog-create').click();

        await waitForStatus(page, baseBackupName + '-2', 'Succeeded', 360000);
      });

      test(`Add more data for second PITR restore [${db} size ${size}]`, async () => {
        await insertTestDB(
          clusterName,
          namespace,
          ['7', '8'],
          ['1', '2', '3', '4', '5', '6', '7', '8']
        );
        // for PG we need one more transaction to be able to restore to the previous one
        if (db == 'postgresql') {
          await insertTestDB(
            clusterName,
            namespace,
            ['9'],
            ['1', '2', '3', '4', '5', '6', '7', '8', '9']
          );
        }
      });

      test(`Wait 1 min for binlogs to be uploaded for second restore [${db} size ${size}]`, async () => {
        const delay = (ms: number) =>
          new Promise((resolve) => setTimeout(resolve, ms));
        await delay(65000);
      });

      test(`Delete data before second restore [${db} size ${size}]`, async () => {
        await dropTestDB(clusterName, namespace);
      });

      test(`Restore latest PITR cluster [${db} size ${size}]`, async ({
        page,
      }) => {
        await page.goto('databases');
        await findDbAndClickActions(page, clusterName, 'Restore from a backup');
        await page
          .getByTestId('radio-option-fromPITR')
          .click({ timeout: 10000 });
        await expect(page.getByTestId('radio-option-fromPITR')).toBeChecked({
          timeout: 10000,
        });
        await expect(
          page.getByPlaceholder('DD/MM/YYYY at hh:mm:ss')
        ).toBeVisible({ timeout: 10000 });
        await expect(
          page.getByPlaceholder('DD/MM/YYYY at hh:mm:ss')
        ).not.toBeEmpty({ timeout: 10000 });

        await page.getByTestId('CalendarIcon').click();
        await page.getByRole('button', { name: 'OK' }).click();

        await page.getByTestId('form-dialog-restore').click({ timeout: 10000 });

        await page.goto('/databases');
        await waitForStatus(page, clusterName, 'Restoring', 30000);
        await waitForStatus(page, clusterName, 'Up', 900000);

        await gotoDbClusterRestores(page, clusterName);
        await waitForStatus(page, 'restore-', 'Succeeded', 120000);
      });

      test(`Check data after second restore [${db} size ${size}]`, async () => {
        const result = await queryTestDB(clusterName, namespace);
        switch (db) {
          case 'pxc':
            expect(result.trim()).toBe('1\n2\n3\n4\n5\n6\n7\n8');
            break;
          case 'psmdb':
            expect(result.trim()).toBe(
              '[{"a":1},{"a":2},{"a":3},{"a":4},{"a":5},{"a":6},{"a":7},{"a":8}]'
            );
            break;
          case 'postgresql':
            expect(result.trim()).toBe('1\n 2\n 3\n 4\n 5\n 6\n 7\n 8');
            break;
        }
      });

      zephyrId = zephyrMap[`restore-newdb-${db}`];
      test(`${zephyrId} - PITR restore to new cluster [${db} size ${size}]`, async ({
        page,
      }) => {
        await test.step('Create a backup to restore from', async () => {
          await gotoDbClusterBackups(page, clusterName);
          await clickOnDemandBackup(page);
          await page
            .getByTestId('text-input-name')
            .fill(baseBackupName + '-new-cluster');
          await expect(page.getByTestId('text-input-name')).not.toBeEmpty();
          if (db !== 'psmdb') {
            // PSMDB uses the same storage location for PITR and backups
            await page.getByTestId('text-input-storage-location').click();
            await page
              .getByRole('option', { name: `${backupStorage}` })
              .click();
          }
          await expect(
            page.getByTestId('text-input-storage-location')
          ).toHaveValue(backupStorage);

          await page.getByTestId('form-dialog-create').click();
          await waitForStatus(
            page,
            baseBackupName + '-new-cluster',
            'Succeeded',
            360000
          );
        });

        await test.step('Add some data after the backup', async () => {
          await insertTestDB(
            clusterName,
            namespace,
            ['9'],
            ['1', '2', '3', '4', '5', '6', '7', '8', '9']
          );
          // for PG we need one more transaction to be able to restore to the previous one
          if (db == 'postgresql') {
            await insertTestDB(
              clusterName,
              namespace,
              ['10'],
              ['1', '2', '3', '4', '5', '6', '7', '8', '9', '10']
            );
          }
        });

        await test.step('Wait for binlogs to be uploaded', async () => {
          await page.waitForTimeout(60000);
        });

        let newClusterName: string;
        await test.step('Create DB from latest PITR', async () => {
          // Go to restore wizard
          await page.goto('databases');
          await findDbAndClickActions(
            page,
            clusterName,
            'Create DB from a backup'
          );
          await page
            .getByTestId('radio-option-fromPITR')
            .click({ timeout: 5000 });
          await expect(page.getByTestId('radio-option-fromPITR')).toBeChecked({
            timeout: 5000,
          });

          await expect(
            page.getByPlaceholder('DD/MM/YYYY at hh:mm:ss')
          ).toBeVisible({ timeout: 5000 });
          await expect(
            page.getByPlaceholder('DD/MM/YYYY at hh:mm:ss')
          ).not.toBeEmpty({ timeout: 5000 });

          // Submit and open DB creation wizard
          await page.getByTestId('CalendarIcon').click();
          await page.getByRole('button', { name: 'OK' }).click();

          await page.getByTestId('form-dialog-create').click({ timeout: 5000 });
          let currentUrl = page.url();
          expect(currentUrl).toContain('/databases/new');

          newClusterName = await page
            .getByTestId('text-input-db-name')
            .inputValue();
          expect(newClusterName).not.toBe('');
          await goToLastStepByStepAndSubmit(page, 5000);
        });

        await test.step('Wait for the new cluster to be created and ready', async () => {
          await page.goto('/databases');
          if (db !== 'postgresql') {
            await waitForStatus(page, newClusterName, 'Initializing', 30000);
          }
          await waitForStatus(page, newClusterName, 'Restoring', 660000);
          await waitForStatus(page, newClusterName, 'Up', 900000);
        });

        await test.step('Verify the restore was successful', async () => {
          // PG does not create a DBR object when restoring to new cluster.
          if (db !== 'postgresql') {
            await gotoDbClusterRestores(page, newClusterName);
            await waitForStatus(page, newClusterName, 'Succeeded', 120000);
          }
        });

        await test.step('Verify the data in the new cluster', async () => {
          const result = await queryTestDB(newClusterName, namespace);
          switch (db) {
            case 'pxc':
              expect(result.trim()).toBe('1\n2\n3\n4\n5\n6\n7\n8\n9');
              break;
            case 'psmdb':
              expect(result.trim()).toBe(
                '[{"a":1},{"a":2},{"a":3},{"a":4},{"a":5},{"a":6},{"a":7},{"a":8},{"a":9}]'
              );
              break;
            case 'postgresql':
              expect(result.trim()).toBe('1\n 2\n 3\n 4\n 5\n 6\n 7\n 8\n 9');
              break;
          }
        });

        await test.step('Delete the restored cluster', async () => {
          await deleteDbCluster(page, newClusterName);
          await waitForStatus(page, newClusterName, 'Deleting', 15000);
          await waitForDelete(page, newClusterName, 240000);
        });
      });

      test(`Delete second restore [${db} size ${size}]`, async ({ page }) => {
        await gotoDbClusterRestores(page, clusterName);
        await findRowAndClickActions(page, baseBackupName + `-2`, 'Delete');
        await expect(page.getByLabel('Delete restore')).toBeVisible();
        await page.getByTestId('confirm-dialog-delete').click();
        await waitForDelete(page, baseBackupName + `-2`, 15000);
      });

      test(`Delete all backups [${db} size ${size}]`, async ({ page }) => {
        for (let i = 1; i <= 2; i++) {
          // PG doesn't list first backup after second restore
          if (db !== 'postgresql' && i !== 1) {
            await gotoDbClusterBackups(page, clusterName);
            await findRowAndClickActions(
              page,
              baseBackupName + `-${i}`,
              'Delete'
            );
            await expect(page.getByLabel('Delete backup')).toBeVisible();
            await page.getByTestId('form-dialog-delete').click();
            await waitForDelete(page, baseBackupName + `-${i}`, 60000);
          }
        }
      });

      test(`Delete cluster [${db} size ${size}]`, async ({ page }) => {
        await deleteDbCluster(page, clusterName);
        await waitForStatus(page, clusterName, 'Deleting', 15000);
        await waitForDelete(page, clusterName, 240000);
      });
    }
  );
});
