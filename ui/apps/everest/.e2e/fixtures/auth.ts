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

import { test as base, expect } from '@playwright/test';
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';
import { TIMEOUTS } from '@e2e/constants';
import { dismissOnboarding } from '@e2e/utils/user';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const AUTH_DIR = path.join(__dirname, '..', '.auth');

const { CI_USER, CI_PASSWORD } = process.env;

/**
 * Rate-limiter stagger delay per worker (ms).
 * The auth endpoint has a configurable rate limit (default 1 req/s, burst=1).
 * Staggering worker logins avoids 429 responses.
 */
const WORKER_STAGGER_MS = 3000;

/** Maximum number of login attempts before giving up. */
const MAX_LOGIN_ATTEMPTS = 3;
const LOGIN_RETRY_DELAY_MS = 3000;

/** Options configurable per-project via `use: { ... }` in project config. */
// eslint-disable-next-line @typescript-eslint/no-empty-object-type
type AuthOptions = {};

type AuthWorkerFixtures = {
  authUser: string;
  authPassword: string;
  workerStorageState: string;
};

// ----- Fixture implementation -----

/**
 * Extended test with per-worker authentication.
 *
 * Each Playwright worker logs in independently through the UI and receives its
 * own refresh token + storage state file. This eliminates refresh-token
 * rotation conflicts when workers run in parallel.
 *
 * Usage in tests:
 *   import { test, expect } from '@e2e/fixtures/auth';
 *
 * For RBAC tests with a different user, override in project config:
 *   use: { authUser: process.env.RBAC_USER, authPassword: process.env.RBAC_PASSWORD }
 */
export const test = base.extend<AuthOptions, AuthWorkerFixtures>({
  // Worker-scoped options with defaults — can be overridden per-project via `use: { ... }`.
  authUser: [CI_USER ?? '', { scope: 'worker', option: true }],
  authPassword: [CI_PASSWORD ?? '', { scope: 'worker', option: true }],

  // Override the built-in storageState to use the per-worker file.
  storageState: ({ workerStorageState }, use) => use(workerStorageState),

  // Worker-scoped: login once per worker, reuse across all tests in that worker.
  workerStorageState: [
    async ({ browser, authUser, authPassword }, use, workerInfo) => {
      const id = workerInfo.workerIndex;
      const stateFile = path.join(AUTH_DIR, `ci_worker_${id}.json`);

      // Ensure .auth directory exists.
      fs.mkdirSync(AUTH_DIR, { recursive: true });

      // Stagger logins to respect the per-IP rate limiter.
      if (id > 0) {
        await new Promise((r) => setTimeout(r, id * WORKER_STAGGER_MS));
      }

      const baseURL = process.env.EVEREST_URL || 'http://localhost:8080';
      const context = await browser.newContext({ baseURL });
      const page = await context.newPage();

      for (let attempt = 1; attempt <= MAX_LOGIN_ATTEMPTS; attempt++) {
        await page.goto('/login');
        await page.waitForLoadState('networkidle');

        await page.getByTestId('text-input-username').fill(authUser);
        await page.getByTestId('text-input-password').fill(authPassword);
        await page.getByTestId('login-button').click();

        // Check if we were rate-limited (429 → toast with "too many attempts").
        const rateLimited = page.getByText(/too many attempts/i);
        const appBar = page.getByTestId('user-appbar-button');

        const result = await Promise.race([
          appBar
            .waitFor({ state: 'visible', timeout: TIMEOUTS.ThirtySeconds })
            .then(() => 'ok' as const),
          rateLimited
            .waitFor({ state: 'visible', timeout: TIMEOUTS.TenSeconds })
            .then(() => 'rate-limited' as const)
            .catch(() => null), // Not rate-limited, keep waiting for appBar
        ]);

        if (result === 'ok') {
          break;
        }

        if (result === 'rate-limited' && attempt < MAX_LOGIN_ATTEMPTS) {
          // Wait for the rate limiter to recover before retrying.
          await page.waitForTimeout(LOGIN_RETRY_DELAY_MS * attempt);
          continue;
        }

        // Last attempt — let the original expect assertion produce a clear error.
        await expect(appBar).toBeVisible({
          timeout: TIMEOUTS.ThirtySeconds,
        });
      }

      await dismissOnboarding(page);

      // Verify that the refresh token cookie was set.
      const cookies = (await context.storageState()).cookies;
      expect(
        cookies.find((c) => c.name === 'everest_refresh_token')
      ).not.toBeUndefined();

      await context.storageState({ path: stateFile });
      await context.close();

      await use(stateFile);
    },
    { scope: 'worker', timeout: TIMEOUTS.ThreeMinutes },
  ],
});

export { expect } from '@playwright/test';
