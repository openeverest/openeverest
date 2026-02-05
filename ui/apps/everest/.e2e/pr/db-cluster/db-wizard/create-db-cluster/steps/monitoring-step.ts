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

import { Page, expect } from '@playwright/test';
import { testMonitoringName2 } from '@e2e/utils/monitoring-instance';
import { TIMEOUTS } from '@e2e/constants';

export const monitoringStepCheck = async (page: Page) => {
  await page.waitForLoadState('load', { timeout: TIMEOUTS.ThirtySeconds });

  await expect(
    page.getByTestId('step-header').getByText('Monitoring')
  ).toBeVisible();
  await expect(
    page
      .getByTestId('step-description')
      .getByText(
        'Monitor the health of your database to detect issues quickly and improve its performance.'
      )
  ).toBeVisible();

  const enableMonitoringCheck = page
    .getByTestId('switch-input-monitoring-label')
    .getByRole('checkbox');
  await expect(enableMonitoringCheck).not.toBeChecked();
  await enableMonitoringCheck.check();
  await expect(enableMonitoringCheck).toBeChecked();

  await page.getByTestId('text-input-monitoring-instance').click();
  const monitoringOptions = page.getByRole('option');
  expect(await monitoringOptions.count()).toBeGreaterThanOrEqual(1);
  await monitoringOptions.first().click();

  // for (const option of monitoringInstancesList) {
  //   await expect(
  //     monitoringOptions.filter({hasText: `${option.name} (${option.url})`})
  //   ).toBeVisible();
  // }
  //
  // await monitoringOptions
  //   .filter({ hasText: `${testMonitoringName2} (http://monitoring)` })
  //   .click();

  // -------------- Page control buttons --------------
  await expect(page.getByTestId('db-wizard-previous-button')).toBeVisible();
  await expect(
    page.getByTestId('db-wizard-previous-button')
  ).not.toBeDisabled();

  await expect(page.getByTestId('db-wizard-submit-button')).toBeVisible();
  await expect(page.getByTestId('db-wizard-submit-button')).not.toBeDisabled();

  await expect(page.getByTestId('db-wizard-cancel-button')).toBeVisible();
  await expect(page.getByTestId('db-wizard-cancel-button')).not.toBeDisabled();

  // -------------- DB Summary --------------
  await dbSummaryMonitoringCheck(page);
};

export const dbSummaryMonitoringCheck = async (page: Page) => {
  // -------------- "Database Summary" section (right side) --------------
  // Check for "Monitoring" panel.
  const monitoringInfo = page.getByTestId('section-monitoring');
  await expect(monitoringInfo.getByText('5. Monitoring')).toBeVisible();
  // there may be several 'preview-content' elements in 'Monitoring' section
  const previewContents = monitoringInfo.getByTestId('preview-content');
  expect(await previewContents.textContent()).toBe('Enabled');
};
