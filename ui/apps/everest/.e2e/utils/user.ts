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

import {
  CI_USER_STORAGE_STATE_FILE,
  SESSION_USER_STORAGE_STATE_FILE,
  TIMEOUTS,
} from '@e2e/constants';
import { Page, expect } from '@playwright/test';
const { CI_USER, CI_PASSWORD } = process.env;
const { SESSION_USER, SESSION_PASS } = process.env;

export const switchUser = async (
  page: Page,
  user: string,
  password: string
) => {
  await page.goto('/');
  await page.getByTestId('user-appbar-button').click();
  await page.getByRole('menuitem').filter({ hasText: 'Log out' }).click();
  await expect(
    page.getByRole('button').filter({ hasText: 'Log in' })
  ).toBeVisible();
  await page.getByTestId('text-input-username').fill(user);
  await page.getByTestId('text-input-password').fill(password);
  await page.waitForTimeout(1000);
  await page.getByTestId('login-button').click();
  await expect(page.getByTestId('user-appbar-button')).toBeVisible({
    timeout: TIMEOUTS.ThirtySeconds,
  });
  await page.context().storageState({ path: CI_USER_STORAGE_STATE_FILE });
};

// Login functions
const login = async (
  page: Page,
  user: string,
  password: string,
  storageFile: string
) => {
  await page.goto('/login');
  await page.getByTestId('text-input-username').fill(user);
  await page.getByTestId('text-input-password').fill(password);
  await page.getByTestId('login-button').click();

  await expect(page.getByTestId('user-appbar-button')).toBeVisible({
    timeout: TIMEOUTS.ThirtySeconds,
  });

  try {
    // Disable 'Let's Go' Modal permanently
    await expect(page.getByTestId('lets-go-button')).toBeVisible({
      timeout: 3 * 1000,
    });
    await page.getByTestId('lets-go-button').click();
  } catch {
    // Modal not visible, skip
  }

  // The session is persisted through the HttpOnly refresh token cookie;
  // the access token itself is kept in memory only.
  const cookies = (await page.context().storageState()).cookies;
  expect(
    cookies.find((cookie) => cookie.name === 'everest_refresh_token')
  ).not.toBeUndefined();
  await page.context().storageState({ path: storageFile });
};

export const loginCIUser = async (page: Page) => {
  await login(page, CI_USER, CI_PASSWORD, CI_USER_STORAGE_STATE_FILE);
};

export const loginSessionUser = async (page: Page) => {
  await login(
    page,
    SESSION_USER,
    SESSION_PASS,
    SESSION_USER_STORAGE_STATE_FILE
  );
};

// Logout functions
const logout = async (page: Page, storageFile: string) => {
  await page.goto('/');
  await expect(page.getByTestId('user-appbar-button')).toBeVisible({
    timeout: TIMEOUTS.ThirtySeconds,
  });
  await page.getByTestId('user-appbar-button').click();
  await page.getByRole('menuitem').filter({ hasText: 'Log out' }).click();

  // Wait for Login page again
  await page.waitForURL('/login', { timeout: TIMEOUTS.ThirtySeconds });
  await expect(page.getByTestId('login-button')).toBeVisible({
    timeout: TIMEOUTS.ThirtySeconds,
  });

  // Cleanup storage file
  await page.evaluate(() => localStorage.clear());
  await page.evaluate(() => sessionStorage.clear());
  await page.context().storageState({ path: storageFile });
};

export const logoutCIUser = async (page: Page) => {
  await logout(page, CI_USER_STORAGE_STATE_FILE);
};

export const logoutSessionUser = async (page: Page) => {
  await logout(page, SESSION_USER_STORAGE_STATE_FILE);
};
