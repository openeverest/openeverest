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

// Visual-regression baseline suite.
//
// Re-run trigger: touched to fire the pr:visual workflow after the release-2.0 merge.
//
// Every data-dependent screenshot renders from fixed fixtures (see ./mocks),
// so results are deterministic and independent of live cluster state. The
// `test` export from ./mocks/fixture already:
//   - authenticates (via @e2e/fixtures/auth),
//   - freezes wall-clock time,
//   - installs the default deterministic API mocks.
// Individual tests only add per-state overrides where a non-default response is
// required (empty list, instance carrying schedules, ...).

import { test, expect, baseTest } from './mocks/fixture';
import type { Page } from '@playwright/test';
import { goToUrl } from '@e2e/utils/generic';
import { TIMEOUTS } from '@e2e/constants';
import { openDbCreationForm } from '@e2e/utils/db-wizard';
import { DB_NAMESPACE, DB_NAME } from './mocks/data';
import {
  mockInstancesListEmptyRoute,
  mockInstanceDetailWithSchedules,
  mockMonitoringConfigsEmptyRoute,
} from './mocks/routes';

const screenshotOpts = {
  fullPage: true,
  maxDiffPixelRatio: 0.001,
};

// Helper: wait for a MRT table to finish loading
async function waitForTableContent(
  page: import('@playwright/test').Page,
  headerText: string
) {
  await page
    .locator(`th:has-text("${headerText}")`)
    .first()
    .waitFor({ state: 'visible', timeout: 60000 });
  await page.waitForTimeout(500);
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
    await expect(page).toHaveScreenshot(
      'settings-storage-locations.png',
      screenshotOpts
    );
  });

  test('Settings - Storage Locations - Add modal', async ({ page }) => {
    await goToUrl(page, '/settings/storage-locations');
    await waitForTableContent(page, 'Bucket');
    await page.getByTestId('add-backup-storage').click();
    await page
      .getByRole('dialog')
      .waitFor({ state: 'visible', timeout: 10000 });
    await page.waitForTimeout(300);
    await expect(page).toHaveScreenshot(
      'settings-storage-add-modal.png',
      screenshotOpts
    );
  });

  // --- SETTINGS: MONITORING ENDPOINTS ---
  test('Settings - Monitoring Endpoints', async ({ page }) => {
    await goToUrl(page, '/settings/monitoring-endpoints');
    await waitForTableContent(page, 'Endpoint');
    await expect(page).toHaveScreenshot(
      'settings-monitoring-endpoints.png',
      screenshotOpts
    );
  });

  test('Settings - Monitoring Endpoints - Add modal', async ({ page }) => {
    await goToUrl(page, '/settings/monitoring-endpoints');
    await waitForTableContent(page, 'Endpoint');
    await page.getByTestId('add-monitoring-endpoint').click();
    await page
      .getByRole('dialog')
      .waitFor({ state: 'visible', timeout: 10000 });
    await page.waitForTimeout(300);
    await expect(page).toHaveScreenshot(
      'settings-monitoring-add-modal.png',
      screenshotOpts
    );
  });

  // --- SETTINGS: NAMESPACES ---
  test('Settings - Namespaces', async ({ page }) => {
    await goToUrl(page, '/settings/namespaces');
    await waitForTableContent(page, 'Namespace');
    await expect(page).toHaveScreenshot(
      'settings-namespaces.png',
      screenshotOpts
    );
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
    await page
      .getByTestId('cluster-overview')
      .waitFor({ state: 'visible', timeout: 60000 });
    await page.waitForTimeout(500);
    await expect(page).toHaveScreenshot(
      'db-details-overview.png',
      screenshotOpts
    );

    // Backups tab
    await page.goto(`/databases/${DB_NAMESPACE}/${DB_NAME}/backups`);
    await page
      .locator('th:has-text("Status")')
      .or(page.getByText(/to start using backups/i))
      .first()
      .waitFor({ state: 'visible', timeout: 60000 });
    await page.waitForTimeout(500);
    await expect(page).toHaveScreenshot(
      'db-details-backups.png',
      screenshotOpts
    );

    // Restores tab
    await page.goto(`/databases/${DB_NAMESPACE}/${DB_NAME}/restores`);
    await page
      .locator('th:has-text("Status")')
      .or(page.getByText(/no restores/i))
      .first()
      .waitFor({ state: 'visible', timeout: 60000 });
    await page.waitForTimeout(500);
    await expect(page).toHaveScreenshot(
      'db-details-restores.png',
      screenshotOpts
    );
  });
});

