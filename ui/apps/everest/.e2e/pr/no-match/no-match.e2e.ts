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
import { checkNoMatchPage } from '@e2e/utils/no-match';
import { TIMEOUTS } from '@e2e/constants';

test.describe.parallel('No match (404) page', () => {
  test('databases page successfully loaded', async ({ page }) => {
    // under the load this page may take some time to render.
    const getResp = page.waitForResponse(
      (resp) =>
        resp.request().method() === 'GET' &&
        resp.url().includes('/databases') &&
        resp.status() === 200
    );

    await page.goto('/databases');
    await getResp;

    const button = page.getByTestId('add-db-cluster-button');
    await expect(button).toBeVisible({ timeout: TIMEOUTS.ThirtySeconds });
  });

  test('non existing url should render no match page', async ({ page }) => {
    await page.goto('/wrong');
    await checkNoMatchPage(page);
  });

  test('non existing url with slash should render no match page', async ({
    page,
  }) => {
    await page.goto('/wrong/wrong');
    await checkNoMatchPage(page);
  });

  test('non existing url with double slash should render no match page', async ({
    page,
  }) => {
    await page.goto('/wrong/wrong/wrong');
    await checkNoMatchPage(page);
  });

  test('existing url with non existing after slash should render no match page', async ({
    page,
  }) => {
    await page.goto('/database/wrong');
    await checkNoMatchPage(page);
  });

  test('overview tab with non existing dbName should render no match page', async ({
    page,
  }) => {
    await page.goto('/databases/wrong/overview');
    await checkNoMatchPage(page);
  });

  test('backups tab with non existing dbName should render no match page', async ({
    page,
  }) => {
    await page.goto('/databases/wrong/backups');
    await checkNoMatchPage(page);
  });
});
