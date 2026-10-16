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
import { getCITokenFromLocalStorage } from '@e2e/utils/localStorage';
import { getClusterDetailedInfo } from '@e2e/utils/storage-class';
import { EVEREST_CI_NAMESPACES } from '@e2e/constants';
import {
  giveUserAdminPermissions,
  setRBACPermissionsK8S,
} from '@e2e/utils/rbac-cmd-line';
import { deleteDbClusterFn } from '@e2e/utils/db-cluster';
import { createDbWithParameters } from '@e2e/utils/rbac';
import { gotoDbClusterBackups } from '@e2e/utils/db-clusters-list';

import {
  confirmDeletion,
  waitForDelete,
  waitForStatus,
} from '@e2e/utils/table';
import {
  fillScheduleModalForm,
  ScheduleTimeOptions,
} from '@e2e/pr/db-cluster/db-wizard/db-wizard-utils';
import {
  clickCreateSchedule,
  clickOnDemandBackup,
} from '@e2e/pr/db-cluster-details/utils';
import {
  moveForward,
  populateAdvancedConfig,
  submitWizard,
} from '@e2e/utils/db-wizard';

function getNextScheduleMinute(incrementMinutes: number): string {
  const d: number = new Date().getMinutes();
  const minute: number = (d + incrementMinutes) % 60;

  return minute.toString();
}
let token: string;

test.describe.configure({ retries: 0 });

