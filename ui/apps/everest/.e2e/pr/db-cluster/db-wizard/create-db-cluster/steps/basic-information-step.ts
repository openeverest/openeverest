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

export const DEFAULT_CLUSTER_VERSION = '8.0.19-7';

export const basicInformationStepCheck = async (
  page: Page,
  engineVersions,
  recommendedEngineVersions,
  clusterName
) => {
  expect(
    await page.getByTestId('switch-input-sharding').getByRole('checkbox')
  ).not.toBeVisible();
  await page.getByTestId('select-db-version-button').click();

  const dbVersionOptions = page.getByRole('option');

  for (const version of engineVersions.psmdb) {
    await expect(
      dbVersionOptions.filter({ hasText: new RegExp(`^${version}$`) })
    ).toBeVisible();
  }

  const defaultOption = await page.getByRole('option', { selected: true });
  expect(await defaultOption.textContent()).toBe(
    recommendedEngineVersions.psmdb
  );

  await page
    .getByRole('option')
    .filter({ hasText: DEFAULT_CLUSTER_VERSION })
    .click();
  await page.getByTestId('text-input-db-name').fill(clusterName);
  expect(
    await page.getByTestId('switch-input-sharding').getByRole('checkbox')
  ).not.toBeDisabled();
};
