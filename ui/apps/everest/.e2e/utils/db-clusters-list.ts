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

import { APIRequestContext, expect, Page } from '@playwright/test';
import { findRowAndClickActions, waitForDbListLoad } from './table';
import { checkError } from '@e2e/utils/generic';
import { TIMEOUTS } from '@e2e/constants';
import { dismissOnboarding, loginCIUser } from './user';

const ensureLoggedInOnDatabasesPage = async (page: Page) => {
  await page.goto('/databases');
  await page.waitForLoadState('domcontentloaded');

  const isAuthenticated = async () =>
    page
      .getByTestId('user-appbar-button')
      .isVisible({ timeout: 1000 })
      .catch(() => false);

  const loginButton = page.getByTestId('login-button');
  const loginHeading = page.getByRole('heading', { name: 'Log in' });
  // Redirect to /login may happen asynchronously after initial navigation.
  // Wait until either authenticated shell or login screen is clearly visible.
  const start = Date.now();
  const settleTimeoutMs = 15000;
  while (Date.now() - start < settleTimeoutMs) {
    if (await isAuthenticated()) {
      return;
    }

    const headingVisible = await loginHeading
      .isVisible({ timeout: 1000 })
      .catch(() => false);
    const buttonVisible = await loginButton
      .isVisible({ timeout: 1000 })
      .catch(() => false);
    if (headingVisible || buttonVisible || page.url().includes('/login')) {
      break;
    }

    await page.waitForTimeout(300);
  }

  const isOnLoginPage =
    page.url().includes('/login') ||
    (await loginHeading.isVisible({ timeout: 3000 }).catch(() => false)) ||
    (await loginButton.isVisible({ timeout: 3000 }).catch(() => false));

  if (isOnLoginPage) {
    await loginCIUser(page);
    await dismissOnboarding(page);
    await page.goto('/databases');
    await page.waitForLoadState('domcontentloaded');
  }
};

export const getDbClustersListAPI = async (
  namespace: string,
  request: APIRequestContext,
  token: string
) => {
  const response = await request.get(
    `/v1/namespaces/${namespace}/database-clusters`,
    {
      headers: {
        Authorization: `Bearer ${token}`,
      },
    }
  );
  await checkError(response);

  return response.json();
};

export const findDbAndClickRow = async (page: Page, rowValue: string) => {
  const dbRow = page.getByRole('row').filter({ hasText: rowValue });
  await dbRow.click();
  await page.waitForLoadState('load', { timeout: TIMEOUTS.ThirtySeconds });
};

export const findDbAndClickActions = async (
  page: Page,
  dbName: string,
  nameOfAction?: string,
  status?: string
) => {
  page
    .getByTestId(`${dbName}-status`)
    .filter({ hasText: status || 'Initializing' });
  await findRowAndClickActions(page, dbName, nameOfAction);
};

export const gotoDbClusterBackups = async (page: Page, clusterName: string) => {
  await ensureLoggedInOnDatabasesPage(page);
  await waitForDbListLoad(page);

  const clusterRow = page.getByRole('row').filter({ hasText: clusterName });

  // Restored clusters may appear with a delay after the wizard completes.
  // Retry a few times with page reloads before failing the test.
  let found = false;
  for (let attempt = 0; attempt < 6; attempt += 1) {
    if ((await clusterRow.count()) > 0) {
      found = true;
      break;
    }

    await page.reload({ waitUntil: 'networkidle' });
    await waitForDbListLoad(page);
  }

  expect(found).toBeTruthy();
  await clusterRow.click();
  await expect(page.getByText('Overview')).toBeVisible();
  await page.getByTestId('backups').click();
};

export const gotoDbClusterRestores = async (
  page: Page,
  clusterName: string
) => {
  await ensureLoggedInOnDatabasesPage(page);
  await waitForDbListLoad(page);

  const clusterRow = page.getByRole('row').filter({ hasText: clusterName });
  await expect(clusterRow).toBeVisible({ timeout: TIMEOUTS.FiveMinutes });
  await clusterRow.click();
  await expect(page.getByText('Overview')).toBeVisible();
  await page.getByTestId('restores').click();
};

export const deleteDbCluster = async (page: Page, clusterName: string) => {
  await ensureLoggedInOnDatabasesPage(page);
  await waitForDbListLoad(page);
  await findDbAndClickActions(page, clusterName, 'delete', 'Up');
  await expect(page.getByText('Delete database')).toBeVisible();
  await expect(page.getByText('Irreversible action')).toBeVisible();
  await page.getByTestId('text-input-confirm-input').fill(clusterName);
  await page.getByTestId('form-dialog-delete').click();
};

export const suspendDbCluster = async (page: Page, clusterName: string) => {
  await ensureLoggedInOnDatabasesPage(page);
  await findDbAndClickActions(page, clusterName, 'suspend', 'Up');
};

export const resumeDbCluster = async (page: Page, clusterName: string) => {
  await ensureLoggedInOnDatabasesPage(page);
  await findDbAndClickActions(page, clusterName, 'resume', 'Paused');
};

export const restartDbCluster = async (page: Page, clusterName: string) => {
  await ensureLoggedInOnDatabasesPage(page);
  await findDbAndClickActions(page, clusterName, 'restart', 'Up');
};