test.describe(
  'Manage backup schedules',
  {
    tag: '@release',
  },
  () => {
    test.describe.configure({ timeout: 720000 });

    let storageClasses = [];
    const namespace2 = EVEREST_CI_NAMESPACES.EVEREST_UI;
    const namespace1 = EVEREST_CI_NAMESPACES.PXC_ONLY;
    const pxcDb = 'sch-pxc-test';
    const psmdbDb = 'sch-psmdb-test';

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

    [
      { db: 'pxc', namespace: namespace1, dbName: pxcDb },
      { db: 'psmdb', namespace: namespace2, dbName: psmdbDb },
    ].forEach(({ db, namespace, dbName }) => {
      test(`Create a ${db} db in ${namespace} namespace`, async ({ page }) => {
        expect(storageClasses.length).toBeGreaterThan(0);
        await createDbWithParameters({
          page: page,
          dbType: db,
          dbName: dbName,
          namespace: namespace,
          storageClasses: storageClasses,
          addMonitoring: db === 'pxc' ? true : false,
        });
      });
      test(`Create backup schedules for ${db} in ${namespace}`, async ({
        page,
      }) => {
        test.setTimeout(30 * 1000);

        const scheduleMinute1 = getNextScheduleMinute(2);
        const timeOption1: ScheduleTimeOptions = {
          frequency: 'hour',
          day: null,
          amPm: null,
          hour: null,
          minute: scheduleMinute1,
        };

        await test.step('Create first schedule', async () => {
          await gotoDbClusterBackups(page, dbName);
          await clickCreateSchedule(page);
          await fillScheduleModalForm(
            page,
            timeOption1,
            '0',
            'first-schedule',
            undefined
          );
          await page.getByTestId('form-dialog-create').click();
        });

        const scheduleMinute2 = getNextScheduleMinute(3);
        const timeOption2: ScheduleTimeOptions = {
          frequency: 'hour',
          day: null,
          amPm: null,
          hour: null,
          minute: scheduleMinute2,
        };

        await test.step('Create second schedule', async () => {
          await gotoDbClusterBackups(page, dbName);
          await clickCreateSchedule(page);
          await fillScheduleModalForm(
            page,
            timeOption2,
            '0',
            'second-schedule',
            undefined
          );
          await page.getByTestId('form-dialog-create').click();
        });

        await test.step('Check schedules text in page', async () => {
          expect(
            page.getByText(`Every hour at minute ${scheduleMinute1}`)
          ).toBeTruthy();
          expect(
            page.getByText(`Every hour at minute ${scheduleMinute2}`)
          ).toBeTruthy();
          expect(page.getByText('2 active schedules')).toBeTruthy();
        });
      });
      test(`Wait for two backups to succeeded for ${db}`, async ({ page }) => {
        await gotoDbClusterBackups(page, dbName);
        await expect(page.getByText(`cron-${dbName}-`)).toHaveCount(2, {
          timeout: 360000,
        });
        await expect(page.getByText('Succeeded')).toHaveCount(2, {
          timeout: 360000,
        });
      });
    });

    // step 6
    test(`User can only access and create backups of ${pxcDb} in ${namespace1} namespace (restriction at namespace level)`, async ({
      page,
    }) => {
      await setRBACPermissionsK8S([
        ['namespaces', 'read', `${namespace1}`],
        ['database-engines', 'read', '*/*'],
        ['database-clusters', '*', '*/*'],
        ['database-cluster-backups', 'create', '*/*'],
        ['database-cluster-backups', 'read', '*/*'],
        ['backup-storages', '*', '*/*'],
        ['monitoring-instances', 'read', '*/*'],
      ]);
      await test.step(`Create a scheduled and a two manual backups for ${pxcDb} in ${namespace1} namespace`, async () => {
        await expect(page.getByRole('table')).toBeVisible();
        await gotoDbClusterBackups(page, pxcDb);

        const createBackupButton = page.getByTestId('menu-button');
        await expect(createBackupButton).toBeVisible();

        createBackupButton.click();
        await expect(page.getByTestId('now-menu-item')).toBeVisible();
        await expect(page.getByTestId('schedule-menu-item')).toBeVisible();
        const scheduleMinute = getNextScheduleMinute(1);
        const timeOption: ScheduleTimeOptions = {
          frequency: 'hour',
          day: null,
          amPm: null,
          hour: null,
          minute: scheduleMinute,
        };
        await page.getByTestId('schedule-menu-item').click();
        await fillScheduleModalForm(
          page,
          timeOption,
          '0',
          'third-schedule',
          undefined
        );
        await page.getByTestId('form-dialog-create').click();

        await clickOnDemandBackup(page);
        await page.getByTestId('text-input-name').fill('backup-1');
        await expect(page.getByTestId('text-input-name')).not.toBeEmpty();
        await expect(
          page.getByTestId('text-input-storage-location')
        ).not.toBeEmpty();
        await page.getByTestId('form-dialog-create').click();

        await expect(page.getByText('backup-1')).not.toBeVisible();
        expect(page.getByText('3 active schedules')).toBeTruthy();
      });

      await test.step(`User can edit and delete backup schedules of ${pxcDb} in ${namespace1} namespace`, async () => {
        await page.getByTestId('scheduled-backups').click();
        await expect(
          page
            .getByTestId('schedule-third-schedule')
            .getByTestId('edit-schedule-button')
        ).toBeVisible();
        await expect(
          page
            .getByTestId('schedule-third-schedule')
            .getByTestId('delete-schedule-button')
        ).toBeVisible();
      });

      await test.step(`User cannot delete backups of ${pxcDb} database in ${namespace1} namespace`, async () => {
        await expect(
          page
            .locator('.MuiTableRow-root')
            .filter({ hasText: 'backup-1' })
            .getByText('Delete')
        ).not.toBeVisible();
      });

      await test.step(`User cannot view, edit or delete scheduled backups or schedules of ${psmdbDb} database in ${namespace2} namespace`, async () => {
        await page.goto('databases');
        await expect(
          page.getByRole('row').filter({ hasText: psmdbDb })
        ).not.toBeVisible();
      });
    });

    // step 11
    test(`User can only access and create backups of databases in ${namespace2} namespace (restriction at backup level)`, async ({
      page,
    }) => {
      await setRBACPermissionsK8S([
        ['namespaces', 'read', '*'],
        ['database-engines', 'read', '*/*'],
        ['database-clusters', '*', '*/*'],
        ['database-cluster-backups', 'create', `${namespace2}/*`],
        ['database-cluster-backups', 'read', `${namespace2}/*`],
        ['backup-storages', '*', '*/*'],
        ['monitoring-instances', 'read', '*/*'],
      ]);
      await test.step(`User can create manual and scheduled backups for ${psmdbDb} database in ${namespace2} namespace`, async () => {
        await gotoDbClusterBackups(page, psmdbDb);
        await expect(page.getByRole('table')).toBeVisible();

        const createBackupButton = page.getByTestId('menu-button');
        await expect(createBackupButton).toBeVisible();

        createBackupButton.click();

        await test.step(`Create buttons are visible`, async () => {
          await expect(page.getByTestId('now-menu-item')).toBeVisible();
          await expect(page.getByTestId('schedule-menu-item')).toBeVisible();
        });

        await test.step(`Create a scheduled and a manual backup`, async () => {
          const scheduleMinute = getNextScheduleMinute(1);
          const timeOption: ScheduleTimeOptions = {
            frequency: 'hour',
            day: null,
            amPm: null,
            hour: null,
            minute: scheduleMinute,
          };
          await page.getByTestId('schedule-menu-item').click();
          await fillScheduleModalForm(
            page,
            timeOption,
            '0',
            'third-schedule',
            undefined
          );
          await page.getByTestId('form-dialog-create').click();

          await clickOnDemandBackup(page);
          await page.getByTestId('text-input-name').fill('backup-1');
          await expect(page.getByTestId('text-input-name')).not.toBeEmpty();
          await expect(
            page.getByTestId('text-input-storage-location')
          ).not.toBeEmpty();
          await page.getByTestId('form-dialog-create').click();

          await expect(page.getByText('backup-1')).not.toBeVisible();
          expect(page.getByText('3 active schedules')).toBeTruthy();
        });

        await test.step('User can edit and delete backup schedules', async () => {
          await page.getByTestId('scheduled-backups').click();
          await expect(
            page
              .getByTestId('schedule-third-schedule')
              .getByTestId('edit-schedule-button')
          ).toBeVisible();
          await expect(
            page
              .getByTestId('schedule-third-schedule')
              .getByTestId('delete-schedule-button')
          ).toBeVisible();
        });

        await test.step(`User cannot view, edit or delete scheduled backups or schedules of ${pxcDb} database in ${namespace1} namespace`, async () => {
          await gotoDbClusterBackups(page, pxcDb);
          await expect(page.getByRole('table')).toBeVisible();
          expect(
            page.getByText(
              'You currently do not have any backups. Create one to get started.'
            )
          ).toBeVisible();

          expect(page.getByTestId('menu-button')).not.toBeVisible();
        });
      });
    });

    // step 15
    test(`User can only read and delete backups of databases in ${namespace2} namespace`, async ({
      page,
    }) => {
      await setRBACPermissionsK8S([
        ['namespaces', 'read', '*'],
        ['database-engines', 'read', '*/*'],
        ['database-clusters', '*', '*/*'],
        ['database-cluster-backups', 'delete', `${namespace2}/*`],
        ['database-cluster-backups', 'read', `${namespace2}/*`],
        ['backup-storages', '*', '*/*'],
      ]);
      await test.step(`User can delete manual and scheduled backups for ${psmdbDb} database in ${namespace2} namespace`, async () => {
        await gotoDbClusterBackups(page, psmdbDb);

        await page
          .locator('.MuiTableRow-root')
          .filter({ hasText: 'backup-1' })
          .getByTestId('row-actions-menu-button')
          .click();

        await expect(page.getByText('Delete')).toBeVisible();
        await page.getByText('Delete').click();
        await confirmDeletion(page, 'form-dialog-delete');
        await waitForDelete(page, 'backup-1', 30000);
      });
      await test.step(`User cannot create manual/scheduled backups or edit, delete backup schedules of ${psmdbDb} database in ${namespace2} namespace`, async () => {
        await gotoDbClusterBackups(page, psmdbDb);
        await expect(page.getByRole('table')).toBeVisible();

        await expect(page.getByText('Create backup')).not.toBeVisible();

        await page.getByTestId('scheduled-backups').click();
        await expect(
          page
            .getByTestId('schedule-third-schedule')
            .getByTestId('delete-schedule-button')
        ).not.toBeVisible();
      });
      await test.step(`User cannot view, create manual/scheduled backups or edit, delete backup schedules of ${pxcDb} database in ${namespace1} namespace`, async () => {
        await page.goto('/databases');
        await expect(page.getByText(pxcDb)).not.toBeVisible({
          timeout: 6000,
        });
      });
    });

    // step 19
    test(`User can only view backups of databases in ${namespace1} namespace`, async ({
      page,
    }) => {
      await setRBACPermissionsK8S([
        ['namespaces', 'read', '*'],
        ['database-engines', 'read', '*/*'],
        ['database-clusters', '*', '*/*'],
        ['database-cluster-backups', 'read', `${namespace1}/*`],
        ['backup-storages', '*', '*/*'],
        ['monitoring-instances', 'read', '*/*'],
      ]);
      await test.step(`User can view manual and scheduled backups for ${pxcDb} database in ${namespace1} namespace`, async () => {
        await gotoDbClusterBackups(page, pxcDb);
        await expect(page.getByRole('table')).toBeVisible();

        await expect(page.getByText('3 active schedules')).toBeVisible();
        const schedulesToggle = page.getByTestId('scheduled-backups');
        await expect(schedulesToggle).toBeVisible();
        await schedulesToggle.click();
        await expect(page.getByTestId('schedule-first-schedule')).toBeVisible();
      });
      await test.step(`User cannot create manual/scheduled backups or edit, delete backup schedules of ${pxcDb} database in ${namespace1} namespace`, async () => {
        await gotoDbClusterBackups(page, pxcDb);
        await expect(page.getByRole('table')).toBeVisible();

        await expect(page.getByText('Create backup')).not.toBeVisible();

        await page.getByTestId('scheduled-backups').click();
        const firstSchedule = page.getByTestId('schedule-first-schedule');
        await expect(
          firstSchedule.getByTestId('delete-schedule-button')
        ).not.toBeVisible();
        await expect(
          firstSchedule.getByTestId('edit-schedule-button')
        ).not.toBeVisible();
      });
      await test.step(`User cannot view, create manual/scheduled backups or edit, delete backup schedules of ${psmdbDb} database in ${namespace2} namespace`, async () => {
        await gotoDbClusterBackups(page, psmdbDb);
        await expect(page.getByRole('table')).toBeVisible();

        await expect(page.getByText('Create backup')).not.toBeVisible();

        expect(
          page.getByText(
            'You currently do not have any backups. Create one to get started.'
          )
        ).toBeVisible();
      });
    });

    [
      { db: 'pxc', namespace: namespace1, dbName: pxcDb },
      { db: 'psmdb', namespace: namespace2, dbName: psmdbDb },
    ].forEach(({ db, namespace, dbName }) => {
      test(`User can restore a backup to a new database in ${namespace}`, async ({
        page,
      }) => {
        await setRBACPermissionsK8S([
          ['namespaces', 'read', '*'],
          ['database-engines', 'read', '*/*'],
          ['database-clusters', 'read', '*/*'],
          ['database-clusters', 'create', '*/*'],
          ['backup-storages', 'read', '*/*'],
          ['database-cluster-backups', 'read', `*/*`],
          ['database-cluster-backups', 'create', `*/*`],
          ['monitoring-instances', 'read', `*/*`],
          ['database-cluster-credentials', 'read', `*/*`],
          ['database-cluster-restores', 'read', `*/*`],
          ['database-cluster-restores', 'create', `*/*`],
        ]);
        expect(storageClasses.length).toBeGreaterThan(0);
        const restoredDbName = `restore-1-${db}`;
        await gotoDbClusterBackups(page, dbName);
        await page
          .locator('.MuiTableRow-root')
          .filter({ hasText: `cron-${dbName}-` })
          .first()
          .getByTestId('MoreHorizIcon')
          .click({ timeout: 10000 });

        const restoreOption = page.getByText('Create new DB');
        await expect(restoreOption).toBeVisible();
        await restoreOption.click();
        await page.getByTestId('form-dialog-create').click();

        await page.waitForURL('**/databases/new');

        await test.step('Complete wizard for new DB', async () => {
          await test.step('Populate basic information', async () => {
            await expect(page.getByTestId('text-input-db-name')).toBeVisible();
            await page.getByTestId('text-input-db-name').fill(restoredDbName);
            await moveForward(page);
          });

          await test.step('Populate resources', async () => {
            await page.getByRole('button').getByText('1 node').click();
            await moveForward(page);
          });

          await test.step('Populate backups', async () => {
            await moveForward(page);
          });

          await test.step('Populate advanced db config', async () => {
            await populateAdvancedConfig(page, db, false, '', true, '');
            await moveForward(page);
          });

          await test.step('Submit wizard', async () => {
            await submitWizard(page);
          });
        });

        await test.step('Check for restored database', async () => {
          await page.goto('/databases');
          await waitForStatus(page, restoredDbName, 'Up', 700000);
          await expect(
            page
              .getByRole('row')
              .filter({ hasText: restoredDbName })
              .getByTestId('actions-menu-button')
          ).toBeVisible();
        });
      });
    });

    test(`User can only restore a backup to a new database in ${namespace2} (restriction at restore level)`, async ({
      page,
    }) => {
      const dbName = psmdbDb;
      const db = 'psmdb';
      await setRBACPermissionsK8S([
        ['namespaces', 'read', '*'],
        ['database-engines', 'read', '*/*'],
        ['database-clusters', 'read', '*/*'],
        ['database-clusters', 'create', '*/*'],
        ['backup-storages', 'read', '*/*'],
        ['database-cluster-backups', 'read', `*/*`],
        ['database-cluster-backups', 'create', `*/*`],
        ['monitoring-instances', 'read', `*/*`],
        ['database-cluster-credentials', 'read', `*/*`],
        ['database-cluster-restores', 'read', `${namespace2}/*`],
        ['database-cluster-restores', 'create', `${namespace2}/*`],
      ]);
      await test.step(`User can restore to a new ${psmdbDb} in ${namespace2} namespace`, async () => {
        expect(storageClasses.length).toBeGreaterThan(0);
        const restoredDbName = `restore-2-psmdb`;
        await gotoDbClusterBackups(page, dbName);
        await page
          .locator('.MuiTableRow-root')
          .filter({ hasText: `cron-${dbName}-` })
          .first()
          .getByTestId('MoreHorizIcon')
          .click({ timeout: 10000 });

        const restoreOption = page.getByText('Create new DB');
        await expect(restoreOption).toBeVisible();
        await restoreOption.click();
        await page.getByTestId('form-dialog-create').click();

        await test.step('Complete wizard for new DB', async () => {
          await test.step('Populate basic information', async () => {
            await expect(page.getByTestId('text-input-db-name')).toBeVisible();
            await page.getByTestId('text-input-db-name').fill(restoredDbName);
            await moveForward(page);
          });

          await test.step('Populate resources', async () => {
            await page.getByRole('button').getByText('1 node').click();
            await moveForward(page);
          });

          await test.step('Populate backups', async () => {
            await moveForward(page);
          });

          await test.step('Populate advanced db config', async () => {
            await populateAdvancedConfig(page, db, false, '', true, '');
            await moveForward(page);
          });

          await test.step('Submit wizard', async () => {
            await submitWizard(page);
          });
        });

        await test.step('Check for restored database', async () => {
          await page.goto('/databases');
          await expect(page.getByText(restoredDbName)).toBeVisible();
          await expect(
            page
              .getByRole('row')
              .filter({ hasText: restoredDbName })
              .getByTestId('actions-menu-button')
          ).toBeVisible();
        });
      });
      await test.step(`User cannot restore to a new ${pxcDb} in ${namespace1} namespace`, async () => {
        await gotoDbClusterBackups(page, pxcDb);
        await expect(page.getByRole('table')).toBeVisible();
        await expect(
          page
            .locator('.MuiTableRow-root')
            .filter({ hasText: 'backup-1' })
            .getByTestId('MoreHorizIcon')
        ).not.toBeVisible();
      });
    });
    test('Delete databases', async ({ page, request }) => {
      await giveUserAdminPermissions();
      await deleteDbClusterFn(request, pxcDb, namespace1);
      await deleteDbClusterFn(request, psmdbDb, namespace2);
      await deleteDbClusterFn(request, 'restore-1-pxc', namespace1);
      await deleteDbClusterFn(request, 'restore-1-psmdb', namespace2);
      await deleteDbClusterFn(request, 'restore-2-psmdb', namespace2);

      await expect(page.getByText(pxcDb)).not.toBeVisible({ timeout: 70000 });
      await expect(page.getByText(psmdbDb)).not.toBeVisible({ timeout: 70000 });
      await expect(page.getByText('restore-1-pxc')).not.toBeVisible({
        timeout: 70000,
      });
      await expect(page.getByText('restore-1-psmdbDb')).not.toBeVisible({
        timeout: 70000,
      });
      await expect(page.getByText('restore-2-psmdbDb')).not.toBeVisible({
        timeout: 70000,
      });
    });
  }
);
