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

import { test, expect } from '@e2e/fixtures/auth';
import { test as baseTest } from '@playwright/test';
import { goToUrl } from '@e2e/utils/generic';
import { TIMEOUTS } from '@e2e/constants';
import { openDbCreationForm } from '@e2e/utils/db-wizard';

// Database cluster for detail page screenshots
const DB_NAMESPACE = 'default';
const DB_NAME = 'inst-u3y';

const screenshotOpts = {
  fullPage: true,
  maxDiffPixelRatio: 0.01,
};

// Helper: wait for a MRT table to finish loading
async function waitForTableContent(page: import('@playwright/test').Page, headerText: string) {
  await page.locator(`th:has-text("${headerText}")`).first().waitFor({ state: 'visible', timeout: 60000 });
  await page.waitForTimeout(500);
}

// Helper: re-login if session expired (token TTL may have elapsed after prior tests)
async function ensureAuthenticated(page: import('@playwright/test').Page) {
  // Wait briefly for potential redirect to login page (client-side redirect after token check)
  const redirected = await page.waitForURL('**/login*', { timeout: 5000 }).then(() => true).catch(() => false);
  if (redirected) {
    await page.getByTestId('text-input-username').waitFor({ state: 'visible', timeout: 5000 });
    await page.getByTestId('text-input-username').fill(process.env.CI_USER || 'admin');
    await page.getByTestId('text-input-password').fill(process.env.CI_PASSWORD || 'admin');
    await page.getByTestId('login-button').click();
    // Dismiss welcome modal if it appears after fresh login
    const letsGoBtn = page.getByTestId('lets-go-button');
    if (await letsGoBtn.isVisible({ timeout: 3000 }).catch(() => false)) {
      await letsGoBtn.click();
    }
    await page.waitForTimeout(1000);
  }
}

// ============================================================
// TIER 1: CRITICAL — Tables & Settings pages
// ============================================================

test.describe('Visual Baseline - Pages', () => {
  test.describe.configure({ timeout: TIMEOUTS.ThreeMinutes });

  // --- DATABASE LIST ---
  test('Databases list page', async ({ page }) => {
    await goToUrl(page, '/databases');
    await waitForTableContent(page, 'Database name');
    await expect(page).toHaveScreenshot('databases-list.png', screenshotOpts);
  });

  // --- SETTINGS: STORAGE LOCATIONS ---
  test('Settings - Storage Locations', async ({ page }) => {
    await goToUrl(page, '/settings/storage-locations');
    await waitForTableContent(page, 'Bucket');
    await expect(page).toHaveScreenshot('settings-storage-locations.png', screenshotOpts);
  });

  test('Settings - Storage Locations - Add modal', async ({ page }) => {
    await goToUrl(page, '/settings/storage-locations');
    await waitForTableContent(page, 'Bucket');
    await page.getByTestId('add-backup-storage').click();
    await page.getByRole('dialog').waitFor({ state: 'visible', timeout: 10000 });
    await page.waitForTimeout(300);
    await expect(page).toHaveScreenshot('settings-storage-add-modal.png', screenshotOpts);
  });

  // --- SETTINGS: MONITORING ENDPOINTS ---
  test('Settings - Monitoring Endpoints', async ({ page }) => {
    await goToUrl(page, '/settings/monitoring-endpoints');
    await waitForTableContent(page, 'Endpoint');
    await expect(page).toHaveScreenshot('settings-monitoring-endpoints.png', screenshotOpts);
  });

  test('Settings - Monitoring Endpoints - Add modal', async ({ page }) => {
    await goToUrl(page, '/settings/monitoring-endpoints');
    await waitForTableContent(page, 'Endpoint');
    await page.getByTestId('add-monitoring-endpoint').click();
    await page.getByRole('dialog').waitFor({ state: 'visible', timeout: 10000 });
    await page.waitForTimeout(300);
    await expect(page).toHaveScreenshot('settings-monitoring-add-modal.png', screenshotOpts);
  });

  // --- SETTINGS: NAMESPACES ---
  test('Settings - Namespaces', async ({ page }) => {
    await goToUrl(page, '/settings/namespaces');
    await waitForTableContent(page, 'Namespace');
    await expect(page).toHaveScreenshot('settings-namespaces.png', screenshotOpts);
  });
});

