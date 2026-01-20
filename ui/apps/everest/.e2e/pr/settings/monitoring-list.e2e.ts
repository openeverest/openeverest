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
import { EVEREST_CI_NAMESPACES } from '@e2e/constants';
const { MONITORING_URL, MONITORING_USER, MONITORING_PASSWORD } = process.env;

test.describe.serial('Monitoring List', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/settings/monitoring-endpoints');
    await page.getByTestId('add-monitoring-endpoint').waitFor();
  });
  const monitoringEndpointName = 'monitoring-test';

  test('Create Monitoring Endpoint', async ({ page }) => {
    await page.getByTestId('add-monitoring-endpoint').click();

    // filling out the form
    await page.getByTestId('text-input-name').fill(monitoringEndpointName);
    const namespaces = page.getByTestId('text-input-namespace');
    await namespaces.click();
    await expect(
      page
        .getByRole('option')
        .filter({ hasText: EVEREST_CI_NAMESPACES.EVEREST_UI })
    ).toBeVisible();
    await page.getByRole('option').last().click();
    await page.getByTestId('text-input-url').fill(MONITORING_URL);
    await page.getByTestId('text-input-user').fill(MONITORING_USER);
    await page.getByTestId('text-input-password').fill(MONITORING_PASSWORD);

    await page.getByTestId('form-dialog-add').click();

    await expect(
      page
        .locator('.MuiTableRow-root')
        .filter({ hasText: monitoringEndpointName })
    ).toBeVisible();
  });

  test('Edit Monitoring Endpoint', async ({ page }) => {
    await page
      .locator('.MuiTableRow-root')
      .filter({ hasText: monitoringEndpointName })
      .getByTestId('MoreHorizIcon')
      .click();

    await page.getByRole('menuitem', { name: 'Edit' }).click();

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
        'OpenEverest does not store PMM credentials, so fill in both the User and Password fields.'
      )
    ).toBeVisible();
    await page.getByTestId('text-input-password').fill(MONITORING_PASSWORD);
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
    await page.getByTestId('text-input-user').fill(MONITORING_USER);
    await expect(page.getByTestId('form-dialog-edit')).toBeEnabled();

    await page.getByTestId('form-dialog-edit').click();
  });

  test('Delete Monitoring Endpoint', async ({ page }) => {
    await findRowAndClickActions(page, monitoringEndpointName, 'Delete');
    await page.getByTestId('confirm-dialog-delete').click();
    await waitForDelete(page, monitoringEndpointName, 10000);
  });
});
