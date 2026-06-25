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

import { test, expect } from '@playwright/test';

const USER = process.env.SESSION_USER!;

test('Rate limiting on login (429)', async ({ page }) => {
  // This test poisons the rate limiter for this IP — must run after all other
  // session tests to avoid flakiness.
  await page.goto('/login');
  await page.waitForLoadState('networkidle');

  const attempts = 5;
  for (let i = 0; i < attempts; i++) {
    await page.getByTestId('text-input-username').fill(USER);
    await page.getByTestId('text-input-password').fill('wrong-password');
    await page.getByTestId('login-button').click();

    const rateLimitMsg = page.getByText(/too many attempts/i);
    if (await rateLimitMsg.isVisible({ timeout: 1000 }).catch(() => false)) {
      await expect(rateLimitMsg).toBeVisible();
      return; // Test passed
    }
  }

  expect(
    false,
    'Server did not return 429 after ' + attempts + ' failed logins'
  ).toBeTruthy();
});
