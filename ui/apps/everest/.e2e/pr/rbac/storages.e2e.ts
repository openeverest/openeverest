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

import { getCITokenFromLocalStorage } from '@e2e/utils/localStorage';
import { getNamespacesFn } from '@e2e/utils/namespaces';
import { setRBACPermissionsK8S } from '@e2e/utils/rbac-cmd-line';
import { expect, test } from '@playwright/test';
import { MOCK_STORAGE_NAME, mockStorages } from './utils';

const { CI_USER: user } = process.env;

test.describe('Backup Storages RBAC', () => {
  let namespace = '';
  test.beforeAll(async ({ request }) => {
    await setRBACPermissionsK8S([['namespaces', 'read', '*']]);
    const token = await getCITokenFromLocalStorage();
    const namespaces = await getNamespacesFn(token, request);
    namespace = namespaces[0];
  });

  test('Show Backup Storages', async ({ page }) => {
    await setRBACPermissionsK8S([
      ['namespaces', 'read', namespace],
      ['backup-storages', 'read', `${namespace}/${MOCK_STORAGE_NAME}`],
    ]);
    await mockStorages(page, namespace);
    await page.goto('/settings/storage-locations');
    await expect(page.getByText(MOCK_STORAGE_NAME)).toBeVisible();
    await expect(page.getByTestId('add-backup-storage')).not.toBeVisible();
  });

  test('Hide Backup Storages when no namespaces allowed', async ({ page }) => {
    await setRBACPermissionsK8S([
      ['backup-storages', 'read', `${namespace}/${MOCK_STORAGE_NAME}`],
    ]);
    await mockStorages(page, namespace);
    await page.goto('/settings/storage-locations');
    await expect(page.getByRole('table')).toBeVisible();
    await expect(page.getByText(MOCK_STORAGE_NAME)).not.toBeVisible();
  });

  test('Hide Backup Storages when no storage allowed', async ({ page }) => {
    await setRBACPermissionsK8S([['namespaces', 'read', namespace]]);
    await mockStorages(page, namespace);
    await page.goto('/settings/storage-locations');
    await expect(page.getByText(MOCK_STORAGE_NAME)).not.toBeVisible();
  });

  test('Create Backup Storages', async ({ page }) => {
    await setRBACPermissionsK8S([
      ['namespaces', 'read', namespace],
      ['backup-storages', 'read', `${namespace}/${MOCK_STORAGE_NAME}`],
      ['backup-storages', 'create', `${namespace}/*`],
      // A storage is added from its class's forms.
      ['backup-storage-classes', 'read', '*'],
    ]);
    await mockStorages(page, namespace);
    await page.goto('/settings/storage-locations');
    await expect(page.getByText(MOCK_STORAGE_NAME)).toBeVisible();
    await expect(page.getByTestId('add-backup-storage')).toBeVisible();
  });

  test('Hide create Backup Storages button when no namespace available', async ({
    page,
  }) => {
    await setRBACPermissionsK8S([
      ['backup-storages', 'read', `${namespace}/${MOCK_STORAGE_NAME}`],
      ['backup-storages', 'create', `${namespace}/*`],
    ]);
    await mockStorages(page, namespace);
    await page.goto('/settings/storage-locations');
    await expect(page.getByRole('table')).toBeVisible();
    await expect(page.getByTestId('add-backup-storage')).not.toBeVisible();
  });
});
