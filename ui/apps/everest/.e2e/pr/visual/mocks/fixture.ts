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

// Hermetic visual-regression fixture. The suite needs NO backend: every /v1
// request (data, platform, and auth) is mocked, and the authenticated session
// is granted by mocking POST /v1/auth/token. This makes screenshots fully
// deterministic and lets CI run without a cluster.
//
// - `test`      : authenticated context (fake session + platform + data mocks).
// - `baseTest`  : unauthenticated context (platform mocks + /auth/token -> 401)
//                 for the login-page screenshot.

import { test as base, expect } from '@playwright/test';
import type { Page } from '@playwright/test';
import {
  installVisualMocks,
  installPlatformMocks,
  mockAuthenticatedSessionRoute,
  mockUnauthenticatedSessionRoute,
} from './routes';

// Suppress the first-run welcome dialog (localStorage-gated) so it never leaks
// into a screenshot.
const dismissWelcomeDialog = (page: Page) =>
  page.addInitScript(() => {
    window.localStorage.setItem('welcomeModal', 'false');
  });

// Freeze wall-clock time so any relative/absolute date rendering is stable.
// setFixedTime overrides Date.now()/new Date() for deterministic rendering
// while keeping real timers running.
const freezeClock = (page: Page) =>
  page.clock.setFixedTime(new Date('2026-07-08T00:00:00Z'));

export const test = base.extend({
  // No real login: start from a clean storage state and let the mocked
  // /auth/token bootstrap the session.
  storageState: { cookies: [], origins: [] },
  page: async ({ page }, use) => {
    await dismissWelcomeDialog(page);
    await freezeClock(page);
    await mockAuthenticatedSessionRoute(page);
    await installPlatformMocks(page);
    await installVisualMocks(page);
    await use(page);
  },
});

// Unauthenticated fixture for the login-page screenshot.
export const baseTest = base.extend({
  storageState: { cookies: [], origins: [] },
  page: async ({ page }, use) => {
    await dismissWelcomeDialog(page);
    await freezeClock(page);
    await mockUnauthenticatedSessionRoute(page);
    await installPlatformMocks(page);
    await use(page);
  },
});

export { expect };
