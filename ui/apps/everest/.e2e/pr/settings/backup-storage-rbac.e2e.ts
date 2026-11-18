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

import { Page } from '@playwright/test';
import { test, expect } from '@e2e/fixtures/auth';
import { goToUrl, limitedSuffixedName } from '@e2e/utils/generic';
import {
  EVEREST_CI_CLUSTER,
  EVEREST_CI_NAMESPACES,
  TIMEOUTS,
} from '@e2e/constants';
import { getCITokenFromLocalStorage } from '@e2e/utils/localStorage';
import {
  backupStorageCredentialsSecretName,
  createBackupStorage,
  deleteBackupStorage,
  getBackupStorage,
} from '@e2e/utils/backup-storage';
import {
  openAddStorageDialog,
  openEditStorageDialog,
  S3_CLASS_TITLE,
  STORAGES_PAGE,
  storageRow,
  submitStorageDialog,
} from '@e2e/utils/backup-storage-form';
import { disableRBAC, setRBACUserPermissions } from '@e2e/utils/rbac-cmd-line';

type Permission = [string, string, string];

const namespace = EVEREST_CI_NAMESPACES.EVEREST_UI;
const inNamespace = (name: string) =>
  `${EVEREST_CI_CLUSTER}/${namespace}/${name}`;

const NAMESPACES: Permission = [
  'namespaces',
  'read',
  `${EVEREST_CI_CLUSTER}/*`,
];
const READ_STORAGES: Permission = ['backup-storages', 'read', inNamespace('*')];
const READ_CLASSES: Permission = [
  'backup-storage-classes',
  'read',
  `${EVEREST_CI_CLUSTER}/*`,
];

// The page reads the user's permissions when it loads.
const openStoragesAs = async (page: Page, permissions: Permission[]) => {
  await setRBACUserPermissions(permissions);
  await goToUrl(page, STORAGES_PAGE);
  await expect(page.getByRole('table')).toBeVisible();
};

const actionsMenu = (page: Page, name: string) =>
  storageRow(page, name).getByTestId('row-actions-menu-button');

// Runs as the RBAC user (see the project's config); the storages are seeded,
// and checked, as the admin.
test.describe.serial('Backup storage permissions', () => {
  test.describe.configure({ timeout: TIMEOUTS.FiveMinutes });

  const first = limitedSuffixedName('pr-bs-rb-a');
  const second = limitedSuffixedName('pr-bs-rb-b');
  const firstSecret = backupStorageCredentialsSecretName(first);
  let adminToken: string;

  test.beforeAll(async ({ request }) => {
    adminToken = await getCITokenFromLocalStorage();
    for (const name of [first, second]) {
      const response = await createBackupStorage(
        request,
        namespace,
        {
          name,
          bucket: `${name}-bucket`,
          region: 'us-east-1',
          endpointURL: 'https://s3.us-east-1.amazonaws.com',
          accessKeyId: 'access-key',
          secretAccessKey: 'secret-key',
          verifyTLS: true,
          forcePathStyle: false,
        },
        adminToken
      );
      expect(response.status()).toBe(201);
    }
  });

  test.afterAll(async ({ request }) => {
    await disableRBAC();
    for (const name of [first, second]) {
      await deleteBackupStorage(request, namespace, name, adminToken);
    }
  });

  test('Lists only the storages the user may read, with nothing to change them', async ({
    page,
  }) => {
    await openStoragesAs(page, [
      NAMESPACES,
      ['backup-storages', 'read', inNamespace(first)],
    ]);

    await expect(storageRow(page, first)).toBeVisible();
    await expect(storageRow(page, second)).toHaveCount(0);
    await expect(page.getByTestId('add-backup-storage')).toHaveCount(0);
    await expect(actionsMenu(page, first)).toHaveCount(0);
  });

  test('Offers Add only to a user who may also read the storage types', async ({
    page,
  }) => {
    const create: Permission = ['backup-storages', 'create', inNamespace('*')];

    await openStoragesAs(page, [NAMESPACES, READ_STORAGES, create]);
    await expect(storageRow(page, first)).toBeVisible();
    await expect(page.getByTestId('add-backup-storage')).toHaveCount(0);

    await openStoragesAs(page, [
      NAMESPACES,
      READ_STORAGES,
      create,
      READ_CLASSES,
    ]);
    const dialog = await openAddStorageDialog(page);
    await dialog.getByTestId('select-class-name-button').click();
    await expect(
      page.getByRole('option', { name: S3_CLASS_TITLE })
    ).toBeVisible();
  });

  test('Offers Edit and Delete storage by storage', async ({ page }) => {
    await openStoragesAs(page, [
      NAMESPACES,
      READ_STORAGES,
      READ_CLASSES,
      ['backup-storages', 'update', inNamespace(first)],
      ['backup-storages', 'delete', inNamespace(second)],
    ]);

    await actionsMenu(page, first).click();
    await expect(page.getByRole('menuitem', { name: 'Edit' })).toBeVisible();
    await expect(page.getByRole('menuitem', { name: 'Delete' })).toHaveCount(0);
    await page.keyboard.press('Escape');

    await actionsMenu(page, second).click();
    await expect(page.getByRole('menuitem', { name: 'Delete' })).toBeVisible();
    await expect(page.getByRole('menuitem', { name: 'Edit' })).toHaveCount(0);
  });

  test('Refuses an edit that needs use on its Secret, and saves it once the Secret may be used', async ({
    page,
    request,
  }) => {
    const mayEdit: Permission[] = [
      NAMESPACES,
      READ_STORAGES,
      READ_CLASSES,
      ['backup-storages', 'update', inNamespace(first)],
    ];
    const region = 'eu-west-1';

    await test.step('Without use on the Secret nothing is sent', async () => {
      await openStoragesAs(page, mayEdit);
      const dialog = await openEditStorageDialog(page, first);
      await dialog.getByLabel('Region').fill(region);
      await dialog.getByTestId('form-dialog-save').click();

      await expect(
        dialog.getByText(
          `Saving these settings needs permission to use Secret ${firstSecret} in the ${namespace} namespace, which you don't have.`
        )
      ).toBeVisible();
      const storage = await getBackupStorage(
        request,
        namespace,
        first,
        adminToken
      );
      expect(storage.spec.parameters.region).toBe('us-east-1');
    });

    await test.step('With use on that one Secret the edit is saved', async () => {
      await openStoragesAs(page, [
        ...mayEdit,
        ['secrets', 'use', inNamespace(firstSecret)],
      ]);
      const dialog = await openEditStorageDialog(page, first);
      await dialog.getByLabel('Region').fill(region);
      await submitStorageDialog(page, dialog, {
        method: 'PUT',
        namespace,
        status: 200,
      });
      await expect(dialog).not.toBeVisible();

      const storage = await getBackupStorage(
        request,
        namespace,
        first,
        adminToken
      );
      expect(storage.spec.parameters.region).toBe(region);
    });
  });
});
