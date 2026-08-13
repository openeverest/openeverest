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
    page
      .getByRole('heading')
      .filter({ hasText: /Create (on-demand )?backup/i })
  ).toBeVisible();
};
