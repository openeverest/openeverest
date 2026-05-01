// everest
// Copyright (C) 2023 Percona LLC
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

import {
  deleteDbCluster,
  findDbAndClickRow,
} from '@e2e/utils/db-clusters-list';
import { moveForward, submitWizard } from '@e2e/utils/db-wizard';
import { waitForDelete, waitForInitializingState } from '@e2e/utils/table';
import { expect, test } from '@playwright/test';
import { selectDbEngine } from '../db-wizard-utils';

const PROXY_CONFIG_VALUE = 'max_connections=100';

test.describe('Proxy Configuration field', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/databases');
  });

  // ─── PXC wizard ──────────────────────────────────────────────────────────

  test('PXC: proxy configuration field visible and persists to overview', async ({
    page,
  }) => {
    const clusterName = 'proxy-cfg-pxc';

    await selectDbEngine(page, 'pxc');
    await page.getByTestId('text-input-db-name').fill(clusterName);
    await moveForward(page);
    // Resources step
    await moveForward(page);
    // Backups step
    await moveForward(page);
    // Advanced Configurations step — now on step 4

    const proxyConfigSwitch = page.getByTestId(
      'switch-input-proxy-config-enabled-label'
    );
    await expect(proxyConfigSwitch).toBeVisible();

    // Enable and fill proxy config
    await proxyConfigSwitch.getByRole('checkbox').check();
    await page.getByTestId('text-input-proxy-config').fill(PROXY_CONFIG_VALUE);

    // Submit wizard
    await submitWizard(page);
    await waitForInitializingState(page, clusterName);

    // Navigate to cluster overview
    await findDbAndClickRow(page, clusterName);

    // Check that proxy config is shown on overview
    await expect(page.getByText('Proxy Configuration')).toBeVisible();
    await expect(page.getByText(PROXY_CONFIG_VALUE)).toBeVisible();

    // Clean up
    await page.goto('/databases');
    await deleteDbCluster(page, clusterName);
    await waitForDelete(page, clusterName, 60000);
  });

  // ─── PG wizard ───────────────────────────────────────────────────────────

  test('PostgreSQL: PG Bouncer Configuration field visible', async ({
    page,
  }) => {
    await selectDbEngine(page, 'postgresql');
    await page.getByTestId('text-input-db-name').fill('proxy-cfg-pg-check');
    await moveForward(page);
    await moveForward(page);
    await moveForward(page);
    // Advanced Configurations step — now on step 4

    const proxyConfigSwitch = page.getByTestId(
      'switch-input-proxy-config-enabled-label'
    );
    await expect(proxyConfigSwitch).toBeVisible();
    await expect(page.getByText('PG Bouncer Configuration')).toBeVisible();
  });

  // ─── PSMDB sharding gate ─────────────────────────────────────────────────

  test('PSMDB: proxy config field hidden without sharding, visible with sharding', async ({
    page,
  }) => {
    // Without sharding (default) → field must NOT be visible
    await selectDbEngine(page, 'psmdb');
    await page.getByTestId('text-input-db-name').fill('proxy-cfg-psmdb-check');
    await moveForward(page);
    await moveForward(page);
    await moveForward(page);
    // Advanced Configurations step — now on step 4

    const proxyConfigSwitch = page.getByTestId(
      'switch-input-proxy-config-enabled-label'
    );
    await expect(proxyConfigSwitch).not.toBeVisible();

    // Start fresh with sharding enabled
    await page.goto('/databases');
    await selectDbEngine(page, 'psmdb');
    await page.getByTestId('text-input-db-name').fill('proxy-cfg-psmdb-sh');
    await page
      .getByTestId('switch-input-sharding-label')
      .getByRole('checkbox')
      .check();
    await moveForward(page);
    await moveForward(page);
    await moveForward(page);
    // Advanced Configurations step — now on step 4

    // With sharding → field must be visible with the correct label
    await expect(proxyConfigSwitch).toBeVisible();
    await expect(page.getByText('Router Configuration')).toBeVisible();
  });

  // ─── Overview edit modal round-trip ──────────────────────────────────────

  test('PXC: proxy config round-trips in edit modal', async ({ page }) => {
    const clusterName = 'proxy-cfg-edit-pxc';

    // Create cluster via wizard with proxy config set
    await selectDbEngine(page, 'pxc');
    await page.getByTestId('text-input-db-name').fill(clusterName);
    await moveForward(page);
    await moveForward(page);
    await moveForward(page);
    // Advanced Configurations step — now on step 4

    const proxyConfigSwitch = page.getByTestId(
      'switch-input-proxy-config-enabled-label'
    );
    await proxyConfigSwitch.getByRole('checkbox').check();
    await page.getByTestId('text-input-proxy-config').fill(PROXY_CONFIG_VALUE);

    await submitWizard(page);
    await waitForInitializingState(page, clusterName);
    await findDbAndClickRow(page, clusterName);

    // Open edit modal — value should be pre-filled
    const editBtn = page.getByTestId('edit-advanced-configuration-db-btn');
    await editBtn.click();

    const configInput = page.getByTestId('text-input-proxy-config');
    await expect(proxyConfigSwitch.getByRole('checkbox')).toBeChecked();
    await expect(configInput).toHaveValue(PROXY_CONFIG_VALUE);

    // Disable toggle → config should be removed after save
    await proxyConfigSwitch.getByRole('checkbox').uncheck();
    await page.getByTestId('form-dialog-submit').click();

    // Re-open and confirm toggle is off
    await editBtn.click();
    await expect(proxyConfigSwitch.getByRole('checkbox')).not.toBeChecked();

    // Clean up
    await page.goto('/databases');
    await deleteDbCluster(page, clusterName);
    await waitForDelete(page, clusterName, 60000);
  });
});
