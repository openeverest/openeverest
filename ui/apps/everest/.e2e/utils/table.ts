import { Page, expect } from '@playwright/test';
import { TIMEOUTS } from '@e2e/constants';

export const findRowAndClickActions = async (
  page: Page,
  name: string,
  nameOfAction?: string
) => {
  // cluster actions menu click
  await page
    .locator('.MuiTableRow-root')
    .filter({ hasText: name })
    .getByTestId('MoreHorizIcon')
    .click({ timeout: 10000 });

  if (nameOfAction) {
    await page.getByRole('menuitem', { name: nameOfAction }).click();
    await page.waitForLoadState('load', { timeout: TIMEOUTS.ThirtySeconds });
  }
};

// TODO remove after DB Cluster's PATCH method is implemented
export const waitForInitializingState = async (page: Page, name: string) => {
  const dbRow = page.getByRole('row').filter({ hasText: name });
  await expect(dbRow).toBeVisible();
  await expect(dbRow.getByText('Creating')).not.toBeVisible({ timeout: 60000 });
};

/**
 * Waits for specific object status in the table list
 * @param page Page object
 * @param name Name of the object to look for (database, backup)
 * @param status Desired status of the cluster
 * @param timeout How long to wait until error thrown
 */
export const waitForStatus = async (
  page: Page,
  name: string,
  status: string,
  timeout: number
) => {
  const dbRow = page.getByRole('row').filter({ hasText: name });
  await expect(dbRow).toBeVisible({ timeout: 15000 });
  await expect(dbRow.getByText(status, { exact: true })).toBeVisible({
    timeout: timeout,
  });
};

/**
 * Waits for specific object to be removed from the table
 * @param page Page object
 * @param name Name of the object to look for (database, backup)
 * @param timeout How long to wait until error thrown
 */
export const waitForDelete = async (
  page: Page,
  name: string,
  timeout: number
) => {
  await page.reload({ waitUntil: 'networkidle' });
  await expect(page.getByRole('row').getByText(name)).toHaveCount(0, {
    timeout: timeout,
  });
};

export const waitForDbListLoad = async (page: Page) => {
  const rows = page.locator('.MuiTableRow-root');
  const start = Date.now();
  const timeout = 10000;

  while (Date.now() - start < timeout) {
    const count = await rows.count();
    if (count > 0) return;
    await page.waitForTimeout(200); // pause before retrying
  }
  throw new Error('Timed out waiting for DB list to load');
};
