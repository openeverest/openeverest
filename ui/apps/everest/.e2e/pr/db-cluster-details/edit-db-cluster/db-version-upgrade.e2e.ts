import { expect, test } from '@playwright/test';
import { getCITokenFromLocalStorage } from '@e2e/utils/localStorage';
import {
  deleteDbCluster,
  findDbAndClickRow,
} from '@e2e/utils/db-clusters-list';
import { getClusterDetailedInfo } from '@e2e/utils/storage-class';
import { moveForward, submitWizard } from '@e2e/utils/db-wizard';
import { waitForStatus, waitForDelete } from '@e2e/utils/table';
import { selectDbEngine } from '@e2e/pr/db-cluster/db-wizard/db-wizard-utils';
import { goToUrl, limitedSuffixedName } from '@e2e/utils/generic';
import { advancedConfigurationSelectFirstStorageClass } from '@e2e/pr/db-cluster/db-wizard/create-db-cluster/steps/advanced-configuration-step';
import { basicInformationSelectNamespaceCheck } from '@e2e/pr/db-cluster/db-wizard/create-db-cluster/steps/basic-information-step';
import { EVEREST_CI_NAMESPACES, TIMEOUTS } from '@e2e/constants';
import { deleteDbClusterFn, getDbClusterAPI } from '@e2e/utils/db-cluster';

const namespace = EVEREST_CI_NAMESPACES.EVEREST_UI,
  testPrefix = 'pr-db-upgd';

let token: string;

[
  { db: 'psmdb', size: 1 },
  { db: 'pxc', size: 1 },
  { db: 'postgresql', size: 1 },
].forEach(
  ({ db, size }: { db: 'pxc' | 'psmdb' | 'postgresql'; size: number }) => {
    test.describe.parallel('Upgrade DB cluster version', () => {
      test.describe.configure({ timeout: 900000 });

      let storageClasses = [];

      test.beforeAll(async ({ request }) => {
        token = await getCITokenFromLocalStorage();

        const { storageClassNames = [] } = await getClusterDetailedInfo(
          token,
          request
        );
        storageClasses = storageClassNames;
        expect(storageClasses.length).toBeGreaterThan(0);
      });

      test(`Upgrade ${db} version`, async ({ page, request }) => {
        const clusterName = limitedSuffixedName(testPrefix);

        await goToUrl(page, '/databases');

        await test.step('Start DB cluster creation wizard', async () => {
          await selectDbEngine(page, db);
        });

        await test.step('Populate basic info', async () => {
          // namespace
          await basicInformationSelectNamespaceCheck(page, namespace);

          await page.getByTestId('text-input-db-name').fill(clusterName);

          await page.getByTestId('select-input-db-version').waitFor();
          await page.getByTestId('select-db-version-button').click();
          const dbVersionOptions = page.getByRole('option');
          await dbVersionOptions.nth(1).click();
        });

        await test.step('Populate resources', async () => {
          // go to resources page
          await moveForward(page);
          const nodesAccordion = page.getByTestId('nodes-accordion');
          await nodesAccordion.waitFor({ timeout: TIMEOUTS.ThirtySeconds });
          // Select "Number of Nodes = 1"
          await nodesAccordion
            .getByTestId(`toggle-button-nodes-${size}`)
            .click();

          // await page
          //   .getByRole('button')
          //   .getByText(size + ' node')
          //   .click();
          await nodesAccordion.getByTestId('text-input-cpu').fill('1');
          await nodesAccordion.getByTestId('text-input-memory').fill('1');
          await nodesAccordion.getByTestId('text-input-disk').fill('1');
          await expect(page.getByText(`1 node - CPU`)).toBeVisible();
        });

        await test.step('Backup Schedules step', async () => {
          //go to backups page
          await moveForward(page);
        });

        await test.step('Advanced Configuration step', async () => {
          //go to advanced configuration
          await moveForward(page);
          // Select Storage Class - mandatory param
          await advancedConfigurationSelectFirstStorageClass(page);
        });

        await test.step('Monitoring step', async () => {
          //go to monitoring
          await moveForward(page);
        });

        try {
          await test.step('Submit wizard', async () => {
            await submitWizard(page);
          });

          // await test.step('Check db list and status', async () => {
          //   await goToUrl(page, '/databases');
          //   // TODO: try re-enable after fix for: https://perconadev.atlassian.net/browse/EVEREST-1693
          //   if (db !== 'psmdb') {
          //     await waitForStatus(page, clusterName, 'Initializing', 15000);
          //   }
          //   await waitForStatus(page, clusterName, 'Up', 600000);
          // });

          await test.step('Wait for DB cluster creation', async () => {
            await expect(async () => {
              // new DB cluster appears in response not immediately
              const dbCluster = await getDbClusterAPI(
                clusterName,
                namespace,
                request,
                token
              );
              expect(dbCluster).toBeDefined();
              expect(dbCluster.status.status === 'ready').toBeTruthy();
            }).toPass({
              intervals: [1000],
              timeout: TIMEOUTS.FiveMinutes,
            });
          });

          await findDbAndClickRow(page, clusterName);

          await test.step('Upgrade db on next available version', async () => {
            //check upgrade btn and open modal
            const upgradeBtn = page.getByTestId('upgrade-db-btn');
            await expect(upgradeBtn).toBeVisible();
            await upgradeBtn.click();

            //populate upgrade version form
            await page.getByTestId('select-input-db-version').waitFor();
            await page.getByTestId('select-db-version-button').click();
            const dbVersionOptionsForUpgrade = page.getByRole('option');
            const selectedDbVersionValue = await dbVersionOptionsForUpgrade
              .first()
              .innerText();

            await dbVersionOptionsForUpgrade.first().click();

            //submit
            const upgradeModalBtn = page.getByTestId('form-dialog-upgrade');
            await expect(upgradeModalBtn).not.toBeDisabled();
            await upgradeModalBtn.click();
            await expect(
              page.getByTestId('upgrade-form-dialog')
            ).not.toBeVisible({
              timeout: 15000,
            });

            //check result
            await expect(page.getByTestId(`${clusterName}-status`)).toHaveText(
              'Upgrading',
              { timeout: 15000 }
            );
            await expect(
              page
                .getByTestId('version-overview-section-row')
                .filter({ hasText: selectedDbVersionValue })
            ).toBeVisible();
          });

          // TODO upgrading to last version leads to hiding upgradeButton
          // TODO list of upgrading versions should have only later versions
          // TODO upgrade button is hided during initialization

          await test.step('Wait for DB cluster upgrade', async () => {
            await expect(async () => {
              const dbCluster = await getDbClusterAPI(
                clusterName,
                namespace,
                request,
                token
              );
              expect(dbCluster).toBeDefined();
              expect(dbCluster.status.status === 'ready').toBeTruthy();
            }).toPass({
              intervals: [TIMEOUTS.FiveSeconds],
              timeout: TIMEOUTS.TwentyMinutes,
            });
          });

          await test.step('Check db list and status', async () => {
            await goToUrl(page, '/databases');
            await waitForStatus(
              page,
              clusterName,
              'Up',
              TIMEOUTS.ThirtySeconds
            );
          });
        } finally {
          await deleteDbClusterFn(request, clusterName, namespace);
        }
      });
    });
  }
);
