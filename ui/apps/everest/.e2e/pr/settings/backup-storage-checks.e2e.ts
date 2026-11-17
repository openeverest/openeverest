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
import { goToUrl, limitedSuffixedName } from '@e2e/utils/generic';
import { EVEREST_CI_NAMESPACES, TIMEOUTS } from '@e2e/constants';
import { getCITokenFromLocalStorage } from '@e2e/utils/localStorage';
import {
  createBackupStorage,
  deleteBackupStorage,
  deleteSecret,
  getSecretResponse,
} from '@e2e/utils/backup-storage';
import {
  expectStorageState,
  fillS3Storage,
  openAddStorageDialog,
  S3StorageFormValues,
  STORAGES_PAGE,
  storageRow,
  submitStorageDialog,
} from '@e2e/utils/backup-storage-form';

const VALID: S3StorageFormValues = {
  bucket: 'pr-bs-checks',
  region: 'us-east-1',
  endpointURL: 'https://s3.us-east-1.amazonaws.com',
  accessKeyId: 'access-key',
  secretAccessKey: 'secret-key',
};

// What is refused in the form, what the API refuses and what core refuses
// once the storage is saved.
test.describe.serial('Backup storage checks', () => {
  test.describe.configure({ timeout: TIMEOUTS.FiveMinutes });

  const namespace = EVEREST_CI_NAMESPACES.EVEREST_UI;
  // Added through the API; its name is taken.
  const taken = limitedSuffixedName('pr-bs-dup');
  const spareSecret = limitedSuffixedName('pr-bs-dup-keys');
  // Added through the dialog with a bucket the s3 class's rules refuse.
  const misnamed = limitedSuffixedName('pr-bs-rule');
  let token: string;

  test.beforeAll(async ({ request }) => {
    token = await getCITokenFromLocalStorage();
    const response = await createBackupStorage(
      request,
      namespace,
      {
        name: taken,
        bucket: 'pr-bs-dup',
        region: 'us-east-1',
        endpointURL: 'https://s3.us-east-1.amazonaws.com',
        accessKeyId: 'access-key',
        secretAccessKey: 'secret-key',
        verifyTLS: true,
        forcePathStyle: false,
      },
      token
    );
    expect(response.status()).toBe(201);
  });

  test.beforeEach(async ({ page }) => {
    await goToUrl(page, STORAGES_PAGE);
  });

  test.afterAll(async ({ request }) => {
    for (const name of [taken, misnamed]) {
      await deleteBackupStorage(request, namespace, name, token);
    }
    await deleteSecret(request, namespace, spareSecret, token);
  });

  test('Refuses values the storage type or the API would refuse, before saving', async ({
    page,
  }) => {
    const writes: string[] = [];
    page.on('request', (request) => {
      if (
        request.method() !== 'GET' &&
        request.url().includes('/v1/clusters')
      ) {
        writes.push(`${request.method()} ${request.url()}`);
      }
    });
    const dialog = await openAddStorageDialog(page);
    const add = dialog.getByTestId('form-dialog-add');

    await fillS3Storage(page, dialog, {
      ...VALID,
      name: 'Not_A_Name',
      namespace,
      // The API takes a bucket without a slash, and the s3 class an http(s) URL.
      bucket: 'team/backups',
      endpointURL: 'ftp://s3.example.com',
    });

    await expect(
      dialog.getByText(
        'The name should only contain lowercase letters, numbers and hyphens.'
      )
    ).toBeVisible();
    await expect(dialog.getByLabel('Bucket')).toHaveAttribute(
      'aria-invalid',
      'true'
    );
    await expect(dialog.getByLabel('Endpoint URL')).toHaveAttribute(
      'aria-invalid',
      'true'
    );
    await expect(add).toBeDisabled();

    await dialog.getByTestId('text-input-name').fill('pr-bs-fixed');
    await dialog.getByLabel('Bucket').fill('team-backups');
    await dialog.getByLabel('Endpoint URL').fill('https://s3.example.com');
    await expect(add).toBeEnabled();
    expect(writes).toEqual([]);
  });

  test('Shows why core refused a storage that breaks a rule of its type', async ({
    page,
  }) => {
    const dialog = await openAddStorageDialog(page);
    await fillS3Storage(page, dialog, {
      ...VALID,
      name: misnamed,
      namespace,
      // The API takes it; an S3 bucket name has no capitals or underscores.
      bucket: 'Not_A_Bucket',
    });
    await submitStorageDialog(page, dialog, {
      method: 'POST',
      namespace,
      status: 201,
    });
    await expect(dialog).not.toBeVisible();

    await expectStorageState(page, misnamed, 'Not valid');
    await expect(
      storageRow(page, misnamed).getByText(
        'Its bucket or prefix breaks a rule of its storage type.'
      )
    ).toBeVisible();
  });

  test('Refuses a storage name taken in the namespace, leaving no Secret behind', async ({
    page,
    request,
  }) => {
    const dialog = await openAddStorageDialog(page);
    await fillS3Storage(page, dialog, {
      ...VALID,
      name: taken,
      namespace,
      credentialsSecretName: spareSecret,
    });
    const secretRolledBack = page.waitForResponse(
      (response) =>
        response.request().method() === 'DELETE' &&
        response.url().endsWith(`/secrets/${spareSecret}`)
    );
    await submitStorageDialog(page, dialog, {
      method: 'POST',
      namespace,
      status: 409,
    });

    await expect(page.getByText(/already exists/)).toBeVisible();
    await expect(dialog).toBeVisible();
    // The Secret the save had created for the storage is deleted again.
    await secretRolledBack;
    expect(
      (await getSecretResponse(request, namespace, spareSecret, token)).status()
    ).toBe(404);
  });
});
