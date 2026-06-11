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
// See the License for the specific language governing permissions ands
// limitations under the License.

import { test, expect } from '@playwright/test';
import { loginSessionUser, logoutSessionUser } from '@e2e/utils/user';
import { execSync } from 'child_process';
import { getCliPath } from '@e2e/utils/session-cli';
import { SESSION_USER_STORAGE_STATE_FILE, TIMEOUTS } from '@e2e/constants';

const USER = process.env.SESSION_USER!;

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
  const res = await request.get('/v1/version', {
    headers: { Authorization: `Bearer ${token}` },
  });
  expect(res.status()).toBe(401);
  const body = await res.json();
  expect(body.message).toMatch(/invalid token|invalid or expired jwt/i);
}
// ——————————————————————————————————————————————————
test.slow();

test.describe.serial('Session', () => {
  let token: string;

  test.beforeEach(async ({ page }) => {
    await test.step('Login as session user', async () => {
      // The access token is kept in memory by the UI (not in localStorage),
      // so it is captured from the token endpoint response during login.
      const tokenResponse = page.waitForResponse(
        (resp) => resp.url().includes('/v1/auth/token') && resp.ok()
      );
      await loginSessionUser(page);
      token = (await (await tokenResponse).json()).access_token;
      expect(token).toBeTruthy();
    });
  });

  test.afterEach(async ({ page }) => {
    await page.evaluate(() => localStorage.clear());
    await page.evaluate(() => sessionStorage.clear());
    await page.context().clearCookies();
    await page
      .context()
      .storageState({ path: SESSION_USER_STORAGE_STATE_FILE });
  });

  test('T191 - Verify token invalidity after user logout from Everest UI - Everest user', async ({
    page,
    request,
  }) => {
    await test.step('Token works before logout', async () =>
      expectAuthenticated(request, token));

    await test.step('Perform UI logout', async () => logoutSessionUser(page));

    await test.step('Verify token is invalid after logout', async () =>
      expectUnauthenticated(request, token));
  });

  test('T190 - Verify user is logged out and token is invalidated after updating user password', async ({
    page,
    request,
  }) => {
    await test.step('Token works before password update', async () =>
      expectAuthenticated(request, token));

    await test.step('Update user password', async () => {
      // Update user password via CLI
      execSync(`${cliPath} accounts set-password -u ${USER} -p newPass12345`, {
        stdio: 'inherit',
      });
    });

    await test.step('Verify UI logout', async () => {
      // After password update, UI should auto-logout
      await page.waitForURL('/login', { timeout: TIMEOUTS.ThirtySeconds });
    });

    await test.step('Verify token is invalid after logout', async () =>
      expectUnauthenticated(request, token));

    await test.step('Restore user password', async () => {
      // Update user password via CLI
      execSync(
        `${cliPath} accounts set-password -u ${USER} -p ${process.env.SESSION_PASS}`,
        {
          stdio: 'inherit',
        }
      );
    });
  });

  test('T189 - Verify user is logged out and token is invalidated after user deletion', async ({
    page,
    request,
  }) => {
    await test.step('Token works before user deletion', async () =>
      expectAuthenticated(request, token));

    await test.step('Delete user', async () => {
      // Delete user via CLI
      execSync(`${cliPath} accounts delete -u ${USER}`, {
        stdio: 'inherit',
      });
    });

    await test.step('Verify UI logout', async () => {
      // After delete, UI should auto-logout
      await page.waitForURL('/login', { timeout: TIMEOUTS.ThirtySeconds });
    });

    await test.step('Verify token is invalid after logout', async () =>
      expectUnauthenticated(request, token));
  });
});