// ============================================================
// TIER 2: Navigation & Layout
// ============================================================

test.describe('Visual Baseline - Navigation', () => {
  test.describe.configure({ timeout: TIMEOUTS.ThreeMinutes });

  test('Sidebar states', async ({ page }) => {
    await goToUrl(page, '/databases');
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
// ============================================================

test.describe('Visual Baseline - DB Wizard', () => {
  test.describe.configure({ timeout: TIMEOUTS.FiveMinutes });

  test('DB Wizard - all steps', async ({ page }) => {
    // Empty monitoring so the plain wizard monitoring step shows the no-endpoint state.
    await mockMonitoringConfigsEmptyRoute(page);

    await goToUrl(page, '/databases');
    await page
      .getByTestId('add-db-cluster-button')
      .waitFor({ state: 'visible', timeout: 60000 });
    await openDbCreationForm(page);
    await page
      .getByTestId('step-header')
      .waitFor({ state: 'visible', timeout: 60000 });
    await page.waitForTimeout(500);

    // Side drawer preview (on first step, shows all sections)
    await expect(page).toHaveScreenshot('db-wizard-with-drawer.png', {
      ...screenshotOpts,
      fullPage: false,
    });

    // Basic Information step (fullPage)
    await expect(page).toHaveScreenshot(
      'db-wizard-basic-info.png',
      screenshotOpts
    );

    // Navigate to Scheduled Backups (step 2)
    await page.getByRole('button', { name: /configure more/i }).click();
    await page.waitForTimeout(1000);
    await page
      .getByTestId('step-header')
      .waitFor({ state: 'visible', timeout: 10000 });
    await page.waitForTimeout(500);
    await expect(page).toHaveScreenshot(
      'db-wizard-backups.png',
      screenshotOpts
    );

    // Navigate to Database Version (step 3)
    await page.getByTestId('db-wizard-continue-button').click();
    await page.waitForTimeout(1000);
    await page
      .getByTestId('step-header')
      .waitFor({ state: 'visible', timeout: 10000 });
    await page.waitForTimeout(500);
    await expect(page).toHaveScreenshot(
      'db-wizard-db-version.png',
      screenshotOpts
    );

    // Navigate to Resources (step 4)
    await page.getByTestId('db-wizard-continue-button').click();
    await page.waitForTimeout(1000);
    await page
      .getByTestId('step-header')
      .waitFor({ state: 'visible', timeout: 10000 });
    await page.waitForTimeout(500);
    await expect(page).toHaveScreenshot(
      'db-wizard-resources.png',
      screenshotOpts
    );

    // Navigate to Monitoring (step 5)
    await page.getByTestId('db-wizard-continue-button').click();
    await page.waitForTimeout(1000);
    await page
      .getByTestId('step-header')
      .waitFor({ state: 'visible', timeout: 10000 });
    await page.waitForTimeout(500);
    await expect(page).toHaveScreenshot(
      'db-wizard-monitoring.png',
      screenshotOpts
    );

    // Navigate to Advanced Configuration (step 6)
    await page.getByTestId('db-wizard-continue-button').click();
    await page.waitForTimeout(1000);
    await page
      .getByTestId('step-header')
      .waitFor({ state: 'visible', timeout: 10000 });
    await page.waitForTimeout(500);
    await expect(page).toHaveScreenshot(
      'db-wizard-advanced.png',
      screenshotOpts
    );
  });
});

// ============================================================
// TIER 3: Additional states & edge cases
// ============================================================

test.describe('Visual Baseline - Additional States', () => {
  test.describe.configure({ timeout: TIMEOUTS.ThreeMinutes });

  test('404 page', async ({ page }) => {
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
// MOCKED STATES — data-driven screenshots served from fixtures
// ============================================================

test.describe('Visual Baseline - Mocked States', () => {
  test.describe.configure({ timeout: TIMEOUTS.FiveMinutes });

  test('DB Details - backups with data', async ({ page }) => {
    await goToUrl(page, `/databases/${DB_NAMESPACE}/${DB_NAME}/backups`);
    await page
      .locator('th:has-text("Status")')
      .first()
      .waitFor({ state: 'visible', timeout: 60000 });
    await page.waitForTimeout(500);
    await expect(page).toHaveScreenshot(
      'db-details-backups-with-data.png',
      screenshotOpts
    );
  });

  test('DB Wizard - monitoring with endpoint available', async ({ page }) => {
    // Default monitoring-configs fixture already exposes an endpoint.
    await goToUrl(page, '/databases');
    await page
      .getByTestId('add-db-cluster-button')
      .waitFor({ state: 'visible', timeout: 60000 });
    await openDbCreationForm(page);
    await page
      .getByTestId('step-header')
      .waitFor({ state: 'visible', timeout: 60000 });

    // Navigate through steps to reach Monitoring
    await page.getByRole('button', { name: /configure more/i }).click();
    await page.waitForTimeout(1000);
    for (let i = 0; i < 3; i++) {
      await page.getByTestId('db-wizard-continue-button').click();
      await page.waitForTimeout(1500);
    }
    await page
      .getByText(/monitoring/i)
      .first()
      .waitFor({ state: 'visible', timeout: 10000 });
    await page.waitForTimeout(500);
    const pageContent = await page.textContent('main');
    if (pageContent?.includes('Advanced configuration')) {
      await page.getByRole('button', { name: /previous/i }).click();
      await page.waitForTimeout(1000);
    }
    await expect(page).toHaveScreenshot(
      'db-wizard-monitoring-with-endpoint.png',
      screenshotOpts
    );
  });

  test('DB Wizard - backup schedule creation', async ({ page }) => {
    await goToUrl(page, '/databases');
    await page
      .getByTestId('add-db-cluster-button')
      .waitFor({ state: 'visible', timeout: 60000 });
    await openDbCreationForm(page);
    await page
      .getByTestId('step-header')
      .waitFor({ state: 'visible', timeout: 60000 });

    // Navigate to Backups step (first step after "Configure more")
    await page.getByRole('button', { name: /configure more/i }).click();
    await page.waitForTimeout(1000);
    await page
      .getByTestId('step-header')
      .waitFor({ state: 'visible', timeout: 10000 });
    await page.waitForTimeout(500);

    // Screenshot of backup step with storage available
    await expect(page).toHaveScreenshot(
      'db-wizard-backups-with-storage.png',
      screenshotOpts
    );

    // Open the "Create backup schedule" dialog if available
    const createScheduleBtn = page.getByTestId('create-schedule');
    if (
      await createScheduleBtn.isVisible({ timeout: 5000 }).catch(() => false)
    ) {
      await createScheduleBtn.click();
      await page
        .getByRole('dialog')
        .waitFor({ state: 'visible', timeout: 10000 });
      await page.waitForTimeout(500);
      await expect(page).toHaveScreenshot(
        'db-wizard-schedule-create-modal.png',
        screenshotOpts
      );
    }
  });

  test('DB Details - edit section modal', async ({ page }) => {
    await goToUrl(page, `/databases/${DB_NAMESPACE}/${DB_NAME}`);
    await page
      .getByTestId('cluster-overview')
      .waitFor({ state: 'visible', timeout: 60000 });
    await page.waitForTimeout(1000);

    // Click the first schema-driven section edit button ({sectionKey}-edit-button).
    const editButton = page.locator('[data-testid$="-edit-button"]').first();
    await editButton.waitFor({ state: 'visible', timeout: 10000 });
    await editButton.click();
    await page
      .getByRole('dialog')
      .waitFor({ state: 'visible', timeout: 10000 });
    await page.waitForTimeout(500);
    await expect(page).toHaveScreenshot(
      'db-details-edit-section-modal.png',
      screenshotOpts
    );
  });

  test('DB Details - backups with schedules expanded', async ({ page }) => {
    // Override the instance detail to carry backup schedules.
    await mockInstanceDetailWithSchedules(page);

    await goToUrl(page, `/databases/${DB_NAMESPACE}/${DB_NAME}/backups`);
    await page
      .locator('th:has-text("Status")')
      .first()
      .waitFor({ state: 'visible', timeout: 60000 });
    await page.waitForTimeout(500);

    // Expand the schedules section
    const schedulesBtn = page.getByTestId('scheduled-backups');
    await schedulesBtn.waitFor({ state: 'visible', timeout: 10000 });
    await schedulesBtn.click();
    await page
      .locator('[data-testid^="schedule-"]')
      .first()
      .waitFor({ state: 'visible', timeout: 10000 });
    await page.waitForTimeout(500);
    await expect(page).toHaveScreenshot(
      'db-details-backups-schedules-expanded.png',
      screenshotOpts
    );
  });

  test('Confirm delete schedule dialog', async ({ page }) => {
    await mockInstanceDetailWithSchedules(page);

    await goToUrl(page, `/databases/${DB_NAMESPACE}/${DB_NAME}/backups`);
    await page
      .locator('th:has-text("Status")')
      .first()
      .waitFor({ state: 'visible', timeout: 60000 });
    await page.waitForTimeout(500);

    const schedulesBtn = page.getByTestId('scheduled-backups');
    await schedulesBtn.waitFor({ state: 'visible', timeout: 10000 });
    await schedulesBtn.click();
    await page
      .locator('[data-testid^="schedule-"]')
      .first()
      .waitFor({ state: 'visible', timeout: 10000 });
    await page.waitForTimeout(300);

    await page.getByTestId('delete-schedule-button').first().click();
    await page
      .getByRole('dialog')
      .waitFor({ state: 'visible', timeout: 10000 });
    await page.waitForTimeout(500);
    await expect(page).toHaveScreenshot(
      'confirm-delete-schedule-dialog.png',
      screenshotOpts
    );
  });

  test('DB Details - create schedule form dialog', async ({ page }) => {
    await mockInstanceDetailWithSchedules(page);

    await goToUrl(page, `/databases/${DB_NAMESPACE}/${DB_NAME}/backups`);
    await page
      .locator('th:has-text("Status")')
      .first()
      .waitFor({ state: 'visible', timeout: 60000 });
    await page.waitForTimeout(500);

    // Open "Create backup" menu and click "Schedule"
    const createBackupBtn = page.getByRole('button', {
      name: /create backup/i,
    });
    await createBackupBtn.waitFor({ state: 'visible', timeout: 10000 });
    await createBackupBtn.click();
    await page
      .getByTestId('schedule-menu-item')
      .waitFor({ state: 'visible', timeout: 5000 });
    await page.getByTestId('schedule-menu-item').click();
    await page
      .getByRole('dialog')
      .waitFor({ state: 'visible', timeout: 10000 });
    await page.waitForTimeout(500);
    await expect(page).toHaveScreenshot(
      'db-details-schedule-form-dialog.png',
      screenshotOpts
    );
  });

  test('DB Details - schedule form with autocomplete open', async ({
    page,
  }) => {
    await mockInstanceDetailWithSchedules(page);

    await goToUrl(page, `/databases/${DB_NAMESPACE}/${DB_NAME}/backups`);
    await page
      .locator('th:has-text("Status")')
      .first()
      .waitFor({ state: 'visible', timeout: 60000 });
    await page.waitForTimeout(500);

    // Open "Create backup" menu → "Schedule"
    const createBackupBtn = page.getByRole('button', {
      name: /create backup/i,
    });
    await createBackupBtn.waitFor({ state: 'visible', timeout: 10000 });
    await createBackupBtn.click();
    await page
      .getByTestId('schedule-menu-item')
      .waitFor({ state: 'visible', timeout: 5000 });
    await page.getByTestId('schedule-menu-item').click();
    await page
      .getByRole('dialog')
      .waitFor({ state: 'visible', timeout: 10000 });
    await page.waitForTimeout(1000);

    // Force-open the backup-storage autocomplete dropdown
    const popupIcon = page
      .getByRole('dialog')
      .getByRole('button', { name: 'Open' })
      .first();
    await popupIcon.waitFor({ state: 'visible', timeout: 10000 });
    await popupIcon.click();
    await page
      .getByRole('listbox')
      .waitFor({ state: 'visible', timeout: 5000 });
    await page.waitForTimeout(300);
    await expect(page).toHaveScreenshot(
      'schedule-form-autocomplete-open.png',
      screenshotOpts
    );
  });

  test('Custom confirm dialog - delete backup', async ({ page }) => {
    await goToUrl(page, `/databases/${DB_NAMESPACE}/${DB_NAME}/backups`);
    await page
      .locator('th:has-text("Status")')
      .first()
      .waitFor({ state: 'visible', timeout: 60000 });
    await page.waitForTimeout(500);

    // Open the actions menu on the first backup row and then Delete
    const actionsBtn = page.getByTestId('row-actions-menu-button').first();
    await actionsBtn.waitFor({ state: 'visible', timeout: 10000 });
    await actionsBtn.click();
    await page
      .getByTestId('row-actions-menu')
      .waitFor({ state: 'visible', timeout: 5000 });
    await page.waitForTimeout(300);

    const deleteItem = page.getByRole('menuitem', { name: /delete/i });
    await deleteItem.waitFor({ state: 'visible', timeout: 5000 });
    await deleteItem.click();
    await page
      .getByRole('dialog')
      .waitFor({ state: 'visible', timeout: 10000 });
    await page.waitForTimeout(500);
    await expect(page).toHaveScreenshot(
      'confirm-delete-backup-dialog.png',
      screenshotOpts
    );
  });
});

// ============================================================
// EMPTY STATE — mocked empty database list
// ============================================================

test.describe('Visual Baseline - Empty States', () => {
  test.describe.configure({ timeout: TIMEOUTS.ThreeMinutes });

  test('Databases list - empty state', async ({ page }) => {
    await mockInstancesListEmptyRoute(page);

    await goToUrl(page, '/databases');
    await page
      .locator('th:has-text("Database name")')
      .or(page.getByText(/no database clusters/i))
      .or(page.getByText(/no results/i))
      .first()
      .waitFor({ state: 'visible', timeout: 60000 });
    await page.waitForTimeout(500);
    await expect(page).toHaveScreenshot(
      'databases-list-empty.png',
      screenshotOpts
    );
  });
});

// ============================================================
// LOGIN PAGE — requires unauthenticated state (no mocks / no clock)
// ============================================================

baseTest.describe('Visual Baseline - Login (unauthenticated)', () => {
  baseTest.use({ storageState: { cookies: [], origins: [] } });

  baseTest('Login page', async ({ page }) => {
    await page.goto('/login');
    await page.getByTestId('text-input-username').waitFor({ state: 'visible' });
    await page.waitForTimeout(500);
    await expect(page).toHaveScreenshot('login-page.png', {
      ...screenshotOpts,
      maxDiffPixelRatio: 0.01,
    });
  });
});
