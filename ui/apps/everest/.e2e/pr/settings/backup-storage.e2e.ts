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
import { EVEREST_CI_NAMESPACES, TIMEOUTS } from '@e2e/constants';
import { getCITokenFromLocalStorage } from '@e2e/utils/localStorage';
import {
  backupStorageCredentialsSecretName,
  deleteBackupStorage,
  getBackupStorage,
} from '@e2e/utils/backup-storage';
import {
  backupStoragesUrl,
  expectStorageState,
  fillS3Storage,
  openAddStorageDialog,
  openEditStorageDialog,
  S3_CLASS_TITLE,
  STORAGES_PAGE,
  storageRow,
  submitStorageDialog,
} from '@e2e/utils/backup-storage-form';

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

  test.beforeAll(async () => {
    token = await getCITokenFromLocalStorage();
    expect(token).not.toHaveLength(0);
  });

  test.beforeEach(async ({ page }) => {
    await goToUrl(page, STORAGES_PAGE);
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
    const dialog = await openAddStorageDialog(page);

    await test.step('Fill in the S3 forms and add it', async () => {
      await fillS3Storage(page, dialog, {
        name: backupStorageName,
        namespace,
        bucket: bucketName,
        region: EVEREST_LOCATION_REGION,
        endpointURL: EVEREST_LOCATION_URL,
        implementation: 'Other S3-compatible',
        forcePathStyle: true,
        accessKeyId: EVEREST_LOCATION_ACCESS_KEY,
        secretAccessKey: EVEREST_LOCATION_SECRET_KEY,
        skipTLSVerification: true,
      });
      await submitStorageDialog(page, dialog, {
        method: 'POST',
        namespace,
        status: 201,
      });
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

    await test.step('The list shows it as valid without a reload', async () => {
      await expectStorageState(page, backupStorageName, 'Valid');
    });
  });

  test('List Backup Storages', async ({ page }) => {
    const row = storageRow(page, backupStorageName);
    await expect(row).toBeVisible();
    await expect(row.getByText(S3_CLASS_TITLE)).toBeVisible();
    await expect(row.getByText(bucketName)).toBeVisible();
    await expect(row.getByText(namespace)).toBeVisible();
    await expectStorageState(page, backupStorageName, 'Valid');
  });

  test('Edit Backup Storage', async ({ page, request }) => {
    const region = 'us-west-2';

    await test.step('Change its region', async () => {
      const dialog = await openEditStorageDialog(page, backupStorageName);
      // Engines catalog backups by location, so it can't change.
      await expect(dialog.getByLabel('Bucket')).toBeDisabled();
      await dialog.getByLabel('Region').fill(region);
      await submitStorageDialog(page, dialog, {
        method: 'PUT',
        namespace,
        status: 200,
      });
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
        resp
          .url()
          .endsWith(`${backupStoragesUrl(namespace)}/${backupStorageName}`) &&
        resp.status() === 204
    );
    await page.getByTestId('confirm-dialog-delete').click();
    await delResponse;

    await waitForDelete(page, backupStorageName, TIMEOUTS.TenSeconds);
  });
});
