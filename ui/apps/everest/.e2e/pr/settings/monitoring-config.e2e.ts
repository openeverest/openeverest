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
import {
  openDbCreationForm,
  populateMonitoringModalForm,
} from '@e2e/utils/db-wizard';
import { setNamespace } from '@e2e/utils/namespaces';
const { MONITORING_URL, MONITORING_USER, MONITORING_PASSWORD } = process.env;

const mockedMonitoringProvider = {
  apiVersion: 'everest.percona.com/v1alpha1',
  kind: 'Provider',
  metadata: {
    name: 'postgresql',
  },
  spec: {
    uiSchema: {
      single: {
        sections: {
          monitoring: {
            label: 'Custom monitoring section',
            components: {
              monitoringConfig: {
                uiType: 'select',
                path: 'spec.components.monitoring.customSpec.monitoringConfigName',
                dataSource: {
                  provider: 'monitoringConfigs',
                },
                fieldParams: {
                  label: 'Monitoring config',
                },
              },
            },
          },
        },
        sectionsOrder: ['monitoring'],
      },
    },
  },
};

const monitoringFallbackTestId = 'monitoring-empty-fallback';

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

  test('Shows DB creation monitoring fallback when no configs exist and allows inline creation', async ({
    page,
    request,
  }) => {
    // This test navigates away from the settings page, runs API cleanup,
    // opens the DB creation form, interacts with a modal and verifies multiple
    // assertions — the default 30 s timeout is not enough.
    test.setTimeout(TIMEOUTS.ThreeMinutes);

    // Keep this suite self-contained: it owns the namespace state needed for the
    // fallback test and should not depend on the global monitoring setup.
    const configs = await listMonitoringConfigs(request, namespace, token);
    for (const config of configs?.items ?? []) {
      const name = config.metadata?.name;
      if (name) {
        try {
          await deleteMonitoringConfig(request, namespace, name, token);
        } catch {
          // Config may already be gone from a previous cleanup pass.
        }
      }
    }

    await expect(async () => {
      const remainingConfigs = await listMonitoringConfigs(
        request,
        namespace,
        token
      );
      expect(remainingConfigs?.items ?? []).toHaveLength(0);
    }).toPass({
      intervals: [1000, 2000, 3000],
      timeout: TIMEOUTS.ThirtySeconds,
    });

    await page.route('**/v1/clusters/main/providers', async (route) => {
      await route.fulfill({
        json: {
          items: [mockedMonitoringProvider],
        },
      });
    });

    await goToUrl(page, '/databases');

    await openDbCreationForm(page);
    await setNamespace(page, namespace);

    const configureMoreButton = page.getByRole('button', {
      name: 'Configure more options',
    });

    if (await configureMoreButton.isVisible().catch(() => false)) {
      await configureMoreButton.click();
    }

    await expect(
      page.getByRole('heading', { name: 'Custom monitoring section' })
    ).toBeVisible({ timeout: TIMEOUTS.ThirtySeconds });

    await test.step('Verify fallback warning is visible', async () => {
      const fallback = page.getByTestId(monitoringFallbackTestId);
      await expect(fallback).toBeVisible({ timeout: TIMEOUTS.ThirtySeconds });

      const addButton = fallback.getByRole('button', {
        name: /add monitoring endpoint/i,
      });
      await expect(addButton).toBeVisible();
    });

    await test.step('Create monitoring config via inline modal', async () => {
      await populateMonitoringModalForm(
        page,
        fallbackConfigName,
        namespace,
        MONITORING_URL!,
        MONITORING_USER!,
        MONITORING_PASSWORD!,
        false
      );
    });

    await test.step('Verify fallback disappears and select shows the new config', async () => {
      await expect(page.getByTestId(monitoringFallbackTestId)).not.toBeVisible({
        timeout: TIMEOUTS.ThirtySeconds,
      });

      const selectInput = page.getByRole('combobox').first();
      await expect(selectInput).toBeVisible({ timeout: TIMEOUTS.TenSeconds });
      await expect(selectInput).toContainText(fallbackConfigName);
    });
  });
});
