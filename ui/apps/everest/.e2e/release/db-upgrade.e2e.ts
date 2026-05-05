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
import { EVEREST_CI_NAMESPACES, technologyMap } from '@e2e/constants';
import {
  waitForStatus,
  waitForDelete,
  findRowAndClickActions,
} from '@e2e/utils/table';
import { clickOnDemandBackup } from '@e2e/pr/db-cluster-details/utils';
import {
  prepareTestDB,
  insertTestDB,
  prepareMongoDBTestDB,
  validateMongoDBSharding,
  configureMongoDBSharding,
} from '@e2e/utils/db-cmd-line';
import { getDbClusterAPI } from '@e2e/utils/db-cluster';
import { findDbAndClickRow } from '@e2e/utils/db-clusters-list';
import { shouldExecuteDBCombination } from '@e2e/utils/generic';
import { getVersionServiceDBVersions } from '@e2e/utils/version-service';
import { Operator } from '@e2e/upgrade/types';
import { getDbOperatorVersionK8s } from '@e2e/utils/generic';

let token: string;

const majorDBVersions = {
  psmdb: ['6.0', '7.0', '8.0'],
};

test.describe.configure({ retries: 0 });

[
  { db: 'psmdb', size: 3, sharding: false },
  { db: 'psmdb', size: 3, sharding: true },
].forEach(({ db, size, sharding }) => {
  test.describe(
    'Database upgrade',
    {
      tag: '@release',
    },
    () => {
      test.skip(!shouldExecuteDBCombination(db, size));
      test.describe.configure({ timeout: 720000 });

      const shSuffix = sharding ? '-sh' : '';
      const clusterName = `${db}-${size}${shSuffix}-upg`;

      let storageClasses = [];
      const namespace = EVEREST_CI_NAMESPACES.EVEREST_UI;
      const monitoringName = 'e2e-endpoint-0';
      const baseBackupName = `upg-${db}-${size}${shSuffix}`;

      let operatorLongName: string | undefined;
      let crVersion: string | undefined;

      test.beforeAll(async ({ request }) => {
        token = await getTokenFromLocalStorage();

        const { storageClassNames = [] } = await getClusterDetailedInfo(
          token,
          request
        );
        storageClasses = storageClassNames;

        operatorLongName =
          { psmdb: Operator.PSMDB, pxc: Operator.PXC, postgresql: Operator.PG }[
            db
          ] || undefined;
        crVersion = await getDbOperatorVersionK8s(namespace, operatorLongName);
      });

      test(`Cluster creation [${db} size ${size} sharding ${sharding}]`, async ({
        page,
        request,
      }) => {
        expect(storageClasses.length).toBeGreaterThan(0);

        await page.goto('/databases');
        await page.getByTestId('add-db-cluster-button').waitFor();
        await page.getByTestId('add-db-cluster-button').click();
        await page.getByTestId(`add-db-cluster-button-${db}`).click();

        await test.step('Populate basic information', async () => {
          const dbVersion = (
            await getVersionServiceDBVersions(
              db,
              crVersion,
              request,
              majorDBVersions[db][0]
            )
          )[0];

          if (process.env.DEBUG_TESTS) {
            console.log('Starting with version: ' + dbVersion);
          }

          await populateBasicInformation(
            page,
            namespace,
            clusterName,
            db,
            storageClasses[0],
            false,
            dbVersion
          );

          if (sharding) {
            const shardingCheckbox = page.getByTestId('switch-input-sharding');
            await shardingCheckbox.click();
            await page.getByTestId('db-wizard-continue-button').isEnabled();
            await expect(
              page.getByTestId('db-wizard-continue-button')
            ).not.toBeDisabled();
          }

          await moveForward(page);
        });

        await test.step('Populate resources', async () => {
          await page
            .getByRole('button')
            .getByText(size + ' node')
            .click();
          if (db === 'psmdb' && sharding) {
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
            await expect(
              page.getByText('3 configuration servers')
            ).toBeVisible();
            await expect(
              page.getByText('3 routers - CPU - 3.00 CPU; Memory - 6.00 GB')
            ).toBeVisible();
            await populateResources(page, 0.6, 1, 1, size, 2, 0.6, 1, 2, 3);
          } else {
            await expect(page.getByText('Nodes (' + size + ')')).toBeVisible();
            await populateResources(page, 0.6, 1, 1, size);
          }
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
        });
      });

      test(`Add data [${db} size ${size} sharding ${sharding}]`, async () => {
        if (db === 'psmdb' && sharding) {
          await prepareMongoDBTestDB(clusterName, namespace);
        } else {
          await prepareTestDB(clusterName, namespace);
        }
      });

      test(`Setup MongoDB-specific sharding [${db} size ${size} sharding ${sharding}]`, async () => {
        test.skip(db !== 'psmdb' || !sharding);
        await configureMongoDBSharding(clusterName, namespace);
        await validateMongoDBSharding(clusterName, namespace, 't1');
        await validateMongoDBSharding(clusterName, namespace, 't2');
      });

      test(`Create demand backup [${db} size ${size} sharding ${sharding}]`, async ({
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

        await waitForStatus(page, baseBackupName + '-1', 'Succeeded', 300000);
      });

      test(`Run major/minor upgrades [${db} size ${size} sharding ${sharding}]`, async ({
        page,
        request,
      }) => {
        test.setTimeout(2400000);
        let i = 0;
        let expectedResult: string[] = ['1', '2', '3'];

        for (const dbVersion of majorDBVersions[db]) {
          if (i !== 0) {
            // we skip first major version because it's just used for initial installation
            expectedResult.push((i + 3).toString());

            await test.step(`Upgrade to ${dbVersion}`, async () => {
              // by default we upgrade to non latest version unless only 1 exists (so that later we can do minor upgrade)
              const dbVersions = await getVersionServiceDBVersions(
                db,
                crVersion,
                request,
                majorDBVersions[db][i]
              );
              let versionElement = 1;
              if (dbVersions.length <= 1) {
                versionElement = 0;
              }
              const nextMajorVersion = (
                await getVersionServiceDBVersions(
                  db,
                  crVersion,
                  request,
                  majorDBVersions[db][i]
                )
              )[versionElement];

              if (process.env.DEBUG_TESTS) {
                console.log('Upgrading to: ' + nextMajorVersion);
              }

              await page.goto('/databases');
              await findDbAndClickRow(page, clusterName);
              await page.getByTestId('upgrade-db-btn').click();
              await expect(page.getByText('Upgrade DB version')).toBeVisible();
              await expect(
                page.getByTestId('form-dialog-upgrade')
              ).toBeDisabled();
              await page.getByRole('combobox').click();
              await page
                .getByRole('option', { name: `${nextMajorVersion}` })
                .click();
              await expect(
                page.getByTestId('form-dialog-cancel')
              ).toBeEnabled();
              await page.getByTestId('form-dialog-upgrade').click();
              await page.goto('/databases');
              await waitForStatus(page, clusterName, 'Upgrading', 15000);
              await waitForStatus(page, clusterName, 'Up', 660000);
              const technology = technologyMap[db] || 'Unknown';
              await expect(
                page.getByText(`${technology} ${nextMajorVersion}`)
              ).toBeVisible();
            });

            await test.step('Insert more data after upgrade', async () => {
              await insertTestDB(
                clusterName,
                namespace,
                [(i + 3).toString()],
                expectedResult
              );
            });

            await test.step(`Create demand backup after upgrade`, async () => {
              await gotoDbClusterBackups(page, clusterName);
              await clickOnDemandBackup(page);
              await page
                .getByTestId('text-input-name')
                .fill(baseBackupName + '-' + (i + 1).toString());
              await expect(page.getByTestId('text-input-name')).not.toBeEmpty();
              await expect(
                page.getByTestId('text-input-storage-location')
              ).not.toBeEmpty();
              await page.getByTestId('form-dialog-create').click();

              await waitForStatus(
                page,
                baseBackupName + '-' + (i + 1).toString(),
                'Succeeded',
                300000
              );
            });

            await test.step(`Upgrade to latest minor version if exists`, async () => {
              const dbMinorVersions = await getVersionServiceDBVersions(
                db,
                crVersion,
                request,
                majorDBVersions[db][i]
              );

              if (dbMinorVersions.length <= 1) {
                console.log(
                  `Skipping upgrade: Only one minor version exists for ${db} and ${majorDBVersions[db][i]}`
                );
                return;
              }

              const nextDbMinorVersion = dbMinorVersions[0];

              if (process.env.DEBUG_TESTS) {
                console.log('Upgrading to: ' + nextDbMinorVersion);
              }

              await page.goto('/databases');
              await findDbAndClickRow(page, clusterName);
              await page.getByTestId('upgrade-db-btn').click();
              await expect(page.getByText('Upgrade DB version')).toBeVisible();
              await expect(
                page.getByTestId('form-dialog-upgrade')
              ).toBeDisabled();
              await page.getByRole('combobox').click();
              await page
                .getByRole('option', { name: `${nextDbMinorVersion}` })
                .click();
              await expect(
                page.getByTestId('form-dialog-cancel')
              ).toBeEnabled();
              await page.getByTestId('form-dialog-upgrade').click();
              await page.goto('/databases');
              await waitForStatus(page, clusterName, 'Upgrading', 15000);
              await waitForStatus(page, clusterName, 'Up', 660000);
              const technology = technologyMap[db] || 'Unknown';
              await expect(
                page.getByText(`${technology} ${nextDbMinorVersion}`)
              ).toBeVisible();
            });
          }
          i++;
        }

        // after all major/minor upgrades done Edit/Upgrade button should not be visible
        await page.goto('/databases');
        await findDbAndClickRow(page, clusterName);
        await expect(page.getByTestId('upgrade-db-btn')).not.toBeVisible();
      });

      test(`Delete all backups [${db} size ${size} sharding ${sharding}]`, async ({
        page,
      }) => {
        for (let i = 1; i <= majorDBVersions[db].length; i++) {
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
            await waitForDelete(page, baseBackupName + `-${i}`, 30000);
          }
        }
      });

      test(`Delete cluster [${db} size ${size} sharding ${sharding}]`, async ({
        page,
      }) => {
        await deleteDbCluster(page, clusterName);
        await waitForStatus(page, clusterName, 'Deleting', 15000);
        await waitForDelete(page, clusterName, 240000);
      });
    }
  );
});
