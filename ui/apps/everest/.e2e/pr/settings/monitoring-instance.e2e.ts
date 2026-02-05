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

import { expect, test } from '@playwright/test';
import { findRowAndClickActions, waitForDelete } from '@e2e/utils/table';
import { EVEREST_CI_NAMESPACES, TIMEOUTS } from '@e2e/constants';
import { goToUrl, limitedSuffixedName } from '@e2e/utils/generic';
import { getCITokenFromLocalStorage } from '@e2e/utils/localStorage';
import {
  deleteMonitoringInstance,
  getMonitoringInstance,
} from '@e2e/utils/monitoring-instance';
const { MONITORING_URL, MONITORING_USER, MONITORING_PASSWORD } = process.env;

test.describe.serial('Monitoring Instances', () => {
  const monitoringConfigName = limitedSuffixedName('pr-set-mon'),
    namespace = EVEREST_CI_NAMESPACES.EVEREST_UI;
  let token: string;

  test.beforeAll(async ({}) => {
    token = await getCITokenFromLocalStorage();
    expect(token).not.toHaveLength(0);
  });

  test.beforeEach(async ({ page }) => {
    await goToUrl(page, '/settings/monitoring-endpoints');
  });

  test.afterAll(async ({ request }) => {
    await expect(async () => {
      await deleteMonitoringInstance(
        request,
        namespace,
        monitoringConfigName,
        token
      );
    }).toPass({
      intervals: [1000],
      timeout: TIMEOUTS.TenSeconds,
    });
  });

  test('Create Monitoring Instance', async ({ page, request }) => {
    await test.step(`Create Monitoring Instance`, async () => {
      await page.getByTestId('add-monitoring-endpoint').click();
      await page.waitForLoadState('load', { timeout: TIMEOUTS.ThirtySeconds });

      // filling out the form
      await page.getByTestId('text-input-name').fill(monitoringConfigName);
      const namespaces = page.getByTestId('text-input-namespace');
      await namespaces.click();
      const nsOption = page.getByRole('option').filter({ hasText: namespace });
      await expect(nsOption).toBeVisible();
      await nsOption.click();
      await page.getByTestId('text-input-url').fill(MONITORING_URL);
      await page.getByTestId('text-input-user').fill(MONITORING_USER);
      await page.getByTestId('text-input-password').fill(MONITORING_PASSWORD);
      await page.getByTestId('form-dialog-add').click();
    });

    await test.step(`Check created Monitoring Instance`, async () => {
      await expect(async () => {
        const dbCluster = await getMonitoringInstance(
          request,
          namespace,
          monitoringConfigName,
          token
        );
        expect(dbCluster).toBeDefined();
      }).toPass({
        intervals: [1000],
        timeout: TIMEOUTS.TenSeconds,
      });
    });
  });

  test('List Monitoring Instance', async ({ page }) => {
    const row = page
      .locator('.MuiTableRow-root')
      .filter({ hasText: monitoringConfigName });
    await expect(row).toBeVisible();
    await expect(row.getByText(MONITORING_URL)).toBeVisible();
    await expect(row.getByText(namespace)).toBeVisible();
  });

  test('Edit Monitoring Instance', async ({ page }) => {
    await findRowAndClickActions(page, monitoringConfigName, 'Edit');

    await expect(page.getByTestId('text-input-name')).toBeDisabled();
    await expect(page.getByTestId('text-input-namespace')).toBeDisabled();
    await page.getByTestId('text-input-url').fill(MONITORING_URL);

    // user can leave the credentials empty
    await expect(page.getByTestId('form-dialog-edit')).toBeEnabled();

    // user should fill both of credentials
    await page.getByTestId('text-input-user').fill(MONITORING_USER);
    await expect(page.getByTestId('form-dialog-edit')).toBeDisabled();
    await expect(
      page.getByText(
        'Percona Everest does not store PMM credentials, so fill in both the User and Password fields.'
      )
    ).toBeVisible();
    await page.getByTestId('text-input-password').fill(MONITORING_PASSWORD);
    await expect(page.getByTestId('form-dialog-edit')).toBeEnabled();
    await expect(
      page.getByText(
        'Percona Everest does not store PMM credentials, so fill in both the User and Password fields.'
      )
    ).not.toBeVisible();
    await page.getByTestId('text-input-user').fill('');
    await expect(page.getByTestId('form-dialog-edit')).toBeDisabled();
    await expect(
      page.getByText(
        'Percona Everest does not store PMM credentials, so fill in both the User and Password fields.'
      )
    ).toBeVisible();
    await page.getByTestId('text-input-user').fill(MONITORING_USER);
    await expect(page.getByTestId('form-dialog-edit')).toBeEnabled();

    await page.getByTestId('form-dialog-edit').click();
  });

  test('Delete Monitoring Instance', async ({ page }) => {
    await findRowAndClickActions(page, monitoringConfigName, 'Delete');

    const delResponse = page.waitForResponse(
      (resp) =>
        resp.request().method() === 'DELETE' &&
        resp
          .url()
          .includes(
            `/v1/namespaces/${namespace}/monitoring-instances/${monitoringConfigName}`
          ) &&
        resp.status() === 204
    );
    await page.getByTestId('confirm-dialog-delete').click();
    await delResponse;

    await waitForDelete(page, monitoringConfigName, TIMEOUTS.TenSeconds);
  });
});
