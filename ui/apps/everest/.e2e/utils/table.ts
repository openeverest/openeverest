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

import { Page, expect } from '@playwright/test';
import { TIMEOUTS } from '@e2e/constants';

export const findRowAndClickActions = async (
  page: Page,
  name: string,
  nameOfAction?: string
) => {
  // Cluster actions menu click. Test IDs changed across UI iterations,
  // so try the known variants in order.
  const row = page.locator('.MuiTableRow-root').filter({ hasText: name });
  await expect(row).toBeVisible({ timeout: 15000 });

  const menuCandidates = [
    row.getByTestId('row-actions-menu-button'),
    row.getByTestId('actions-menu-button'),
    row.getByTestId('MoreHorizIcon'),
  ];

  let clicked = false;
  for (const candidate of menuCandidates) {
    if (await candidate.isVisible().catch(() => false)) {
      await candidate.click({ timeout: 10000 });
      clicked = true;
      break;
    }
  }

  if (!clicked) {
    throw new Error(`Could not find row actions menu for table row: ${name}`);
  }

  if (nameOfAction) {
    await page.getByRole('menuitem', { name: nameOfAction }).click();
    await page.waitForLoadState('load', { timeout: TIMEOUTS.ThirtySeconds });
  }
};

// TODO remove after DB Cluster's PATCH method is implemented
export const waitForInitializingState = async (page: Page, name: string) => {
  const dbRow = page.getByRole('row').filter({ hasText: name });
  await expect(dbRow).toBeVisible();
  await expect(dbRow.getByText('Creating')).not.toBeVisible({ timeout: 60000 });
};

/**
 * Waits for specific object status in the table list
 * @param page Page object
 * @param name Name of the object to look for (database, backup)
 * @param status Desired status of the cluster
 * @param timeout How long to wait until error thrown
 */
export const waitForStatus = async (
  page: Page,
  name: string,
  status: string,
  timeout: number
) => {
  const dbRow = page.getByRole('row').filter({ hasText: name });
  await expect(dbRow).toBeVisible({ timeout: 15000 });
  await expect(dbRow.getByText(status, { exact: true })).toBeVisible({
    timeout: timeout,
  });
};

/**
 * Waits for specific object to be removed from the table
 * @param page Page object
 * @param name Name of the object to look for (database, backup)
 * @param timeout How long to wait until error thrown
 */
export const waitForDelete = async (
  page: Page,
  name: string,
  timeout: number
) => {
  await page.reload({ waitUntil: 'networkidle' });
  await expect(page.getByRole('row').getByText(name)).toHaveCount(0, {
    timeout: timeout,
  });
};

export const waitForDbListLoad = async (page: Page) => {
  const rows = page.locator('.MuiTableRow-root');
  const createDbButton = page.getByTestId('add-db-cluster-button');
  const start = Date.now();
  const timeout = TIMEOUTS.ThirtySeconds;

  while (Date.now() - start < timeout) {
    const [rowCount, hasCreateButton] = await Promise.all([
      rows.count(),
      createDbButton.isVisible().catch(() => false),
    ]);

    // The databases page is considered loaded when either table rows are shown
    // or the empty-state/toolbar create button is rendered.
    if (rowCount > 0 || hasCreateButton) {
      return;
    }

    await page.waitForTimeout(200); // pause before retrying
  }
  throw new Error('Timed out waiting for DB list to load');
};
