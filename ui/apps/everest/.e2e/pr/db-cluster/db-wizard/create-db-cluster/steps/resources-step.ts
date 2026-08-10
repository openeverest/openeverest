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

import { Page, expect } from '@playwright/test';

export const resourcesStepCheck = async (page: Page) => {
  await expect(page.getByTestId('step-header')).toBeVisible();
  await expect(page.getByTestId('step-description')).toBeVisible();

  await page.getByTestId('node-resources-toggle-button-large').click();
  await page.getByTestId('text-input-cpu').fill('0.6');
  await page.getByTestId('text-input-memory').fill('1');
  await page.getByTestId('text-input-disk').fill('1');

  expect(await page.getByText('x 3 nodes').count()).toBe(3);
  await expect(page.getByTestId('cpu-resource-sum')).toHaveText('= 1.80 CPU');
  await expect(page.getByTestId('memory-resource-sum')).toHaveText('= 3.00 GB');
  await expect(page.getByTestId('disk-resource-sum')).toHaveText(' = 3.00 Gi');

  // When requests are unsynced from limits, leaving a request field empty must
  // block submission instead of silently falling back to the limit value.
  const syncSwitch = page
    .getByTestId('switch-input-node-requests-synced-label')
    .getByRole('checkbox');
  await syncSwitch.uncheck();

  const cpuRequest = page.getByTestId('text-input-cpu-requests');
  await expect(cpuRequest).toBeVisible();

  await cpuRequest.fill('');
  await expect(page.getByText('This field is required')).toBeVisible();

  // A valid request below the limit clears the error.
  await cpuRequest.fill('0.6');
  await expect(page.getByText('This field is required')).not.toBeVisible();

  // Restore the synced state so the rest of the flow uses the defaults.
  await syncSwitch.check();
};
