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

// Covers the deletion-policy behavior scoped in issue #2886:
// - Backup deletion policy: Delete / Retain
// - Instance deletion policy: Cascade / Orphan
// - UI deletion flow and statuses
// PITR, restore and (scheduled/on-demand) backup creation flows are covered
// by other suites and are out of scope here.

import { request as apiRequest, Page } from '@playwright/test';
import { expect, test } from '@e2e/fixtures/auth';
import {
  deleteDbCluster,
  gotoDbClusterBackups,
} from '@e2e/utils/db-clusters-list';
import { getCITokenFromLocalStorage } from '@e2e/utils/localStorage';
import { getClusterDetailedInfo } from '@e2e/utils/storage-class';
import {
  moveForward,
  submitWizard,
  populateBasicInformation,
  populateEngineResources,
  openDbCreationForm,
} from '@e2e/utils/db-wizard';
import { clickOnDemandBackup } from '@e2e/pr/db-cluster-details/utils';
import { EVEREST_CI_NAMESPACES, getBucketNamespacesMap } from '@e2e/constants';
import {
  waitForStatus,
  waitForDelete,
  findRowAndClickActions,
} from '@e2e/utils/table';

const PXC_PROVIDER_NAME = 'percona-xtradb-cluster';

let token: string;
let backupStorage: string;

function getBackupStorage(): string {
  const bucketNamespacesMap = getBucketNamespacesMap();
  const hasEverestTesting = bucketNamespacesMap.some(
    ([bucket, ns]) => bucket === 'everest-testing' && ns === 'everest'
  );

  if (
    process.env.EVEREST_S3_ACCESS_KEY &&
    process.env.EVEREST_S3_SECRET_KEY &&
    hasEverestTesting
  ) {
    return 'everest-testing';
  }
  return 'bucket-1';
}

test.describe.configure({ retries: 0 });

