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
  backupStorageCredentialsSecretName,
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

  const storageUrl = `/v1/clusters/${EVEREST_CI_CLUSTER}/namespaces/${namespace}/backup-storages`;
  let token: string;

  test.beforeAll(async () => {
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

  // The CI store is SeaweedFS behind a self-signed certificate.
  test('Add Backup Storage', async ({ page, request }) => {
    const dialog = page.getByRole('dialog');

    await test.step('Fill in the S3 forms', async () => {
      await page.getByTestId('add-backup-storage').click();
      await dialog.getByTestId('text-input-name').fill(backupStorageName);
      await dialog.getByTestId('text-input-namespace').click();
      await page.getByRole('option', { name: namespace, exact: true }).click();
      await dialog.getByTestId('select-class-name-button').click();
      await page
        .getByRole('option', { name: 'S3-compatible object storage' })
        .click();
      await dialog.getByLabel('Bucket').fill(bucketName);
      await dialog.getByLabel('Region').fill(EVEREST_LOCATION_REGION);
      await dialog.getByLabel('Endpoint URL').fill(EVEREST_LOCATION_URL);
      await dialog.getByLabel('Implementation').click();
      await page.getByRole('option', { name: 'Other S3-compatible' }).click();
      await dialog.getByLabel('Force path-style URLs').check();
      await dialog
        .getByLabel('Access key ID')
        .fill(EVEREST_LOCATION_ACCESS_KEY);
      await dialog
        .getByLabel('Secret access key')
        .fill(EVEREST_LOCATION_SECRET_KEY);
      await dialog.getByLabel('Skip TLS certificate verification').check();
    });

    await test.step('Add it', async () => {
      const created = page.waitForResponse(
        (resp) =>
          resp.request().method() === 'POST' &&
          resp.url().endsWith(storageUrl) &&
          resp.status() === 201
      );
      await page.getByTestId('form-dialog-add').click();
      await created;
      await expect(dialog).not.toBeVisible();
    });

    await test.step('Check it references a new credentials Secret', async () => {
      const storage = await getBackupStorage(
        request,
        namespace,
        backupStorageName,
        token
      );
      expect(storage.spec).toMatchObject({
        backupStorageClassRef: { name: 's3' },
        authentication: {
          mode: 'AccessKey',
          credentialsSecretRef: {
            name: backupStorageCredentialsSecretName(backupStorageName),
          },
        },
        location: { objectStore: { bucket: bucketName } },
        parameters: {
          region: EVEREST_LOCATION_REGION,
          endpointURL: EVEREST_LOCATION_URL,
          implementation: 'other',
          forcePathStyle: true,
        },
        tls: { insecureSkipVerify: true },
      });
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

  test('Edit Backup Storage', async ({ page, request }) => {
    const dialog = page.getByRole('dialog');
    const region = 'us-west-2';

    await test.step('Change its region', async () => {
      await findRowAndClickActions(page, backupStorageName, 'Edit');
      // Engines catalog backups by location, so it can't change.
      await expect(dialog.getByLabel('Bucket')).toBeDisabled();
      await dialog.getByLabel('Region').fill(region);
      const updated = page.waitForResponse(
        (resp) =>
          resp.request().method() === 'PUT' &&
          resp.url().endsWith(`${storageUrl}/${backupStorageName}`) &&
          resp.status() === 200
      );
      await page.getByTestId('form-dialog-save').click();
      await updated;
      await expect(dialog).not.toBeVisible();
    });

    await test.step('Check the rest stays as it was', async () => {
      const storage = await getBackupStorage(
        request,
        namespace,
        backupStorageName,
        token
      );
      expect(storage.spec).toMatchObject({
        location: { objectStore: { bucket: bucketName } },
        parameters: {
          region,
          endpointURL: EVEREST_LOCATION_URL,
          implementation: 'other',
          forcePathStyle: true,
        },
        authentication: {
          credentialsSecretRef: {
            name: backupStorageCredentialsSecretName(backupStorageName),
          },
        },
        tls: { insecureSkipVerify: true },
      });
    });
  });

  test('Delete Backup Storage', async ({ page }) => {
    await findRowAndClickActions(page, backupStorageName, 'Delete');
    const delResponse = page.waitForResponse(
      (resp) =>
        resp.request().method() === 'DELETE' &&
        resp.url().endsWith(`${storageUrl}/${backupStorageName}`) &&
        resp.status() === 204
    );
    await page.getByTestId('confirm-dialog-delete').click();
    await delResponse;

    await waitForDelete(page, backupStorageName, TIMEOUTS.TenSeconds);
  });
});