// ============================================================
// TIER 2: HIGH — DB Details (Tabs, Grid layout, nested content)
// ============================================================

test.describe('Visual Baseline - DB Cluster Details', () => {
  test.describe.configure({ timeout: TIMEOUTS.FiveMinutes });

  test('DB Details - all tabs', async ({ page }) => {
    // Overview tab
    await goToUrl(page, `/databases/${DB_NAMESPACE}/${DB_NAME}`);
    await ensureAuthenticated(page);
    await page.getByTestId('cluster-overview').waitFor({ state: 'visible', timeout: 60000 });
    await page.waitForTimeout(500);
    await expect(page).toHaveScreenshot('db-details-overview.png', screenshotOpts);

    // Backups tab
    await page.goto(`/databases/${DB_NAMESPACE}/${DB_NAME}/backups`);
    await page.locator('th:has-text("Status")').or(
      page.getByText(/to start using backups/i)
    ).first().waitFor({ state: 'visible', timeout: 60000 });
    await page.waitForTimeout(500);
    await expect(page).toHaveScreenshot('db-details-backups.png', screenshotOpts);

    // Restores tab
    await page.goto(`/databases/${DB_NAMESPACE}/${DB_NAME}/restores`);
    await page.locator('th:has-text("Status")').or(
      page.getByText(/no restores/i)
    ).first().waitFor({ state: 'visible', timeout: 60000 });
    await page.waitForTimeout(500);
    await expect(page).toHaveScreenshot('db-details-restores.png', screenshotOpts);
  });
});

// ============================================================
// TIER 2: Navigation & Layout
// ============================================================

test.describe('Visual Baseline - Navigation', () => {
  test.describe.configure({ timeout: TIMEOUTS.ThreeMinutes });

  test('Sidebar states', async ({ page }) => {
    await goToUrl(page, '/databases');
    await ensureAuthenticated(page);
    await waitForTableContent(page, 'Database name');

    // Collapsed state
    await expect(page).toHaveScreenshot('sidebar-collapsed.png', {
      ...screenshotOpts,
      fullPage: false,
    });

    // Expanded state — click the drawer toggle button
    const expandButton = page.getByTestId('open-drawer-button');
    await expandButton.waitFor({ state: 'visible', timeout: 10000 });
    await expandButton.click();
    await page.waitForTimeout(500);
    await expect(page).toHaveScreenshot('sidebar-expanded.png', {
      ...screenshotOpts,
      fullPage: false,
    });
  });
});

// ============================================================
// TIER 1: CRITICAL — Database Creation Wizard (all steps)
// NOTE: This test is last among authenticated tests because it takes ~40s
// and each test gets a fresh page with the original auth token.
// ============================================================

