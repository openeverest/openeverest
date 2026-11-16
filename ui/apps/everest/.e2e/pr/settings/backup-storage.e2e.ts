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

import { test, expect } from '@e2e/fixtures/auth';
import { findRowAndClickActions, waitForDelete } from '@e2e/utils/table';
import { goToUrl, limitedSuffixedName } from '@e2e/utils/generic';
import {
  EVEREST_CI_NAMESPACES,
  EVEREST_CI_CLUSTER,
  TIMEOUTS,
} from '@e2e/constants';
import { getCITokenFromLocalStorage } from '@e2e/utils/localStorage';
import {
  createBackupStorage,
  deleteBackupStorage,
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

  test.beforeAll(async ({ request }) => {
    token = await getCITokenFromLocalStorage();
    expect(token).not.toHaveLength(0);

    const response = await createBackupStorage(
      request,
      namespace,
      {
        name: backupStorageName,
        bucket: bucketName,
        region: EVEREST_LOCATION_REGION,
        endpointURL: EVEREST_LOCATION_URL,
        implementation: 'other',
        accessKeyId: EVEREST_LOCATION_ACCESS_KEY,
        secretAccessKey: EVEREST_LOCATION_SECRET_KEY,
        verifyTLS: false,
        forcePathStyle: true,
      },
      token
    );
    expect(response.status()).toBe(201);
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

  test('List Backup Storages', async ({ page }) => {
    const row = page
      .locator('.MuiTableRow-root')
      .filter({ hasText: backupStorageName });
    await expect(row).toBeVisible();
    await expect(row.getByText('S3-compatible object storage')).toBeVisible();
    await expect(row.getByText(bucketName)).toBeVisible();
    await expect(row.getByText(namespace)).toBeVisible();
    // Core accepts the storage once it has checked it against its class.
    await expect(row.getByText('Valid', { exact: true })).toBeVisible({
      timeout: TIMEOUTS.ThirtySeconds,
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
            `/v1/clusters/${EVEREST_CI_CLUSTER}/namespaces/${namespace}/backup-storages/${backupStorageName}`
          ) &&
        resp.status() === 204
    );
    await page.getByTestId('confirm-dialog-delete').click();
    await delResponse;

    await waitForDelete(page, backupStorageName, TIMEOUTS.TenSeconds);
  });
});
