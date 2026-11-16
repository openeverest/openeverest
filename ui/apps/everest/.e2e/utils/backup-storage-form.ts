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

import { expect, Locator, Page } from '@playwright/test';
import { EVEREST_CI_CLUSTER, TIMEOUTS } from '@e2e/constants';
import { findRowAndClickActions } from '@e2e/utils/table';

export const STORAGES_PAGE = '/settings/storage-locations';

export const S3_CLASS_TITLE = 'S3-compatible object storage';

export const backupStoragesUrl = (namespace: string) =>
  `/v1/clusters/${EVEREST_CI_CLUSTER}/namespaces/${namespace}/backup-storages`;

// What the S3 class's forms ask for; a value left out stays as the form has it.
export interface S3StorageFormValues {
  name?: string;
  namespace?: string;
  bucket?: string;
  prefix?: string;
  region?: string;
  endpointURL?: string;
  implementation?: string;
  forcePathStyle?: boolean;
  credentialsSecretName?: string;
  accessKeyId?: string;
  secretAccessKey?: string;
  skipTLSVerification?: boolean;
}

export const storageDialog = (page: Page) => page.getByRole('dialog');

export const storageRow = (page: Page, name: string) =>
  page.locator('.MuiTableRow-root').filter({ hasText: name });

export const openAddStorageDialog = async (page: Page) => {
  await page.getByTestId('add-backup-storage').click();
  const dialog = storageDialog(page);
  await expect(dialog).toBeVisible();
  return dialog;
};

export const openEditStorageDialog = async (page: Page, name: string) => {
  await findRowAndClickActions(page, name, 'Edit');
  const dialog = storageDialog(page);
  // The class's forms show once the dialog has read the storage.
  await expect(dialog.getByLabel('Bucket')).toBeVisible();
  return dialog;
};

export const pickS3Class = async (page: Page, dialog: Locator) => {
  await dialog.getByTestId('select-class-name-button').click();
  await page.getByRole('option', { name: S3_CLASS_TITLE }).click();
  await expect(dialog.getByLabel('Bucket')).toBeVisible();
};

export const fillS3Credentials = async (
  dialog: Locator,
  { accessKeyId, secretAccessKey }: S3StorageFormValues
) => {
  if (accessKeyId !== undefined) {
    await dialog.getByLabel('Access key ID').fill(accessKeyId);
  }
  if (secretAccessKey !== undefined) {
    await dialog.getByLabel('Secret access key').fill(secretAccessKey);
  }
};

// Fills the add dialog: name and namespace, then the S3 class's forms.
export const fillS3Storage = async (
  page: Page,
  dialog: Locator,
  values: S3StorageFormValues
) => {
  if (values.name !== undefined) {
    await dialog.getByTestId('text-input-name').fill(values.name);
  }
  if (values.namespace !== undefined) {
    await dialog.getByTestId('text-input-namespace').click();
    await page
      .getByRole('option', { name: values.namespace, exact: true })
      .click();
  }
  await pickS3Class(page, dialog);
  if (values.bucket !== undefined) {
    await dialog.getByLabel('Bucket').fill(values.bucket);
  }
  if (values.prefix !== undefined) {
    await dialog.getByLabel('Prefix').fill(values.prefix);
  }
  if (values.region !== undefined) {
    await dialog.getByLabel('Region').fill(values.region);
  }
  if (values.endpointURL !== undefined) {
    await dialog.getByLabel('Endpoint URL').fill(values.endpointURL);
  }
  if (values.implementation !== undefined) {
    await dialog.getByLabel('Implementation').click();
    await page.getByRole('option', { name: values.implementation }).click();
  }
  if (values.forcePathStyle) {
    await dialog.getByLabel('Force path-style URLs').check();
  }
  if (values.credentialsSecretName !== undefined) {
    await dialog
      .getByLabel('Credentials Secret name')
      .fill(values.credentialsSecretName);
  }
  await fillS3Credentials(dialog, values);
  if (values.skipTLSVerification) {
    await dialog.getByLabel('Skip TLS certificate verification').check();
  }
};

// Submits the dialog and waits for the storage write the API answers with
// the status; the dialog closes only once the write succeeded.
export const submitStorageDialog = async (
  page: Page,
  dialog: Locator,
  {
    method,
    namespace,
    status,
  }: { method: 'POST' | 'PUT'; namespace: string; status: number }
) => {
  const written = page.waitForResponse(
    (response) =>
      response.request().method() === method &&
      response.url().includes(backupStoragesUrl(namespace)) &&
      response.status() === status
  );
  await dialog
    .getByTestId(method === 'POST' ? 'form-dialog-add' : 'form-dialog-save')
    .click();
  await written;
};

// Core accepts or refuses a storage shortly after each change of it or of
// its Secret; the list follows without a reload.
export const expectStorageState = (
  page: Page,
  name: string,
  state: 'Valid' | 'Not valid'
) =>
  expect(storageRow(page, name).getByText(state, { exact: true })).toBeVisible({
    timeout: TIMEOUTS.ThirtySeconds,
  });