test.describe('Visual Baseline - DB Wizard', () => {
  test.describe.configure({ timeout: TIMEOUTS.FiveMinutes });

  test('DB Wizard - all steps', async ({ page }) => {
    await goToUrl(page, '/databases');
    await ensureAuthenticated(page);
    await page.getByTestId('add-db-cluster-button').waitFor({ state: 'visible', timeout: 60000 });
    await openDbCreationForm(page);
    await page.getByTestId('step-header').waitFor({ state: 'visible', timeout: 60000 });
    await page.waitForTimeout(500);

    // Side drawer preview (on first step, shows all sections)
    await expect(page).toHaveScreenshot('db-wizard-with-drawer.png', {
      ...screenshotOpts,
      fullPage: false,
    });

    // Basic Information step (fullPage)
    await expect(page).toHaveScreenshot('db-wizard-basic-info.png', screenshotOpts);

    // Navigate to Scheduled Backups (step 2)
    await page.getByRole('button', { name: /configure more/i }).click();
    await page.waitForTimeout(1000);
    await page.getByTestId('step-header').waitFor({ state: 'visible', timeout: 10000 });
    await page.waitForTimeout(500);
    await expect(page).toHaveScreenshot('db-wizard-backups.png', screenshotOpts);

    // Navigate to Database Version (step 3)
    await page.getByTestId('db-wizard-continue-button').click();
    await page.waitForTimeout(1000);
    await page.getByTestId('step-header').waitFor({ state: 'visible', timeout: 10000 });
    await page.waitForTimeout(500);
    await expect(page).toHaveScreenshot('db-wizard-db-version.png', screenshotOpts);

    // Navigate to Resources (step 4)
    await page.getByTestId('db-wizard-continue-button').click();
    await page.waitForTimeout(1000);
    await page.getByTestId('step-header').waitFor({ state: 'visible', timeout: 10000 });
    await page.waitForTimeout(500);
    await expect(page).toHaveScreenshot('db-wizard-resources.png', screenshotOpts);

    // Navigate to Monitoring (step 5)
    await page.getByTestId('db-wizard-continue-button').click();
    await page.waitForTimeout(1000);
    await page.getByTestId('step-header').waitFor({ state: 'visible', timeout: 10000 });
    await page.waitForTimeout(500);
    await expect(page).toHaveScreenshot('db-wizard-monitoring.png', screenshotOpts);

    // Open "Add monitoring endpoint" dialog from wizard
    const addMonitoringBtn = page.getByRole('button', { name: /add monitoring endpoint/i });
    if (await addMonitoringBtn.isVisible({ timeout: 3000 }).catch(() => false)) {
      await addMonitoringBtn.click();
      await page.getByRole('dialog').waitFor({ state: 'visible', timeout: 10000 });
      await page.waitForTimeout(500);
      await expect(page).toHaveScreenshot('db-wizard-monitoring-add-modal.png', screenshotOpts);
      // Close the dialog
      await page.getByRole('dialog').getByRole('button', { name: /cancel/i }).click();
      await page.waitForTimeout(300);
    }

    // Navigate to Advanced Configuration (step 6)
    await page.getByTestId('db-wizard-continue-button').click();
    await page.waitForTimeout(1000);
    await page.getByTestId('step-header').waitFor({ state: 'visible', timeout: 10000 });
    await page.waitForTimeout(500);
    await expect(page).toHaveScreenshot('db-wizard-advanced.png', screenshotOpts);
  });
});

// ============================================================
// TIER 3: Additional states & edge cases
// ============================================================

test.describe('Visual Baseline - Additional States', () => {
  test.describe.configure({ timeout: TIMEOUTS.ThreeMinutes });

  test('404 page', async ({ page }) => {
    // First ensure we're authenticated (navigate to a known page)
    await goToUrl(page, '/databases');
    await ensureAuthenticated(page);
    // Now navigate to the non-existent page
    await goToUrl(page, '/nonexistent-page-for-visual-test');
    await page.waitForTimeout(1000);
    await expect(page).toHaveScreenshot('404-page.png', {
      ...screenshotOpts,
      maxDiffPixelRatio: 0.02,
      timeout: 15000,
    });
  });
});

// ============================================================
// MOCKED STATES — Screenshots with mock API data for states
// that require specific backend data (monitoring, backups, schedules)
// ============================================================

