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
  findDbAndClickRow,
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
import { waitForStatus, waitForDelete } from '@e2e/utils/table';
import { getDbClusterAPI } from '@e2e/utils/db-cluster';
import { shouldExecuteDBCombination } from '@e2e/utils/generic';

let token: string;

test.describe.configure({ retries: 0 });

[
  { db: 'psmdb', size: 3 },
  { db: 'pxc', size: 3 },
  { db: 'postgresql', size: 3 },
].forEach(({ db, size }) => {
  test.describe(
    'Storage scaling',
    {
      tag: '@release',
    },
    () => {
      test.skip(!shouldExecuteDBCombination(db, size));
      test.describe.configure({ timeout: 720000 });

      const clusterName = `${db}-${size}-scale`;

      let storageClasses = [];
      const namespace = EVEREST_CI_NAMESPACES.EVEREST_UI;
      const monitoringName = 'e2e-endpoint-0';

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
          await populateResources(page, 0.6, 1, 2, size);
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
          expect(addedCluster?.spec.engine.storage.size.toString()).toBe('2Gi');
          expect(addedCluster?.spec.proxy.expose.type).toBe('ClusterIP');
          if (db != 'psmdb') {
            expect(addedCluster?.spec.proxy.replicas).toBe(size);
          }
        });
      });

      test(`Validate storage scaling [${db} size ${size}]`, async ({
        page,
        request,
      }) => {
        await page.goto('/databases');
        await findDbAndClickRow(page, clusterName);
        await page.getByTestId('edit-resources-button').click();

        await test.step('Attempt to decrease disk size', async () => {
          const diskInput = page.getByTestId('text-input-disk');
          const saveButton = page.getByTestId('form-dialog-save');

          // Ensure the initial value is correct
          await expect(diskInput).toHaveValue(/2(\.0)?/);

          // Attempt to decrease the disk size
          await diskInput.fill('1');

          // Verify that the save button is disabled
          await expect(saveButton).toBeDisabled();

          // Verify that the error message is displayed
          await expect(
            page.locator('text=Descaling is not allowed')
          ).toBeVisible();
        });

        await test.step('Test invalid values', async () => {
          const diskInput = page.getByTestId('text-input-disk');
          const saveButton = page.getByTestId('form-dialog-save');

          // Attempt to enter empty disk size
          await diskInput.fill('');

          // Verify that the save button is disabled
          await expect(saveButton).toBeDisabled();

          // Verify that the error message is displayed
          await expect(
            page.locator('text=String must contain at least 1 character(s)')
          ).toBeVisible();

          // Attempt to enter empty disk size
          await diskInput.fill('5.5');

          // Verify that the save button is disabled
          await expect(saveButton).toBeDisabled();

          // Verify that the error message is displayed
          await expect(
            page.locator('text=Disk size must be an integer number')
          ).toBeVisible();

          // Attempt to enter letter as disk size
          await diskInput.fill('a');

          // Verify that the save button is disabled
          await expect(saveButton).toBeDisabled();

          // Verify that the error message is displayed
          await expect(
            page.locator('text=Please enter a valid number')
          ).toBeVisible();
        });

        await test.step('Increase disk size', async () => {
          const diskInput = page.getByTestId('text-input-disk');
          const saveButton = page.getByTestId('form-dialog-save');

          // Increase the disk size
          await diskInput.fill('5');

          // Verify that the warning message is displayed
          await expect(
            page.locator(
              'text=Disk upscaling is irreversible and may temporarily block further resize actions until complete.'
            )
          ).toBeVisible();

          // Verify that the save button is enabled
          await expect(saveButton).toBeEnabled();

          // Save the changes
          await saveButton.click();

          // Verify that the modal is closed
          await expect(
            page.locator('[data-testid="resources-edit-modal"]')
          ).toBeHidden();

          // Verify that the edit resources button is disabled while resizing
          await expect(page.getByTestId('edit-resources-button')).toBeDisabled({
            timeout: 30000,
          });

          // Check DB status to be Up
          await page.goto('/databases');
          await waitForStatus(page, clusterName, 'Resizing volumes', 60000);
          await waitForStatus(page, clusterName, 'Up', 660000);
        });

        await test.step('Check db cluster k8s object options', async () => {
          const addedCluster = await getDbClusterAPI(
            clusterName,
            EVEREST_CI_NAMESPACES.EVEREST_UI,
            request,
            token
          );

          expect(addedCluster?.spec.engine.storage.size.toString()).toBe('5Gi');
        });
      });

      test(`Delete cluster [${db} size ${size}]`, async ({ page }) => {
        await deleteDbCluster(page, clusterName);
        await waitForStatus(page, clusterName, 'Deleting', 15000);
        await waitForDelete(page, clusterName, 240000);
      });
    }
  );
});
