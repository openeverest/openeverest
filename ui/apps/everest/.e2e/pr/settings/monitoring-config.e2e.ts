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
import {
  EVEREST_CI_CLUSTER,
  EVEREST_CI_NAMESPACES,
  TIMEOUTS,
} from '@e2e/constants';
import { goToUrl, limitedSuffixedName } from '@e2e/utils/generic';
import { getCITokenFromLocalStorage } from '@e2e/utils/localStorage';
import {
  deleteMonitoringConfig,
  getMonitoringConfig,
} from '@e2e/utils/monitoring-config';
const { MONITORING_URL, MONITORING_USER, MONITORING_PASSWORD } = process.env;

test.describe.serial('Monitoring Configs', () => {
  const monitoringConfigName = limitedSuffixedName('pr-set-mon'),
    namespace = EVEREST_CI_NAMESPACES.EVEREST_UI;
  let token: string;

  test.beforeAll(async ({}) => {
    const t = await getCITokenFromLocalStorage();
    expect(t).toBeDefined();
    expect(t).not.toHaveLength(0);
    token = t!;
  });

  test.beforeEach(async ({ page }) => {
    await goToUrl(page, '/settings/monitoring-endpoints');
  });

  test.afterAll(async ({ request }) => {
    await expect(async () => {
      await deleteMonitoringConfig(
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

  test('Create Monitoring Endpoint', async ({ page, request }) => {
    await test.step(`Create Monitoring Endpoint`, async () => {
      await page.getByTestId('add-monitoring-endpoint').click();
      await page.waitForLoadState('load', { timeout: TIMEOUTS.ThirtySeconds });

      // filling out the form
      await page.getByTestId('text-input-name').fill(monitoringConfigName);
      await page.getByTestId('text-input-namespace').click();
      await page.getByRole('option', { name: namespace }).click();
      await page.getByTestId('text-input-url').fill(MONITORING_URL!);
      await page.getByTestId('text-input-user').fill(MONITORING_USER!);
      await page.getByTestId('text-input-password').fill(MONITORING_PASSWORD!);
      await page.getByTestId('form-dialog-add').click();

      await page.waitForURL('/settings/monitoring-endpoints', {
        timeout: TIMEOUTS.ThirtySeconds,
      });
    });

    await test.step(`Check created Monitoring Endpoint`, async () => {
      await expect(async () => {
        const monitoringConfig = await getMonitoringConfig(
          request,
          namespace,
          monitoringConfigName,
          token
        );
        expect(monitoringConfig).toBeDefined();
        expect(monitoringConfig.metadata?.name).toBe(monitoringConfigName);
      }).toPass({
        intervals: [1000, 2000, 3000],
        timeout: TIMEOUTS.ThirtySeconds,
      });
    });
  });

  test('List Monitoring Endpoint', async ({ page }) => {
    const row = page
      .locator('.MuiTableRow-root')
      .filter({ hasText: monitoringConfigName });
    await expect(row).toBeVisible();
    await expect(row.getByText(MONITORING_URL!)).toBeVisible();
    await expect(row.getByText(namespace)).toBeVisible();
  });

  test('Edit Monitoring Endpoint', async ({ page }) => {
    await findRowAndClickActions(page, monitoringConfigName, 'Edit');

    await expect(page.getByTestId('text-input-name')).toBeDisabled();
    await expect(page.getByTestId('text-input-namespace')).toBeDisabled();
    await page.getByTestId('text-input-url').fill(MONITORING_URL!);

    // user can leave the credentials empty
    await expect(page.getByTestId('form-dialog-edit')).toBeEnabled();

    // user should fill both of credentials
    await page.getByTestId('text-input-user').fill(MONITORING_USER!);
    await expect(page.getByTestId('form-dialog-edit')).toBeDisabled();
    await expect(
      page.getByText(
        'OpenEverest does not store PMM credentials, so fill in both the User and Password fields.'
      )
    ).toBeVisible();
    await page.getByTestId('text-input-password').fill(MONITORING_PASSWORD!);
    await expect(page.getByTestId('form-dialog-edit')).toBeEnabled();
    await expect(
      page.getByText(
        'OpenEverest does not store PMM credentials, so fill in both the User and Password fields.'
      )
    ).not.toBeVisible();
    await page.getByTestId('text-input-user').fill('');
    await expect(page.getByTestId('form-dialog-edit')).toBeDisabled();
    await expect(
      page.getByText(
        'OpenEverest does not store PMM credentials, so fill in both the User and Password fields.'
      )
    ).toBeVisible();
    await page.getByTestId('text-input-user').fill(MONITORING_USER!);
    await expect(page.getByTestId('form-dialog-edit')).toBeEnabled();

    await page.getByTestId('form-dialog-edit').click();
  });

  test('Delete Monitoring Endpoint', async ({ page }) => {
    await findRowAndClickActions(page, monitoringConfigName, 'Delete');

    const delResponse = page.waitForResponse(
      (resp) =>
        resp.request().method() === 'DELETE' &&
        resp
          .url()
          .includes(
            `/v1/clusters/${EVEREST_CI_CLUSTER}/namespaces/${namespace}/monitoring-configs/${monitoringConfigName}`
          ) &&
        resp.status() === 204
    );
    await page.getByTestId('confirm-dialog-delete').click();
    await delResponse;

    await waitForDelete(page, monitoringConfigName, TIMEOUTS.TenSeconds);
  });
});