test.describe('Visual Baseline - Mocked States', () => {
  test.describe.configure({ timeout: TIMEOUTS.FiveMinutes });

  // Mock data for monitoring configs
  const mockMonitoringConfigs = {
    apiVersion: 'monitoring.openeverest.io/v1alpha1',
    kind: 'MonitoringConfigList',
    items: [{
      apiVersion: 'monitoring.openeverest.io/v1alpha1',
      kind: 'MonitoringConfig',
      metadata: { name: 'pmm-prod', namespace: 'default' },
      spec: {
        type: 'pmm',
        pmm: {
          credentialsSecretName: 'monitoring-config-pmm-prod-credentials',
          url: 'http://monitoring.example.com',
          verifyTLS: false,
        },
      },
      status: { inUse: false, lastObservedGeneration: 1 },
    }],
  };

  // Mock data for backup storages
  const mockBackupStorages = {
    apiVersion: 'storage.openeverest.io/v1alpha1',
    kind: 'BackupStorageList',
    items: [{
      apiVersion: 'storage.openeverest.io/v1alpha1',
      kind: 'BackupStorage',
      metadata: { name: 's3-prod-backups', namespace: 'default' },
      spec: {
        type: 's3',
        s3: {
          bucket: 'everest-backups-prod',
          region: 'us-east-1',
          credentialsSecretName: 'backup-storage-s3-prod-credentials',
          endpointURL: 'https://s3.amazonaws.com',
          verifyTLS: true,
          forcePathStyle: false,
        },
      },
      status: {},
    }],
  };

  // Mock data for backups list
  const mockBackups = {
    apiVersion: 'backup.openeverest.io/v1alpha1',
    kind: 'BackupList',
    items: [
      {
        apiVersion: 'backup.openeverest.io/v1alpha1',
        kind: 'Backup',
        metadata: { name: 'daily-backup-2026-06-29', namespace: 'default' },
        spec: {
          instanceName: 'inst-u3y',
          backupClassName: 'psmdb-backup-class',
          storageName: 's3-prod-backups',
          scheduleName: 'daily-full',
          deletionPolicy: 'Delete',
        },
        status: {
          state: 'Succeeded',
          startedAt: '2026-06-29T00:00:05Z',
          completedAt: '2026-06-29T00:05:30Z',
          size: '1.2Gi',
          executionMode: 'ProviderManaged',
        },
      },
      {
        apiVersion: 'backup.openeverest.io/v1alpha1',
        kind: 'Backup',
        metadata: { name: 'daily-backup-2026-06-28', namespace: 'default' },
        spec: {
          instanceName: 'inst-u3y',
          backupClassName: 'psmdb-backup-class',
          storageName: 's3-prod-backups',
          scheduleName: 'daily-full',
          deletionPolicy: 'Delete',
        },
        status: {
          state: 'Succeeded',
          startedAt: '2026-06-28T00:00:05Z',
          completedAt: '2026-06-28T00:04:12Z',
          size: '1.1Gi',
          executionMode: 'ProviderManaged',
        },
      },
      {
        apiVersion: 'backup.openeverest.io/v1alpha1',
        kind: 'Backup',
        metadata: { name: 'on-demand-backup-001', namespace: 'default' },
        spec: {
          instanceName: 'inst-u3y',
          backupClassName: 'psmdb-backup-class',
          storageName: 's3-prod-backups',
          deletionPolicy: 'Delete',
        },
        status: {
          state: 'Running',
          startedAt: '2026-06-29T10:30:00Z',
          executionMode: 'ProviderManaged',
        },
      },
    ],
  };

  test('DB Details - backups with data', async ({ page }) => {
    // Mock backup storages and backups list
    await page.route(/\/backup-storages$/, async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(mockBackupStorages) });
    });
    await page.route(/\/instances\/[^/]+\/backups$/, async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(mockBackups) });
    });

    await goToUrl(page, `/databases/${DB_NAMESPACE}/${DB_NAME}/backups`);
    await ensureAuthenticated(page);
    await page.locator('th:has-text("Status")').first().waitFor({ state: 'visible', timeout: 60000 });
    await page.waitForTimeout(500);
    await expect(page).toHaveScreenshot('db-details-backups-with-data.png', screenshotOpts);
  });

  test('DB Wizard - monitoring with endpoint available', async ({ page }) => {
    // Mock monitoring configs to show the monitoring step with available endpoint
    await page.route(/\/monitoring-configs$/, async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(mockMonitoringConfigs) });
    });

    await goToUrl(page, '/databases');
    await ensureAuthenticated(page);
    await page.getByTestId('add-db-cluster-button').waitFor({ state: 'visible', timeout: 60000 });
    await openDbCreationForm(page);
    await page.getByTestId('step-header').waitFor({ state: 'visible', timeout: 60000 });

    // Navigate through all steps to reach Monitoring
    await page.getByRole('button', { name: /configure more/i }).click();
    await page.waitForTimeout(1000);
    // Skip to monitoring (step order: backups → db-version → resources → monitoring)
    for (let i = 0; i < 3; i++) {
      await page.getByTestId('db-wizard-continue-button').click();
      await page.waitForTimeout(1500);
    }
    // Wait specifically for the Monitoring step content
    await page.getByText(/monitoring/i).first().waitFor({ state: 'visible', timeout: 10000 });
    await page.waitForTimeout(500);
    // Verify we're on the monitoring step, not advanced config
    const pageContent = await page.textContent('main');
    if (pageContent?.includes('Advanced configuration')) {
      // Went one step too far — go back
      await page.getByRole('button', { name: /previous/i }).click();
      await page.waitForTimeout(1000);
    }
    await expect(page).toHaveScreenshot('db-wizard-monitoring-with-endpoint.png', screenshotOpts);
  });

  test('DB Wizard - backup schedule creation', async ({ page }) => {
    // Mock backup storages AND backup classes so the backup step appears and shows schedule form
    await page.route(/\/backup-storages$/, async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(mockBackupStorages) });
    });
    await page.route(/\/backup-classes$/, async (route) => {
      await route.fulfill({
        status: 200, contentType: 'application/json',
        body: JSON.stringify({
          items: [{
            metadata: { name: 'psmdb-backup-class' },
            spec: { providers: [{ name: 'percona-server-mongodb' }], storages: [{ name: 's3-storage', backupStorageRef: { name: 's3-prod-backups' } }] },
          }],
        }),
      });
    });

    await goToUrl(page, '/databases');
    await ensureAuthenticated(page);
    await page.getByTestId('add-db-cluster-button').waitFor({ state: 'visible', timeout: 60000 });
    await openDbCreationForm(page);
    await page.getByTestId('step-header').waitFor({ state: 'visible', timeout: 60000 });

    // Navigate to Backups step (first step after "Configure more")
    await page.getByRole('button', { name: /configure more/i }).click();
    await page.waitForTimeout(1000);
    await page.getByTestId('step-header').waitFor({ state: 'visible', timeout: 10000 });
    await page.waitForTimeout(500);

    // Screenshot of backup step with storage available
    await expect(page).toHaveScreenshot('db-wizard-backups-with-storage.png', screenshotOpts);

    // Click "Create backup schedule" button if visible
    const createScheduleBtn = page.getByTestId('create-schedule');
    if (await createScheduleBtn.isVisible({ timeout: 5000 }).catch(() => false)) {
      await createScheduleBtn.click();
      await page.getByRole('dialog').waitFor({ state: 'visible', timeout: 10000 });
      await page.waitForTimeout(500);
      await expect(page).toHaveScreenshot('db-wizard-schedule-create-modal.png', screenshotOpts);
    }
  });

  test('DB Details - edit section modal', async ({ page }) => {
    await goToUrl(page, `/databases/${DB_NAMESPACE}/${DB_NAME}`);
    await ensureAuthenticated(page);
    // Re-navigate after auth in case redirect changed the URL
    if (!page.url().includes(DB_NAME)) {
      await goToUrl(page, `/databases/${DB_NAMESPACE}/${DB_NAME}`);
    }
    await page.getByTestId('cluster-overview').waitFor({ state: 'visible', timeout: 60000 });
    await page.waitForTimeout(1000);

    // Click first available edit button (format: {sectionKey}-edit-button)
    const editButton = page.locator('[data-testid$="-edit-button"]').first();
    await editButton.waitFor({ state: 'visible', timeout: 10000 });
    await editButton.click();
    await page.getByRole('dialog').waitFor({ state: 'visible', timeout: 10000 });
    await page.waitForTimeout(500);
    await expect(page).toHaveScreenshot('db-details-edit-section-modal.png', screenshotOpts);
  });

  test('DB Details - backups with schedules expanded', async ({ page }) => {
    // Mock the instance response to include backup schedules
    await page.route(/\/instances\/inst-u3y$/, async (route) => {
      const response = await route.fetch();
      const body = await response.json();
      // Inject schedules into the instance spec
      body.spec = body.spec || {};
      body.spec.backup = {
        ...body.spec.backup,
        enabled: true,
        classRef: { name: 'psmdb-backup-class' },
        storages: [{
          storageRef: { name: 's3-prod-backups' },
          schedules: [
            { name: 'daily-full', cron: '0 0 * * *', enabled: true, retentionCopies: 7 },
            { name: 'weekly-full', cron: '0 0 * * 0', enabled: true, retentionCopies: 4 },
          ],
        }],
      };
      await route.fulfill({ response, json: body });
    });
    await page.route(/\/backup-storages$/, async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(mockBackupStorages) });
    });
    await page.route(/\/instances\/[^/]+\/backups$/, async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(mockBackups) });
    });
    await page.route(/\/backup-classes$/, async (route) => {
      await route.fulfill({
        status: 200, contentType: 'application/json',
        body: JSON.stringify({ items: [{ metadata: { name: 'psmdb-backup-class' }, spec: { providers: [{ name: 'percona-server-mongodb' }], storages: [{ name: 's3-storage', backupStorageRef: { name: 's3-prod-backups' } }] } }] }),
      });
    });

    await goToUrl(page, `/databases/${DB_NAMESPACE}/${DB_NAME}/backups`);
    await ensureAuthenticated(page);
    if (!page.url().includes(DB_NAME)) {
      await goToUrl(page, `/databases/${DB_NAMESPACE}/${DB_NAME}/backups`);
    }
    await page.locator('th:has-text("Status")').first().waitFor({ state: 'visible', timeout: 60000 });
    await page.waitForTimeout(500);

    // Expand the schedules section
    const schedulesBtn = page.getByTestId('scheduled-backups');
    await schedulesBtn.waitFor({ state: 'visible', timeout: 10000 });
    await schedulesBtn.click();
    await page.locator('[data-testid^="schedule-"]').first().waitFor({ state: 'visible', timeout: 10000 });
    await page.waitForTimeout(500);
    await expect(page).toHaveScreenshot('db-details-backups-schedules-expanded.png', screenshotOpts);
  });

  test('Confirm delete schedule dialog', async ({ page }) => {
    // Same mocked state with schedules
    await page.route(/\/instances\/inst-u3y$/, async (route) => {
      const response = await route.fetch();
      const body = await response.json();
      body.spec = body.spec || {};
      body.spec.backup = {
        ...body.spec.backup,
        enabled: true,
        classRef: { name: 'psmdb-backup-class' },
        storages: [{
          storageRef: { name: 's3-prod-backups' },
          schedules: [
            { name: 'daily-full', cron: '0 0 * * *', enabled: true, retentionCopies: 7 },
            { name: 'weekly-full', cron: '0 0 * * 0', enabled: true, retentionCopies: 4 },
          ],
        }],
      };
      await route.fulfill({ response, json: body });
    });
    await page.route(/\/backup-storages$/, async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(mockBackupStorages) });
    });
    await page.route(/\/instances\/[^/]+\/backups$/, async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(mockBackups) });
    });
    await page.route(/\/backup-classes$/, async (route) => {
      await route.fulfill({
        status: 200, contentType: 'application/json',
        body: JSON.stringify({ items: [{ metadata: { name: 'psmdb-backup-class' }, spec: { providers: [{ name: 'percona-server-mongodb' }], storages: [{ name: 's3-storage', backupStorageRef: { name: 's3-prod-backups' } }] } }] }),
      });
    });

    await goToUrl(page, `/databases/${DB_NAMESPACE}/${DB_NAME}/backups`);
    await ensureAuthenticated(page);
    if (!page.url().includes(DB_NAME)) {
      await goToUrl(page, `/databases/${DB_NAMESPACE}/${DB_NAME}/backups`);
    }
    await page.locator('th:has-text("Status")').first().waitFor({ state: 'visible', timeout: 60000 });
    await page.waitForTimeout(500);

    // Expand schedules and click delete
    const schedulesBtn = page.getByTestId('scheduled-backups');
    await schedulesBtn.waitFor({ state: 'visible', timeout: 10000 });
    await schedulesBtn.click();
    await page.locator('[data-testid^="schedule-"]').first().waitFor({ state: 'visible', timeout: 10000 });
    await page.waitForTimeout(300);

    // Click delete on first schedule
    await page.getByTestId('delete-schedule-button').first().click();
    await page.getByRole('dialog').waitFor({ state: 'visible', timeout: 10000 });
    await page.waitForTimeout(500);
    await expect(page).toHaveScreenshot('confirm-delete-schedule-dialog.png', screenshotOpts);
  });

  test('DB Details - create schedule form dialog', async ({ page }) => {
    // Mock instance with backup config and backup classes/storages for the schedule form
    await page.route(/\/instances\/inst-u3y$/, async (route) => {
      const response = await route.fetch();
      const body = await response.json();
      body.spec = body.spec || {};
      body.spec.backup = {
        ...body.spec.backup,
        enabled: true,
        classRef: { name: 'psmdb-backup-class' },
        storages: [{
          storageRef: { name: 's3-prod-backups' },
          schedules: [
            { name: 'daily-full', cron: '0 0 * * *', enabled: true, retentionCopies: 7 },
          ],
        }],
      };
      await route.fulfill({ response, json: body });
    });
    await page.route(/\/backup-storages$/, async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(mockBackupStorages) });
    });
    await page.route(/\/instances\/[^/]+\/backups$/, async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(mockBackups) });
    });
    await page.route(/\/backup-classes$/, async (route) => {
      await route.fulfill({
        status: 200, contentType: 'application/json',
        body: JSON.stringify({ items: [{ metadata: { name: 'psmdb-backup-class' }, spec: { providers: [{ name: 'percona-server-mongodb' }], storages: [{ name: 's3-storage', backupStorageRef: { name: 's3-prod-backups' } }] } }] }),
      });
    });

    await goToUrl(page, `/databases/${DB_NAMESPACE}/${DB_NAME}/backups`);
    await ensureAuthenticated(page);
    if (!page.url().includes(DB_NAME)) {
      await goToUrl(page, `/databases/${DB_NAMESPACE}/${DB_NAME}/backups`);
    }
    await page.locator('th:has-text("Status")').first().waitFor({ state: 'visible', timeout: 60000 });
    await page.waitForTimeout(500);

    // Open "Create backup" menu and click "Schedule"
    const createBackupBtn = page.getByRole('button', { name: /create backup/i });
    await createBackupBtn.waitFor({ state: 'visible', timeout: 10000 });
    await createBackupBtn.click();
    await page.getByTestId('schedule-menu-item').waitFor({ state: 'visible', timeout: 5000 });
    await page.getByTestId('schedule-menu-item').click();
    await page.getByRole('dialog').waitFor({ state: 'visible', timeout: 10000 });
    await page.waitForTimeout(500);
    await expect(page).toHaveScreenshot('db-details-schedule-form-dialog.png', screenshotOpts);
  });

  test('DB Details - schedule form with autocomplete open', async ({ page }) => {
    // Mock instance with backup config and backup classes/storages — then open the
    // schedule dialog and expand the backup storage autocomplete dropdown
    await page.route(/\/instances\/inst-u3y$/, async (route) => {
      const response = await route.fetch();
      const body = await response.json();
      body.spec = body.spec || {};
      body.spec.backup = {
        ...body.spec.backup,
        enabled: true,
        classRef: { name: 'psmdb-backup-class' },
        storages: [],
      };
      await route.fulfill({ response, json: body });
    });
    await page.route(/\/backup-storages$/, async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(mockBackupStorages) });
    });
    await page.route(/\/instances\/[^/]+\/backups$/, async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(mockBackups) });
    });
    await page.route(/\/backup-classes$/, async (route) => {
      await route.fulfill({
        status: 200, contentType: 'application/json',
        body: JSON.stringify({ items: [{ metadata: { name: 'psmdb-backup-class' }, spec: { providers: [{ name: 'percona-server-mongodb' }], storages: [{ name: 's3-storage', backupStorageRef: { name: 's3-prod-backups' } }] } }] }),
      });
    });

    await goToUrl(page, `/databases/${DB_NAMESPACE}/${DB_NAME}/backups`);
    await ensureAuthenticated(page);
    if (!page.url().includes(DB_NAME)) {
      await goToUrl(page, `/databases/${DB_NAMESPACE}/${DB_NAME}/backups`);
    }
    await page.locator('th:has-text("Status")').first().waitFor({ state: 'visible', timeout: 60000 });
    await page.waitForTimeout(500);

    // Open "Create backup" menu → "Schedule"
    const createBackupBtn = page.getByRole('button', { name: /create backup/i });
    await createBackupBtn.waitFor({ state: 'visible', timeout: 10000 });
    await createBackupBtn.click();
    await page.getByTestId('schedule-menu-item').waitFor({ state: 'visible', timeout: 5000 });
    await page.getByTestId('schedule-menu-item').click();
    await page.getByRole('dialog').waitFor({ state: 'visible', timeout: 10000 });
    await page.waitForTimeout(1000);

    // Click the autocomplete popup icon (Open button) to force-open dropdown
    const popupIcon = page.getByRole('dialog').getByRole('button', { name: 'Open' }).first();
    await popupIcon.waitFor({ state: 'visible', timeout: 10000 });
    await popupIcon.click();
    await page.getByRole('listbox').waitFor({ state: 'visible', timeout: 5000 });
    await page.waitForTimeout(300);
    await expect(page).toHaveScreenshot('schedule-form-autocomplete-open.png', screenshotOpts);
  });

  test('Custom confirm dialog - delete backup', async ({ page }) => {
    // Mock backups with data to enable the delete action
    await page.route(/\/backup-storages$/, async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(mockBackupStorages) });
    });
    await page.route(/\/instances\/[^/]+\/backups$/, async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(mockBackups) });
    });
    await page.route(/\/backup-classes$/, async (route) => {
      await route.fulfill({
        status: 200, contentType: 'application/json',
        body: JSON.stringify({ items: [{ metadata: { name: 'psmdb-backup-class' }, spec: { providers: [{ name: 'percona-server-mongodb' }], storages: [{ name: 's3-storage', backupStorageRef: { name: 's3-prod-backups' } }] } }] }),
      });
    });

    await goToUrl(page, `/databases/${DB_NAMESPACE}/${DB_NAME}/backups`);
    await ensureAuthenticated(page);
    if (!page.url().includes(DB_NAME)) {
      await goToUrl(page, `/databases/${DB_NAMESPACE}/${DB_NAME}/backups`);
    }
    await page.locator('th:has-text("Status")').first().waitFor({ state: 'visible', timeout: 60000 });
    await page.waitForTimeout(500);

    // Click the actions menu on the first backup row and then Delete
    const actionsBtn = page.getByTestId('row-actions-menu-button').first();
    await actionsBtn.waitFor({ state: 'visible', timeout: 10000 });
    await actionsBtn.click();
    await page.getByTestId('row-actions-menu').waitFor({ state: 'visible', timeout: 5000 });
    await page.waitForTimeout(300);

    // Click Delete menu item
    const deleteItem = page.getByRole('menuitem', { name: /delete/i });
    await deleteItem.waitFor({ state: 'visible', timeout: 5000 });
    await deleteItem.click();
    await page.getByRole('dialog').waitFor({ state: 'visible', timeout: 10000 });
    await page.waitForTimeout(500);
    await expect(page).toHaveScreenshot('confirm-delete-backup-dialog.png', screenshotOpts);
  });
});

