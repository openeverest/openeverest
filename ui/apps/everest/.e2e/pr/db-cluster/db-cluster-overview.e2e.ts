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

import { test, expect } from '@playwright/test';
import {
  createDbClusterFn,
  deleteDbClusterFn,
  getDbClusterAPI,
} from '@e2e/utils/db-cluster';
import { EVEREST_CI_NAMESPACES, TIMEOUTS } from '@e2e/constants';
import { findDbAndClickRow } from '@e2e/utils/db-clusters-list';
import { goToUrl, limitedSuffixedName } from '@e2e/utils/generic';
import { getCITokenFromLocalStorage } from '@e2e/utils/localStorage';

const dbClusterName = 'pr-db-ovw';

// TODO  dbVersion: '8.0.36-28.1' was added in later tests versions to beforeAll functions, need to be checked why and do we need return it,
test.describe.parallel('DB cluster overview', async () => {
  test.beforeEach(async ({ page }) => {
    await goToUrl(page, '/databases');
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
      page
        .getByTestId('type-overview-section-row')
        .filter({ hasText: 'PostgreSQL' })
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
      page
        .getByTestId('type-overview-section-row')
        .filter({ hasText: 'PostgreSQL' })
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
      page.getByTestId('PG Bouncer-resources-toggle-button-custom')
    ).toHaveAttribute('aria-pressed', 'true');
  });

  test('Delete Action', async ({ page, request }) => {
    const dbName = limitedSuffixedName(dbClusterName + '-del'),
      namespace = EVEREST_CI_NAMESPACES.EVEREST_UI,
      token = await getCITokenFromLocalStorage();
    expect(token).not.toHaveLength(0);

    try {
      await test.step(`Create ${dbName} DB cluster`, async () => {
        await createDbClusterFn(
          request,
          {
            dbName: dbName,
            dbType: 'postgresql',
            numberOfNodes: '1',
          },
          namespace
        );
      });

      await test.step(`Wait for DB cluster ${dbName} creation`, async () => {
        await expect(async () => {
          // new DB cluster appears in response not immediately.
          // wait for new DB cluster to appear.
          const dbCluster = await getDbClusterAPI(
            dbName,
            namespace,
            request,
            token
          );
          expect(dbCluster).toBeDefined();
        }).toPass({
          intervals: [1000],
          timeout: TIMEOUTS.TenSeconds,
        });

        await goToUrl(page, '/databases');
        await expect(page.getByText(dbName)).toBeVisible({
          timeout: TIMEOUTS.TenSeconds,
        });
      });

      await test.step(`Deleting ${dbName} DB cluster via UI`, async () => {
        await findDbAndClickRow(page, dbName);

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
        await expect(
          page.getByTestId('irreversible-action-alert')
        ).toBeVisible();
        const deleteConfirmationButton = page
          .getByRole('button')
          .filter({ hasText: 'Delete' });
        await expect(deleteConfirmationButton).toBeDisabled();
        await page.getByTestId('text-input-confirm-input').fill(dbName);
        await expect(deleteConfirmationButton).toBeEnabled();
        await deleteConfirmationButton.click();
      });
    } finally {
      await deleteDbClusterFn(request, dbName, namespace);
    }
  });
});
