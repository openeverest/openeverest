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

import { test, expect } from '@playwright/test';
import { findRowAndClickActions, waitForDelete } from '@e2e/utils/table';
import { goToUrl, limitedSuffixedName } from '@e2e/utils/generic';
import { EVEREST_CI_NAMESPACES, TIMEOUTS } from '@e2e/constants';
import { getCITokenFromLocalStorage } from '@e2e/utils/localStorage';
import {
  deleteBackupStorage,
  getBackupStorage,
} from '@e2e/utils/backup-storage';

const {
  EVEREST_LOCATION_ACCESS_KEY,
  EVEREST_LOCATION_SECRET_KEY,
  EVEREST_LOCATION_REGION,
  EVEREST_LOCATION_URL,
} = process.env;

test.describe.serial('Backup storage', () => {
  test.describe.configure({ timeout: TIMEOUTS.FiveMinutes });

  const namespace = EVEREST_CI_NAMESPACES.EVEREST_UI,
    backupStorageName = limitedSuffixedName('pr-set-bac-str'),
    bucketName = limitedSuffixedName('bucket');
  let token: string;

  test.beforeAll(async ({}) => {
    token = await getCITokenFromLocalStorage();
    expect(token).not.toHaveLength(0);
  });

  test.beforeEach(async ({ page }) => {
    await goToUrl(page, '/settings/storage-locations');
  });

  test.afterAll(async ({ request }) => {
    await expect(async () => {
      await deleteBackupStorage(request, namespace, backupStorageName, token);
    }).toPass({
      intervals: [1000],
      timeout: TIMEOUTS.TenSeconds,
    });
  });

  test('Create Backup Storage', async ({ page, request }) => {
    await test.step(`Create Backup Storage`, async () => {
      await page.getByTestId('add-backup-storage').click();
      await page.getByTestId('text-input-name').fill(backupStorageName);
      await page.getByTestId('text-input-description').fill('test-description');

      await page.getByTestId('text-input-namespace').click();
      await page.getByRole('option', { name: namespace }).click();
      await expect(page.getByTestId('select-input-type')).toHaveValue('s3');
      await page.getByTestId('text-input-bucket-name').fill(bucketName);
      await page.getByTestId('text-input-region').fill(EVEREST_LOCATION_REGION);
      await page.getByTestId('text-input-url').fill(EVEREST_LOCATION_URL);
      await page
        .getByTestId('text-input-access-key')
        .fill(EVEREST_LOCATION_ACCESS_KEY);
      await page
        .getByTestId('text-input-secret-key')
        .fill(EVEREST_LOCATION_SECRET_KEY);
      await page.getByTestId('checkbox-verify-tls').setChecked(false);
      await page.getByTestId('checkbox-force-path-style').setChecked(true);
      await page.getByTestId('form-dialog-add').click();
    });

    await test.step(`Check created Backup Storage`, async () => {
      await expect(async () => {
        const response = await getBackupStorage(
          request,
          namespace,
          backupStorageName,
          token
        );
        expect(response).toBeDefined();
      }).toPass({
        intervals: [1000],
        timeout: TIMEOUTS.TenSeconds,
      });
    });
  });

  test('List Backup Storages', async ({ page }) => {
    const row = page
      .locator('.MuiTableRow-root')
      .filter({ hasText: backupStorageName });
    await expect(row).toBeVisible();
    await expect(row.getByText('S3 Compatible')).toBeVisible();
    await expect(row.getByText(bucketName)).toBeVisible();
    await expect(row.getByText(namespace)).toBeVisible();
  });

  test('Edit Backup Storage', async ({ page }) => {
    const newDescription = 'new-test-description';

    await test.step('Update Backup Storage description', async () => {
      await findRowAndClickActions(page, backupStorageName, 'Edit');
      await page.getByTestId('text-input-description').fill(newDescription);
      const updResponse = page.waitForResponse(
        (resp) =>
          resp.request().method() === 'PATCH' &&
          resp
            .url()
            .includes(
              `/v1/namespaces/${namespace}/backup-storages/${backupStorageName}`
            ) &&
          resp.status() === 200
      );
      await page.getByTestId('form-dialog-edit').click();
      await updResponse;
    });

    await test.step('Check updated Backup Storage description', async () => {
      await goToUrl(page, '/settings/storage-locations');
      await findRowAndClickActions(page, backupStorageName, 'Edit');
      await expect(page.getByTestId('text-input-description')).toHaveValue(
        newDescription
      );
      await page.getByTestId('form-dialog-cancel').click();
    });
  });

  test('Delete Backup Storage', async ({ page }) => {
    await findRowAndClickActions(page, backupStorageName, 'Delete');
    const delResponse = page.waitForResponse(
      (resp) =>
        resp.request().method() === 'DELETE' &&
        resp
          .url()
          .includes(
            `/v1/namespaces/${namespace}/backup-storages/${backupStorageName}`
          ) &&
        resp.status() === 204
    );
    await page.getByTestId('confirm-dialog-delete').click();
    await delResponse;

    await waitForDelete(page, backupStorageName, TIMEOUTS.TenSeconds);
  });
});
