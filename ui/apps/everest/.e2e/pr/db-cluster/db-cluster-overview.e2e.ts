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

import { test, expect } from '@playwright/test';
import { createDbClusterFn, deleteDbClusterFn } from '@e2e/utils/db-cluster';
import { getLastAvailableEngineVersion } from '@e2e/utils/database-engines';
import { getTokenFromLocalStorage } from '@e2e/utils/localStorage';
import { getNamespacesFn } from '@e2e/utils/namespaces';
import { EVEREST_CI_NAMESPACES } from '@e2e/constants';
import { findDbAndClickRow } from '@e2e/utils/db-clusters-list';

test.describe('DB Cluster Overview', async () => {
  const dbClusterName = 'cluster-overview-test';
  let mysqlVersion: string;

  test.beforeAll(async ({ request }) => {
    const token = await getTokenFromLocalStorage();
    const namespaces = await getNamespacesFn(token, request);
    const namespace = namespaces[0];
    mysqlVersion = await getLastAvailableEngineVersion(
      token,
      namespace,
      request,
      'mysql'
    );
    // MySQL 8.4.x uses 3GB memory for "Small" preset, other versions use 2GB
    const isMySQL84x = /^8\.4\./.test(mysqlVersion);
    const smallMemory = isMySQL84x ? 3 : 2;
    await createDbClusterFn(request, {
      dbName: dbClusterName,
      dbVersion: mysqlVersion,
      dbType: 'mysql',
      numberOfNodes: '1',
      cpu: 1,
      memory: smallMemory,
      proxyCpu: 0.5,
      proxyMemory: 0.8,
      externalAccess: true,
      sourceRanges: [
        {
          sourceRange: 'http://192.168.1.1',
        },
      ],
    });
  });

  test.beforeEach(async ({ page }) => {
    await page.goto('/databases');
  });

  test.afterAll(async ({ request }) => {
    await deleteDbClusterFn(request, dbClusterName);
  });

  test('Overview information', async ({ page }) => {
    await findDbAndClickRow(page, dbClusterName);

    await expect(
      page.getByRole('heading', {
        name: dbClusterName,
      })
    ).toBeVisible();

    await expect(page.getByTestId(`${dbClusterName}-status`)).toBeVisible();

    await expect(
      page.getByTestId('basic-information-overview-section')
    ).toBeVisible();
    await expect(
      page.getByTestId('type-overview-section-row').filter({ hasText: 'MySQL' })
    ).toBeVisible();
    await expect(
      page
        .getByTestId('name-overview-section-row')
        .filter({ hasText: `${dbClusterName}` })
    ).toBeVisible();
    await expect(
      page
        .getByTestId('namespace-overview-section-row')
        .filter({ hasText: `${EVEREST_CI_NAMESPACES.EVEREST_UI}` })
    ).toBeVisible();
    await expect(
      page.getByTestId('type-overview-section-row').filter({ hasText: 'MySQL' })
    ).toBeVisible();

    await expect(
      page.getByTestId('connection-details-overview-section')
    ).toBeVisible();

    await expect(page.getByTestId('monitoring-overview-section')).toBeVisible();

    await expect(
      page.getByTestId('advanced-configuration-overview-section')
    ).toBeVisible();
    await expect(
      page
        .getByTestId('ext.access-overview-section-row')
        .filter({ hasText: 'Enabled' })
    ).toBeVisible();
  });

  test('Show the correct resources during editing', async ({ page }) => {
    await findDbAndClickRow(page, dbClusterName);
    await page.getByTestId('edit-resources-button').click();
    await expect(
      page.getByTestId('node-resources-toggle-button-small')
    ).toHaveAttribute('aria-pressed', 'true');
    await page.getByTestId('proxies-accordion').click();
    await expect(
      page.getByTestId('proxy-resources-toggle-button-medium')
    ).toHaveAttribute('aria-pressed', 'true');
  });

  test('Delete Action', async ({ page, request }) => {
    const dbName = 'delete-test';

    await createDbClusterFn(request, {
      dbName: dbName,
      dbType: 'mysql',
      numberOfNodes: '1',
    });

    const row = page.getByText(dbName);
    await row.click();

    await expect(
      page.getByRole('heading', {
        name: dbName,
      })
    ).toBeVisible();

    const actionButton = page.getByTestId('actions-button');
    await actionButton.click();

    const deleteButton = page.getByTestId(`${dbName}-delete`);
    await deleteButton.click();

    await page.getByTestId(`${dbName}-form-dialog`).waitFor();
    await expect(page.getByTestId('irreversible-action-alert')).toBeVisible();
    const deleteConfirmationButton = page
      .getByRole('button')
      .filter({ hasText: 'Delete' });
    await expect(deleteConfirmationButton).toBeDisabled();
    await page.getByTestId('text-input-confirm-input').fill(dbName);
    await expect(deleteConfirmationButton).toBeEnabled();
    await deleteConfirmationButton.click();
  });
});
