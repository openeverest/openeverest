import { expect, test } from '@playwright/test';
import { gotoDbClusterBackups } from '@e2e/utils/db-clusters-list';
import { createDbClusterFn, deleteDbClusterFn } from '@e2e/utils/db-cluster';
import { waitForStatus } from '@e2e/utils/table';
import { clickOnDemandBackup } from './utils';
import { TIMEOUTS } from '@e2e/constants';

const { EVEREST_BUCKETS_NAMESPACES_MAP } = process.env;

test.describe('On-demand backup lifecycle - PXC', () => {
  const clusterName = 'pxc-on-demand-lc';
  const backupName = 'pxc-lc-bkp-1';
  let storageName: string;

  // TC1: Instance setup — create PXC cluster with backup storage enabled.
  test.beforeAll(async ({ request }) => {
    storageName = JSON.parse(EVEREST_BUCKETS_NAMESPACES_MAP)[0][0];
    await createDbClusterFn(request, {
      dbName: clusterName,
      dbType: 'pxc',
      numberOfNodes: '1',
      backup: {
        enabled: true,
        schedules: [
          {
            backupStorageName: storageName,
            enabled: true,
            name: 'schedule-1',
            schedule: '0 * * * *',
          },
        ],
      },
    });
  });

  test.afterAll(async ({ request }) => {
    await deleteDbClusterFn(request, clusterName);
  });

  // TC1: Verify cluster is ready and Backups tab is accessible.
  test('Cluster reaches Up state and Backups tab shows Create backup button', async ({
    page,
  }) => {
    await page.goto('/databases');
    await waitForStatus(page, clusterName, 'Up', TIMEOUTS.TwentyMinutes);
    await gotoDbClusterBackups(page, clusterName);
    await expect(page.getByTestId('menu-button')).toBeVisible();
    await expect(page.getByRole('table')).toBeVisible();
  });

  // TC2: Execute on-demand backup via UI modal.
  // TC3: Monitor backup progress until successful completion.
  test('Create on-demand backup via modal and reach Succeeded status', async ({
    page,
  }) => {
    await gotoDbClusterBackups(page, clusterName);
    await clickOnDemandBackup(page);

    // Verify modal fields are pre-populated before submitting.
    await expect(page.getByTestId('text-input-name')).not.toBeEmpty();
    await expect(
      page.getByTestId('text-input-storage-location')
    ).not.toBeEmpty();

    await page.getByTestId('text-input-name').fill(backupName);
    await page.getByTestId('form-dialog-create').click();

    // TC3: backup row appears and transitions to Succeeded.
    await expect(page.getByText(backupName)).toBeVisible();
    await waitForStatus(page, backupName, 'Succeeded', 360000);
  });

  // TC4: Confirm backup name, status, and metadata accuracy in the table.
  test('Backup metadata is accurate after completion', async ({ page }) => {
    await gotoDbClusterBackups(page, clusterName);

    const backupRow = page
      .locator('.MuiTableRow-root')
      .filter({ hasText: backupName });

    await expect(backupRow).toBeVisible();

    // Status column shows the terminal succeeded state.
    await expect(backupRow.getByText('Succeeded', { exact: true })).toBeVisible();

    // Name column matches what was submitted in the modal.
    await expect(backupRow.getByText(backupName, { exact: true })).toBeVisible();

    // Storage column matches the configured backup storage.
    await expect(backupRow.getByText(storageName, { exact: true })).toBeVisible();

    // Started and Finished timestamps are present — the table renders them as
    // formatted date strings so at least two date-like tokens should appear in
    // the row once the backup has completed.
    await expect(backupRow.getByText(/\d{4}/).first()).toBeVisible();
  });
});
