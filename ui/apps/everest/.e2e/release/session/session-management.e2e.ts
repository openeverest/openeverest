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

import { test, expect } from '@playwright/test';
import { loginSessionUser, dismissOnboarding } from '@e2e/utils/user';
import { execSync } from 'child_process';
import { getCliPath } from '@e2e/utils/session-cli';
import { TIMEOUTS } from '@e2e/constants';

const USER = process.env.SESSION_USER!;
const PASS = process.env.SESSION_PASS!;

const cliPath = getCliPath();

// ——————————————————————————————————————————————————
// check API response for valid and invalid tokens
async function expectAuthenticated(request, token: string) {
  const res = await request.get('/v1/version', {
    headers: { Authorization: `Bearer ${token}` },
  });
  expect(res.status()).toBe(200);
  const body = await res.json();
  expect(body).toHaveProperty('version');
}

async function expectUnauthenticated(request, token: string) {
  // Blocklist is K8s watch-cached — may take 1-3s to propagate.
  await expect
    .poll(
      async () => {
        const res = await request.get('/v1/version', {
          headers: { Authorization: `Bearer ${token}` },
        });
        return res.status();
      },
      { timeout: 10_000, intervals: [500] }
    )
    .toBe(401);
}
// ——————————————————————————————————————————————————

test.slow();

