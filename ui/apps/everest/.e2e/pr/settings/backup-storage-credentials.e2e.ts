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
import { findRowAndClickActions, waitForDelete } from '@e2e/utils/table';
import {
  backupStorageCredentialsSecretName,
  createBackupStorage,
  createBackupStorageWithoutItsSecret,
  deleteBackupStorage,
  getBackupStorage,
  getBackupStorageResponse,
  getSecretResponse,
  S3BackupStorage,
} from '@e2e/utils/backup-storage';
import {
  expectStorageState,
  fillS3Credentials,
  fillS3Storage,
  openAddStorageDialog,
  openEditStorageDialog,
  S3StorageFormValues,
  STORAGES_PAGE,
  storageRow,
  submitStorageDialog,
} from '@e2e/utils/backup-storage-form';

const S3_CLASS_SECRET_LABELS = {
  'openeverest.io/backup-storage-class': 's3',
  'openeverest.io/definition': 'AccessKey',
};

const seed = (name: string): S3BackupStorage => ({
  name,
  bucket: `${name}-bucket`,
  region: 'us-east-1',
  endpointURL: 'https://s3.us-east-1.amazonaws.com',
  accessKeyId: 'access-key',
  secretAccessKey: 'secret-key',
  verifyTLS: true,
  forcePathStyle: false,
});

const formValues = (name: string, namespace: string): S3StorageFormValues => {
  const { bucket, region, endpointURL, accessKeyId, secretAccessKey } =
    seed(name);
  return {
    name,
    namespace,
    bucket,
    region,
    endpointURL,
    accessKeyId,
    secretAccessKey,
  };
};

// Credentials live in a Secret the storage names; the form creates, reuses,
// updates and repairs that Secret.
test.describe.serial('Backup storage credentials', () => {
  test.describe.configure({ timeout: TIMEOUTS.FiveMinutes });

  const namespace = EVEREST_CI_NAMESPACES.EVEREST_UI;
  // Added through the API with its own Secret.
  const owner = limitedSuffixedName('pr-bs-own');
  const ownerSecret = backupStorageCredentialsSecretName(owner);
  // Added through the dialog, on the owner's Secret.
  const sharer = limitedSuffixedName('pr-bs-shr');
  // Added through the API, naming a Secret that doesn't exist.
  const orphan = limitedSuffixedName('pr-bs-orp');
  const orphanSecret = backupStorageCredentialsSecretName(orphan);
  let token: string;

  test.beforeAll(async ({ request }) => {
    token = await getCITokenFromLocalStorage();
    expect(
      (
        await createBackupStorage(request, namespace, seed(owner), token)
      ).status()
    ).toBe(201);
    expect(
      (
        await createBackupStorageWithoutItsSecret(
          request,
          namespace,
          seed(orphan),
          token
        )
      ).status()
    ).toBe(201);
  });

  test.beforeEach(async ({ page }) => {
    await goToUrl(page, STORAGES_PAGE);
  });

  test.afterAll(async ({ request }) => {
    for (const name of [sharer, orphan, owner]) {
      await deleteBackupStorage(request, namespace, name, token);
    }
  });

  test('Points to the existing Secret when its name is taken, and reuses it', async ({
    page,
    request,
  }) => {
    const dialog = await openAddStorageDialog(page);

    await test.step('A taken Secret name is refused before the storage is added', async () => {
      await fillS3Storage(page, dialog, {
        ...formValues(sharer, namespace),
        credentialsSecretName: ownerSecret,
      });
      await dialog.getByTestId('form-dialog-add').click();

      await expect(
        dialog.getByText(
          `A Secret named ${ownerSecret} already exists. Pick it under "Use an existing Secret", or enter another Credentials Secret name.`
        )
      ).toBeVisible();
      expect(
        (
          await getBackupStorageResponse(request, namespace, sharer, token)
        ).status()
      ).toBe(404);
    });

    await test.step('The Secret is offered as an existing one of this type and mode', async () => {
      await dialog.getByLabel('Use an existing Secret').check();
      await dialog.getByTestId('select-existing-secret-name-button').click();
      await page.getByRole('option', { name: ownerSecret }).click();
      await submitStorageDialog(page, dialog, {
        method: 'POST',
        namespace,
        status: 201,
      });
      await expect(dialog).not.toBeVisible();
    });

    await test.step('The new storage names that Secret and has none of its own', async () => {
      const storage = await getBackupStorage(request, namespace, sharer, token);
      expect(storage.spec.authentication).toEqual({
        mode: 'AccessKey',
        credentialsSecretRef: { name: ownerSecret },
      });
      expect(
        (
          await getSecretResponse(
            request,
            namespace,
            backupStorageCredentialsSecretName(sharer),
            token
          )
        ).status()
      ).toBe(404);
      await expectStorageState(page, sharer, 'Valid');
    });
  });

  test('Replaces the credentials in the Secret, leaving the storage as it is', async ({
    page,
    request,
  }) => {
    const before = await getBackupStorage(request, namespace, owner, token);
    const secretBefore = await (
      await getSecretResponse(request, namespace, ownerSecret, token)
    ).json();

    const dialog = await openEditStorageDialog(page, owner);
    await dialog.getByLabel('Enter new credentials').check();
    await fillS3Credentials(dialog, {
      accessKeyId: 'rotated-access-key',
      secretAccessKey: 'rotated-secret-key',
    });
    const secretPatched = page.waitForResponse(
      (response) =>
        response.request().method() === 'PATCH' &&
        response.url().endsWith(`/secrets/${ownerSecret}`) &&
        response.status() === 200
    );
    await dialog.getByTestId('form-dialog-save').click();
    await secretPatched;
    await expect(dialog).not.toBeVisible();

    const secretAfter = await (
      await getSecretResponse(request, namespace, ownerSecret, token)
    ).json();
    expect(secretAfter.metadata.resourceVersion).not.toBe(
      secretBefore.metadata.resourceVersion
    );
    const after = await getBackupStorage(request, namespace, owner, token);
    expect(after.spec).toEqual(before.spec);
  });

  test('Repairs a storage whose Secret is gone with new credentials', async ({
    page,
    request,
  }) => {
    await expectStorageState(page, orphan, 'Not valid');
    await expect(
      storageRow(page, orphan).getByText(
        'Its credentials Secret, or a key it needs, is missing.'
      )
    ).toBeVisible();

    const dialog = await openEditStorageDialog(page, orphan);
    await dialog.getByLabel('Enter new credentials').check();
    await fillS3Credentials(dialog, {
      accessKeyId: 'access-key',
      secretAccessKey: 'secret-key',
    });
    await dialog.getByTestId('form-dialog-save').click();
    await expect(dialog).not.toBeVisible();

    // Core accepts the storage as soon as its Secret is back.
    await expectStorageState(page, orphan, 'Valid');
    const secret = await (
      await getSecretResponse(request, namespace, orphanSecret, token)
    ).json();
    expect(secret.metadata.labels).toMatchObject(S3_CLASS_SECRET_LABELS);
  });

  test('Deleting a storage keeps the Secret another storage shares', async ({
    page,
    request,
  }) => {
    await findRowAndClickActions(page, sharer, 'Delete');
    await page.getByTestId('confirm-dialog-delete').click();
    await waitForDelete(page, sharer, TIMEOUTS.TenSeconds);

    expect(
      (await getSecretResponse(request, namespace, ownerSecret, token)).status()
    ).toBe(200);
    await expectStorageState(page, owner, 'Valid');
  });
});
