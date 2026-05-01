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

import { EVEREST_CI_NAMESPACES } from '@e2e/constants';
import { getEnginesVersions } from '@e2e/utils/database-engines';
import {
  deleteDbCluster,
  findDbAndClickRow,
} from '@e2e/utils/db-clusters-list';
import { getTokenFromLocalStorage } from '@e2e/utils/localStorage';
import { goToStep, moveForward, submitWizard } from '@e2e/utils/db-wizard';
import { waitForDelete, waitForInitializingState } from '@e2e/utils/table';
import { expect, test } from '@playwright/test';
import { selectDbEngine } from '../db-wizard-utils';

const namespace = EVEREST_CI_NAMESPACES.EVEREST_UI;
const PROXY_CONFIG_VALUE = 'max_connections=100';

test.describe('Proxy Configuration field', () => {
  let engineVersions = {
    pxc: [] as string[],
    psmdb: [] as string[],
    postgresql: [] as string[],
  };

  test.beforeAll(async ({ request }) => {
    const token = await getTokenFromLocalStorage();
    engineVersions = await getEnginesVersions(token, namespace, request);
  });

  test.beforeEach(async ({ page }) => {
    await page.goto('/databases');
  });

  // ─── PXC wizard ──────────────────────────────────────────────────────────

  test('PXC: proxy configuration field visible and persists to overview', async ({
    page,
  }) => {
    const clusterName = 'proxy-cfg-pxc';

    await selectDbEngine(page, 'mysql');
    await page.getByTestId('text-input-db-name').fill(clusterName);
    await moveForward(page);
    // Resources step
    await moveForward(page);
    // Backups step
    await moveForward(page);
    // Advanced Configurations step
    await goToStep(page, 4);

    // Proxy Configuration card should be visible for PXC
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
    await waitForDelete(page, clusterName, namespace);
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
    await goToStep(page, 4);

    const proxyConfigSwitch = page.getByTestId(
      'switch-input-proxy-config-enabled-label'
    );
    await expect(proxyConfigSwitch).toBeVisible();
    // Label should include "PG Bouncer"
    await expect(page.getByText('PG Bouncer Configuration')).toBeVisible();
  });

  // ─── PSMDB sharding gate ─────────────────────────────────────────────────

  test('PSMDB: proxy config field hidden without sharding, visible with sharding', async ({
    page,
  }) => {
    await selectDbEngine(page, 'mongodb');
    await page.getByTestId('text-input-db-name').fill('proxy-cfg-psmdb-check');
    await moveForward(page);
    await moveForward(page);
    await moveForward(page);
    await goToStep(page, 4);

    // Sharding is off by default → field should NOT be visible
    const proxyConfigSwitch = page.getByTestId(
      'switch-input-proxy-config-enabled-label'
    );
    await expect(proxyConfigSwitch).not.toBeVisible();

    // Go back to Basic Information to enable sharding
    await goToStep(page, 1);
    const shardingToggle = page.getByTestId('switch-input-sharding-label');
    if (await shardingToggle.isVisible()) {
      await shardingToggle.getByRole('checkbox').check();
    }
    await goToStep(page, 4);

    // Now the field should be visible
    await expect(proxyConfigSwitch).toBeVisible();
    await expect(page.getByText('Router Configuration')).toBeVisible();
  });

  // ─── Overview edit modal round-trip ──────────────────────────────────────

  test('PXC: proxy config field round-trips when re-opening edit modal', async ({
    page,
    request,
  }) => {
    const token = await getTokenFromLocalStorage();
    const clusterName = 'proxy-cfg-edit-pxc';

    // Create via API helper (skip UI wizard for speed)
    await page.goto(`/databases/${namespace}/${clusterName}/overview`);

    // Edit via Overview modal
    const editBtn = page.getByTestId('edit-advanced-configuration-db-btn');
    await editBtn.click();

    const proxyConfigSwitch = page.getByTestId(
      'switch-input-proxy-config-enabled-label'
    );
    await expect(proxyConfigSwitch).toBeVisible();

    // Enable and set value
    await proxyConfigSwitch.getByRole('checkbox').check();
    const configInput = page.getByTestId('text-input-proxy-config');
    await configInput.fill(PROXY_CONFIG_VALUE);
    await page.getByTestId('form-dialog-submit').click();

    // Re-open edit modal to check the value was persisted
    await editBtn.click();
    await proxyConfigSwitch.getByRole('checkbox').isChecked();
    await expect(configInput).toHaveValue(PROXY_CONFIG_VALUE);

    // Disable – config should be removed
    await proxyConfigSwitch.getByRole('checkbox').uncheck();
    await page.getByTestId('form-dialog-submit').click();

    // Re-open and confirm disabled
    await editBtn.click();
    await expect(proxyConfigSwitch.getByRole('checkbox')).not.toBeChecked();
  });
});