test.describe.serial('Session management', () => {
  test('Page reload preserves session via cookie refresh', async ({ page }) => {
    await page.goto('/login');
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(6000);
    await page.getByTestId('text-input-username').fill(USER);
    await page.getByTestId('text-input-password').fill(PASS);
    await page.getByTestId('login-button').click();
    await expect(page.getByTestId('user-appbar-button')).toBeVisible({
      timeout: TIMEOUTS.ThirtySeconds,
    });
    await expect(page).not.toHaveURL('/login');

    // Reload — in-memory access token is lost, but HttpOnly refresh cookie
    // allows silent re-authentication.
    await page.reload();

    await expect(page.getByTestId('user-appbar-button')).toBeVisible({
      timeout: TIMEOUTS.ThirtySeconds,
    });
    await expect(page).not.toHaveURL('/login');
  });

  test('Cross-tab logout synchronization', async ({ context }) => {
    test.setTimeout(60000);
    await context.clearCookies();
    const page1 = await context.newPage();

    // Login in tab 1
    await page1.goto('/login');
    await page1.waitForLoadState('networkidle');
    await page1.waitForTimeout(6000);
    await page1.getByTestId('text-input-username').fill(USER);
    await page1.getByTestId('text-input-password').fill(PASS);
    await page1.getByTestId('login-button').click();
    await expect(page1.getByTestId('user-appbar-button')).toBeVisible({
      timeout: TIMEOUTS.ThirtySeconds,
    });

    await dismissOnboarding(page1);
    await page1.waitForLoadState('networkidle');

    // Tab 2: navigate to app — shared HttpOnly cookie lets bootstrapSession()
    // authenticate automatically (realistic scenario: user opens a second tab).
    // Wait for rate limiter token bucket to refill (1 req/s on /auth/token).
    await page1.waitForTimeout(3000);

    const page2 = await context.newPage();
    await page2.goto('/');
    await expect(page2.getByTestId('user-appbar-button')).toBeVisible({
      timeout: TIMEOUTS.ThirtySeconds,
    });

    await dismissOnboarding(page2);

    // Verify tab 1 is still logged in
    await expect(page1.getByTestId('user-appbar-button')).toBeVisible({
      timeout: TIMEOUTS.TenSeconds,
    });

    // Logout from tab 1 — tab 2 should be redirected via BroadcastChannel
    await page1.getByTestId('user-appbar-button').click();
    await page1.getByRole('menuitem').filter({ hasText: 'Log out' }).click();
    await page1.waitForURL('/login', { timeout: TIMEOUTS.ThirtySeconds });
    await page2.waitForURL('/login', { timeout: TIMEOUTS.ThirtySeconds });

    await page1.close();
    await page2.close();
  });

  test('Invalid credentials show error message', async ({ page }) => {
    await page.goto('/login');
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(6000);
    await page.getByTestId('text-input-username').fill(USER);
    await page.getByTestId('text-input-password').fill('definitely-wrong');
    await page.getByTestId('login-button').click();

    // Should show "Invalid credentials" error notification
    await expect(page.getByText('Invalid credentials')).toBeVisible({
      timeout: TIMEOUTS.TenSeconds,
    });

    // Should remain on login page
    await expect(page).toHaveURL(/\/login/);
  });

  test('Access token is invalidated after UI logout', async ({
    page,
    request,
  }) => {
    // Login and capture access token
    const tokens: string[] = [];
    page.on('response', async (resp) => {
      if (resp.url().includes('/v1/auth/token') && resp.ok()) {
        try {
          const body = await resp.json();
          if (body.access_token) tokens.push(body.access_token);
        } catch {}
      }
    });
    await loginSessionUser(page);
    await page.waitForLoadState('networkidle');
    const token = tokens[tokens.length - 1];
    expect(token).toBeTruthy();

    await test.step('Token works before logout', async () =>
      expectAuthenticated(request, token));

    await test.step('Perform UI logout', async () => {
      await expect(page.getByTestId('user-appbar-button')).toBeVisible({
        timeout: TIMEOUTS.ThirtySeconds,
      });
      await page.getByTestId('user-appbar-button').click();
      await page.getByRole('menuitem').filter({ hasText: 'Log out' }).click();
      await page.waitForURL('/login', { timeout: TIMEOUTS.ThirtySeconds });
    });

    await test.step('Verify token is invalid after logout', async () =>
      expectUnauthenticated(request, token));
  });

  test('Password change invalidates session and forces re-login', async ({
    page,
    request,
  }) => {
    // Login and capture access token
    const tokens: string[] = [];
    page.on('response', async (resp) => {
      if (resp.url().includes('/v1/auth/token') && resp.ok()) {
        try {
          const body = await resp.json();
          if (body.access_token) tokens.push(body.access_token);
        } catch {}
      }
    });
    await loginSessionUser(page);
    await page.waitForLoadState('networkidle');
    const token = tokens[tokens.length - 1];
    expect(token).toBeTruthy();

    await test.step('Token works before password update', async () =>
      expectAuthenticated(request, token));

    await test.step('Update user password', async () => {
      // Update user password via CLI
      execSync(`${cliPath} accounts set-password -u ${USER} -p newPass12345`, {
        stdio: 'inherit',
      });
    });

    await test.step('Verify UI logout', async () => {
      // Wait for blocklist cache sync, then reload to trigger failed refresh.
      await page.waitForTimeout(3000);
      await page.reload();
      await page.waitForURL('/login', { timeout: TIMEOUTS.ThirtySeconds });
    });

    await test.step('Verify token is invalid after logout', async () =>
      expectUnauthenticated(request, token));

    await test.step('Restore user password', async () => {
      execSync(`${cliPath} accounts set-password -u ${USER} -p ${PASS}`, {
        stdio: 'inherit',
      });
    });
  });

  // ———————————— Destructive tests (last!) ————————————
  test('User deletion invalidates session and forces re-login', async ({
    page,
    request,
  }) => {
    // Login and capture access token
    const tokens: string[] = [];
    page.on('response', async (resp) => {
      if (resp.url().includes('/v1/auth/token') && resp.ok()) {
        try {
          const body = await resp.json();
          if (body.access_token) tokens.push(body.access_token);
        } catch {}
      }
    });
    await loginSessionUser(page);
    await page.waitForLoadState('networkidle');
    const token = tokens[tokens.length - 1];
    expect(token).toBeTruthy();

    await test.step('Token works before user deletion', async () =>
      expectAuthenticated(request, token));

    await test.step('Delete user', async () => {
      execSync(`${cliPath} accounts delete -u ${USER}`, {
        stdio: 'inherit',
      });
    });

    await test.step('Verify UI logout', async () => {
      // Wait for blocklist cache sync, then reload to trigger failed refresh.
      await page.waitForTimeout(3000);
      await page.reload();
      await page.waitForURL('/login', { timeout: TIMEOUTS.ThirtySeconds });
    });

    await test.step('Verify token is invalid after logout', async () =>
      expectUnauthenticated(request, token));
  });
});
