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
import { getTokenFromLocalStorage } from '@e2e/utils/localStorage';
import {
  deleteDbCluster,
  findDbAndClickRow,
} from '@e2e/utils/db-clusters-list';
import { getClusterDetailedInfo } from '@e2e/utils/storage-class';
import { moveForward, submitWizard } from '@e2e/utils/db-wizard';
import { waitForStatus, waitForDelete } from '@e2e/utils/table';
import { selectDbEngine } from '@e2e/pr/db-cluster/db-wizard/db-wizard-utils';

let token: string;

[
  { db: 'psmdb', size: 1 },
  // { db: 'pxc', size: 1 },
  // { db: 'postgresql', size: 1 },
].forEach(
  ({ db, size }: { db: 'pxc' | 'psmdb' | 'postgresql'; size: number }) => {
    test.describe('Overview page', () => {
      test.describe.configure({ timeout: 900000 });

      const clusterName = `${db}-${size}-upgrade`;

      let storageClasses = [];

      test.beforeAll(async ({ request }) => {
        token = await getTokenFromLocalStorage();

        const { storageClassNames = [] } = await getClusterDetailedInfo(
          token,
          request
        );
        storageClasses = storageClassNames;
      });

      test(`Upgrade ${db} version`, async ({ page, request }) => {
        expect(storageClasses.length).toBeGreaterThan(0);

        await page.goto('/databases');
        await selectDbEngine(page, db);

        await test.step('Populate basic info', async () => {
          await page.getByTestId('text-input-db-name').fill(clusterName);

          await page.getByTestId('select-input-db-version').waitFor();
          await page.getByTestId('select-db-version-button').click();
          const dbVersionOptions = await page.getByRole('option');

          await dbVersionOptions.last().click();
          // go to resources page
          await moveForward(page);
        });

        await test.step('Populate resources', async () => {
          await page
            .getByRole('button')
            .getByText(size + ' node')
            .click();
          await expect(page.getByText(`1 node:`)).toBeVisible();
        });

        await test.step('Move forward form with default values', async () => {
          //go to backups page
          await moveForward(page);
          //go to advanced configuration
          await moveForward(page);
          //go to monitoring
          await moveForward(page);
        });

        await test.step('Submit form', async () => {
          await submitWizard(page);
        });

        await test.step('Check db list and status', async () => {
          await page.goto('/databases');
          // TODO: try re-enable after fix for: https://perconadev.atlassian.net/browse/EVEREST-1693
          if (db !== 'psmdb') {
            await waitForStatus(page, clusterName, 'Initializing', 15000);
          }
          await waitForStatus(page, clusterName, 'Up', 600000);
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
          const dbVersionOptionsForUpgrade = await page.getByRole('option');
          const selectedDbVersionValue = await dbVersionOptionsForUpgrade
            .nth(2)
            .innerText();

          await dbVersionOptionsForUpgrade.nth(2).click();

          //submit
          const upgradeModalBtn = page.getByTestId('form-dialog-upgrade');
          expect(upgradeModalBtn).not.toBeDisabled();
          await upgradeModalBtn.click();
          await expect(page.getByTestId('upgrade-form-dialog')).not.toBeVisible(
            {
              timeout: 15000,
            }
          );

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

        await test.step('Check db list and status', async () => {
          await page.goto('/databases');
          await waitForStatus(page, clusterName, 'Up', 720000);
        });
      });

      test(`Delete cluster [${db} size ${size}]`, async ({ page }) => {
        await deleteDbCluster(page, clusterName);
        await waitForDelete(page, clusterName, 300000);
      });
    });
  }
);