test.describe.serial(
  'Deletion policy',
  {
    tag: '@release',
  },
  () => {
    test.describe.configure({ timeout: 720000 });

    const db = 'pxc';
    const namespace = EVEREST_CI_NAMESPACES.EVEREST_UI;
    // A second, short-lived cluster is needed because an instance can only
    // be deleted once, and we need to exercise both Cascade and Orphan.
    const clusterName = 'pxc-1-delpolicy';
    const orphanClusterName = 'pxc-1-delpolicy-o';
    let storageClasses: string[] = [];

    test.beforeAll(async ({ request }) => {
      token = await getCITokenFromLocalStorage();

      const { storageClassNames = [] } = await getClusterDetailedInfo(
        token,
        request
      );
      storageClasses = storageClassNames;
      backupStorage = getBackupStorage();
    });

    // Best-effort teardown: delete the clusters via the API so the suite leaves nothing behind.
    // Runs even when a step failed — unlike a trailing "delete" test, which serial mode would
    // skip after a failure — keeping the shared namespace clean.
    test.afterAll(async () => {
      try {
        const cleanupToken = await getCITokenFromLocalStorage();
        const ctx = await apiRequest.newContext({
          baseURL: process.env.EVEREST_URL || 'http://localhost:8080',
        });
        for (const name of [clusterName, orphanClusterName]) {
          await ctx.delete(
            `/v1/clusters/main/namespaces/${namespace}/instances/${name}`,
            { headers: { Authorization: `Bearer ${cleanupToken}` } }
          );
        }
        await ctx.dispose();
      } catch {
        // Teardown must never fail the suite; any leftover cluster is removed by
        // the next run's pre-create delete.
      }
    });

    const createCluster = async (page: Page, name: string) => {
      expect(storageClasses.length).toBeGreaterThan(0);

      await page.goto('/databases');
      await openDbCreationForm(page, PXC_PROVIDER_NAME);

      await test.step('Populate basic information', async () => {
        await populateBasicInformation(
          page,
          namespace,
          name,
          db,
          storageClasses[0],
          false,
          ''
        );
        await moveForward(page);
      });

      await test.step('Populate resources', async () => {
        // Resources comes after Database Version in the PXC wizard.
        await moveForward(page);
        // The topology defaults (4 CPU / 8Gi / 100Gi per node) do not fit a CI runner.
        // Whole CPUs only: the wizard sends 0.6 as a JSON float, which the API rejects.
        await populateEngineResources(page, 1, 1, 1);
        await moveForward(page);
      });

      await test.step('Populate backups', async () => {
        await moveForward(page);
      });

      await test.step('Populate monitoring', async () => {
        const proxyHeading = page.getByRole('heading', { name: 'Proxy' });
        const monitoringPreview = page.getByText(/Monitoring endpoint:/i);

        if (await proxyHeading.isVisible().catch(() => false)) {
          await expect(proxyHeading).toBeVisible();
        } else if (await monitoringPreview.isVisible().catch(() => false)) {
          await expect(monitoringPreview).toBeVisible();
        }
        await moveForward(page);
      });

      await test.step('Submit wizard', async () => {
        await submitWizard(page);
      });

      await test.step('Check db list and status', async () => {
        await page.goto('/databases');
        await waitForStatus(page, name, 'Ready', 720000);
      });
    };

    const createBackup = async (
      page: Page,
      clusterUnderTest: string,
      backupName: string
    ) => {
      await gotoDbClusterBackups(page, clusterUnderTest);
      await clickOnDemandBackup(page);
      await page.getByTestId('text-input-name').fill(backupName);
      await expect(page.getByTestId('text-input-name')).not.toBeEmpty();

      const storageCombobox = page.getByRole('combobox', {
        name: /backup storage/i,
      });
      if (await storageCombobox.isVisible().catch(() => false)) {
        const currentStorage = (await storageCombobox.inputValue()).trim();
        if (backupStorage && currentStorage !== backupStorage) {
          await storageCombobox.click();
          await page.getByRole('option', { name: `${backupStorage}` }).click();
        }
        await expect(storageCombobox).toHaveValue(backupStorage);
      } else {
        await expect(
          page.getByTestId('text-input-storage-location')
        ).not.toBeEmpty();
      }

      const createButton = page
        .getByRole('dialog')
        .getByRole('button', { name: /^create$/i })
        .or(page.getByTestId('form-dialog-create'));
      await expect(createButton.first()).toBeVisible();
      await createButton.first().click();

      await waitForStatus(page, backupName, 'Succeeded', 360000);
    };

    test(`Cluster creation [${db} size 1]`, async ({ page }) => {
      await createCluster(page, clusterName);
    });

    test('Create backup for TC-01', async ({ page }) => {
      await createBackup(page, clusterName, 'delpolicy-delete');
    });

    test('TC-01/TC-06: Delete backup with policy = Delete', async ({
      page,
    }) => {
      await gotoDbClusterBackups(page, clusterName);

      const deleteRequest = page.waitForRequest(
        (req) =>
          req.method() === 'DELETE' &&
          req.url().includes('/backups/delpolicy-delete') &&
          req.url().includes('deletionPolicy=Delete')
      );

      await findRowAndClickActions(page, 'delpolicy-delete', 'Delete');
      await expect(page.getByLabel('Delete backup')).toBeVisible();
      // Check "Delete backups storage data": maps to policy = Delete.
      await page.getByTestId('checkbox-data-checkbox').click();
      await page.getByTestId('form-dialog-delete').click();

      // Confirms the Delete/Retain choice actually reaches the API as a
      // distinct deletionPolicy, not a fallback to the old cleanup flag.
      await deleteRequest;

      await test.step('Backup shows Deleting status and its Delete action is disabled', async () => {
        await waitForStatus(page, 'delpolicy-delete', 'Deleting', 15000);
        await findRowAndClickActions(page, 'delpolicy-delete');
        await expect(
          page.getByRole('menuitem', { name: 'Delete' })
        ).toBeDisabled();
        await page.keyboard.press('Escape');
      });

      await waitForDelete(page, 'delpolicy-delete', 60000);
    });

    test('Create backup for TC-02', async ({ page }) => {
      await createBackup(page, clusterName, 'delpolicy-retain');
    });

    test('TC-02: Delete backup with policy = Retain', async ({ page }) => {
      await gotoDbClusterBackups(page, clusterName);

      const deleteRequest = page.waitForRequest(
        (req) =>
          req.method() === 'DELETE' &&
          req.url().includes('/backups/delpolicy-retain') &&
          req.url().includes('deletionPolicy=Retain')
      );

      await findRowAndClickActions(page, 'delpolicy-retain', 'Delete');
      await expect(page.getByLabel('Delete backup')).toBeVisible();
      // Leave the "Delete backups storage data" checkbox unchecked: Retain.
      await page.getByTestId('form-dialog-delete').click();

      await deleteRequest;
      await waitForStatus(page, 'delpolicy-retain', 'Deleting', 15000);
      await waitForDelete(page, 'delpolicy-retain', 60000);
    });

    test('Create backup for TC-03/TC-04', async ({ page }) => {
      await createBackup(page, clusterName, 'delpolicy-cancel');
    });

    test('TC-03: Cancel backup delete', async ({ page }) => {
      await gotoDbClusterBackups(page, clusterName);
      await findRowAndClickActions(page, 'delpolicy-cancel', 'Delete');
      await expect(page.getByLabel('Delete backup')).toBeVisible();
      await page.getByTestId('form-dialog-cancel').click();

      await expect(page.getByLabel('Delete backup')).not.toBeVisible();
      const row = page.getByRole('row').filter({ hasText: 'delpolicy-cancel' });
      await expect(row.getByText('Succeeded', { exact: true })).toBeVisible();
      await expect(
        row.getByText('Deleting', { exact: true })
      ).not.toBeVisible();
    });

    test('TC-04: Delete instance with policy = Cascade', async ({ page }) => {
      const deleteRequest = page.waitForRequest(
        (req) =>
          req.method() === 'DELETE' &&
          req.url().includes(`/instances/${clusterName}`) &&
          req.url().includes('deletionPolicy=Cascade')
      );

      // "delpolicy-cancel" backup is still around, so the keep-backup-data
      // checkbox renders; leave it unchecked for Cascade semantics.
      await deleteDbCluster(page, clusterName, false);

      await deleteRequest;
      await waitForStatus(page, clusterName, 'Deleting', 15000);
      await waitForDelete(page, clusterName, 240000);
    });

    test(`Cluster creation for TC-05 [${db} size 1]`, async ({ page }) => {
      await createCluster(page, orphanClusterName);
    });

    test('Create backup for TC-05', async ({ page }) => {
      await createBackup(page, orphanClusterName, 'delpolicy-orphan');
    });

    test('TC-05: Delete instance with policy = Orphan', async ({ page }) => {
      const deleteRequest = page.waitForRequest(
        (req) =>
          req.method() === 'DELETE' &&
          req.url().includes(`/instances/${orphanClusterName}`) &&
          req.url().includes('deletionPolicy=Orphan')
      );

      // Check "Keep backups storage data": maps to policy = Orphan.
      await deleteDbCluster(page, orphanClusterName, true);

      await deleteRequest;
      await waitForStatus(page, orphanClusterName, 'Deleting', 15000);
      await waitForDelete(page, orphanClusterName, 240000);
    });
  }
);
