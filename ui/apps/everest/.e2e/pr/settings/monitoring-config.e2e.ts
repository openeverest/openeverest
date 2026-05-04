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
  listMonitoringConfigs,
} from '@e2e/utils/monitoring-config';
import { goToStep } from '@e2e/utils/db-wizard';
const { MONITORING_URL, MONITORING_USER, MONITORING_PASSWORD } = process.env;

test.describe.serial('Monitoring Configs', () => {
  const monitoringConfigName = limitedSuffixedName('pr-set-mon'),
    fallbackConfigName = limitedSuffixedName('pr-mon-fb'),
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
    for (const name of [monitoringConfigName, fallbackConfigName]) {
      await expect(async () => {
        await deleteMonitoringConfig(request, namespace, name, token);
      }).toPass({
        intervals: [1000],
        timeout: TIMEOUTS.TenSeconds,
      });
    }
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

  test('Shows wizard monitoring fallback when no configs exist and allows inline creation', async ({
    page,
    request,
  }) => {
    // Keep this suite self-contained: it owns the namespace state needed for the
    // fallback test and should not depend on the global monitoring setup.
    const configs = await listMonitoringConfigs(request, namespace, token);
    for (const config of configs?.items ?? []) {
      const name = config.metadata?.name;
      if (name) {
        await deleteMonitoringConfig(request, namespace, name, token);
      }
    }

    await goToUrl(page, '/databases');

    // Open the DB creation wizard for PostgreSQL
    await page.getByTestId('add-db-cluster-button').click();
    await page
      .getByTestId('add-db-cluster-button-menu')
      .getByRole('menuitem')
      .first()
      .waitFor();
    await page.getByTestId('add-db-cluster-button-postgresql').click();
    await page.waitForURL('/databases/new');

    // Navigate to the Monitoring step
    await goToStep(page, 'monitoring');

    await test.step('Verify fallback warning is visible', async () => {
      const fallback = page.getByTestId('monitoring-empty-fallback');
      await expect(fallback).toBeVisible({ timeout: TIMEOUTS.ThirtySeconds });

      await expect(fallback).toContainText('monitoring');
      await expect(fallback).toContainText(namespace);

      const addButton = fallback.getByRole('button', {
        name: /add monitoring endpoint/i,
      });
      await expect(addButton).toBeVisible();
    });

    await test.step('Create monitoring config via inline modal', async () => {
      const fallback = page.getByTestId('monitoring-empty-fallback');
      const addButton = fallback.getByRole('button', {
        name: /add monitoring endpoint/i,
      });
      await addButton.click();

      await page.getByTestId('text-input-name').fill(fallbackConfigName);

      const namespaceInput = page.getByTestId('text-input-namespace');
      if (await namespaceInput.isVisible()) {
        await namespaceInput.click();
        await page.getByRole('option', { name: namespace }).click();
      }

      await page.getByTestId('text-input-url').fill(MONITORING_URL!);
      await page.getByTestId('text-input-user').fill(MONITORING_USER!);
      await page.getByTestId('text-input-password').fill(MONITORING_PASSWORD!);

      await expect(page.getByTestId('form-dialog-add')).toBeEnabled();
      await page.getByTestId('form-dialog-add').click();
    });

    await test.step('Verify fallback disappears and select shows the new config', async () => {
      await expect(
        page.getByTestId('monitoring-empty-fallback')
      ).not.toBeVisible({ timeout: TIMEOUTS.ThirtySeconds });

      const selectInput = page.locator(
        '[data-testid*="select-input"][data-testid*="monitoring"]'
      );
      await expect(selectInput).toBeVisible({ timeout: TIMEOUTS.TenSeconds });
      await expect(selectInput).toHaveValue(fallbackConfigName);
    });

    await test.step('Verify the preview sidebar shows monitoring config name', async () => {
      const monitoringSection = page.getByTestId('section-monitoring');
      await expect(monitoringSection).toBeVisible();
      const previewContent = monitoringSection.getByTestId('preview-content');
      await expect(previewContent).toContainText(fallbackConfigName);
    });
  });
});