// ============================================================
// EMPTY STATE — Mocks API response to simulate empty database list
// ============================================================

test.describe('Visual Baseline - Empty States', () => {
  test.describe.configure({ timeout: TIMEOUTS.ThreeMinutes });

  test('Databases list - empty state', async ({ page }) => {
    // Intercept database instances list API and return empty response
    await page.route(/\/namespaces\/[^/]+\/instances$/, async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([]),
      });
    });

    await goToUrl(page, '/databases');
    await ensureAuthenticated(page);
    // Wait for the empty table to render (header should still appear)
    await page.locator('th:has-text("Database name")').or(
      page.getByText(/no database clusters/i)
    ).or(
      page.getByText(/no results/i)
    ).first().waitFor({ state: 'visible', timeout: 60000 });
    await page.waitForTimeout(500);
    await expect(page).toHaveScreenshot('databases-list-empty.png', screenshotOpts);
  });
});

// ============================================================
// LOGIN PAGE — Requires unauthenticated state
// ============================================================

baseTest.describe('Visual Baseline - Login (unauthenticated)', () => {
  baseTest.use({ storageState: { cookies: [], origins: [] } });

  baseTest('Login page', async ({ page }) => {
    await page.goto('/login');
    await page.getByTestId('text-input-username').waitFor({ state: 'visible' });
    await page.waitForTimeout(500);
    await expect(page).toHaveScreenshot('login-page.png', {
      ...screenshotOpts,
      maxDiffPixelRatio: 0.02,
    });
  });
});
