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

export const clickCreateSchedule = async (page: Page) => {
  const createBackupButton = await page.getByTestId('menu-button');
  await createBackupButton.click();
  const scheduleMenuItem = await page.getByTestId('schedule-menu-item');
  await scheduleMenuItem.click();
  await expect(
    page.getByRole('heading').filter({ hasText: 'Create backup schedule' })
  ).toBeVisible();
};

export const clickOnDemandBackup = async (page: Page) => {
  const createBackupButton = page.getByTestId('menu-button');
  await createBackupButton.click();
  const onDemandMenuItem = page.getByTestId('now-menu-item');
  await onDemandMenuItem.click();
  await expect(
    page.getByRole('heading').filter({ hasText: /Create (on-demand )?backup/i })
  ).toBeVisible();
};
