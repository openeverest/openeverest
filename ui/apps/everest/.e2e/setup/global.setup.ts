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

import { test as setup } from '@playwright/test';
import 'dotenv/config';
import { getCITokenFromLocalStorage } from '../utils/localStorage';
import { doBackupCall } from '../utils/request';
import {
  CI_USER_STORAGE_STATE_FILE,
  getBucketNamespacesMap,
} from '../constants';
import { saveOldRBACPermissions } from '@e2e/utils/rbac-cmd-line';

const {
  EVEREST_LOCATION_ACCESS_KEY,
  EVEREST_LOCATION_SECRET_KEY,
  EVEREST_LOCATION_REGION,
  EVEREST_LOCATION_URL,
  EVEREST_S3_ACCESS_KEY,
  EVEREST_S3_SECRET_KEY,
  EVEREST_S3_REGION,
  EVEREST_S3_URL,
} = process.env;

setup.describe.serial('Global setup', () => {
  // setup('Login', async ({ page }) => {
  //   await login(page);
  //   await page.evaluate(() => {
  //     // This is a dummy token
  //     localStorage.setItem(
  //       'everestToken',
  //       'eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJpc3MiOiJldmVyZXN0IiwiaWF0IjoxNzM5NDU2NjU1LCJleHAiOjE3NzA5OTI2NTUsImF1ZCI6Ind3dy5leGFtcGxlLmNvbSIsInN1YiI6Impyb2NrZXRAZXhhbXBsZS5jb20iLCJHaXZlbk5hbWUiOiJKb2hubnkiLCJTdXJuYW1lIjoiUm9ja2V0IiwiRW1haWwiOiJqcm9ja2V0QGV4YW1wbGUuY29tIiwiUm9sZSI6WyJNYW5hZ2VyIiwiUHJvamVjdCBBZG1pbmlzdHJhdG9yIl19.NKJwQE1KY9srM9bZPZxL3zd_563ugnELMPFal9lFf78'
  //     );
  //   });
  //   // Wait for the user to be logged out and end up in login page again
  //   await page.waitForURL(/login/);
  //   await login(page);
  // });

  // setup('Close modal permanently', async ({ page }) => {
  //   // await page.goto('/');
  //   // await expect(page.getByTestId('lets-go-button')).toBeVisible();
  //   // await page.getByTestId('lets-go-button').click();
  //   // await page.context().storageState({ path: STORAGE_STATE_FILE });
  //
  //   try {
  //     await page.goto('/');
  //     const letsGoButton = page.getByTestId('lets-go-button');
  //     if (await letsGoButton.isVisible({ timeout: 3000 })) {
  //       await letsGoButton.click();
  //     }
  //   } catch {
  //     // Modal not visible, skip
  //   }
  //   await page.context().storageState({ path: STORAGE_STATE_FILE });
  // });

  // setup('Backup storages', async ({ request }) => {
  //   const token = await getTokenFromLocalStorage();
  //   const promises: Promise<any>[] = [];
  //   // This has a nested array structure, in the form of
  //   // [
  //   //   ['bucket1', 'namespace1'],
  //   //   ['bucket2', 'namespace2'],
  //   // ]
  //   const bucketNamespacesMap = getBucketNamespacesMap();
  //
  //   bucketNamespacesMap.forEach(async ([bucket, namespace]) => {
  //     const isEverestTesting = bucket === 'everest-testing';
  //     const data = isEverestTesting
  //       ? {
  //           name: bucket,
  //           description: 'CI test S3 bucket',
  //           type: 's3',
  //           bucketName: bucket,
  //           secretKey: EVEREST_S3_SECRET_KEY,
  //           accessKey: EVEREST_S3_ACCESS_KEY,
  //           allowedNamespaces: [],
  //           url: EVEREST_S3_URL,
  //           region: EVEREST_S3_REGION,
  //           verifyTLS: true,
  //           forcePathStyle: false,
  //         }
  //       : {
  //           name: bucket,
  //           description: 'CI test bucket',
  //           type: 's3',
  //           bucketName: bucket,
  //           secretKey: EVEREST_LOCATION_SECRET_KEY,
  //           accessKey: EVEREST_LOCATION_ACCESS_KEY,
  //           allowedNamespaces: [],
  //           url: EVEREST_LOCATION_URL,
  //           region: EVEREST_LOCATION_REGION,
  //           verifyTLS: false,
  //           forcePathStyle: true,
  //         };
  //
  //     promises.push(
  //       doBackupCall(() =>
  //         request.post(`/v1/namespaces/${namespace}/backup-storages/`, {
  //           data,
  //           headers: {
  //             Authorization: `Bearer ${token}`,
  //           },
  //         })
  //       )
  //     );
  //   });
  //   await Promise.all(promises);
  // });

  setup('Save old RBAC permissions', async ({ request }) => {
    await saveOldRBACPermissions();
  });
});
