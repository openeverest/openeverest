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

import { expect, test } from '@playwright/test';
import {
  createDbClusterFn,
  deleteDbClusterFn,
  getDbClusterAPI,
} from '@e2e/utils/db-cluster';
import { findDbAndClickActions } from '@e2e/utils/db-clusters-list';
import { goToUrl, limitedSuffixedName } from '@e2e/utils/generic';
import { EVEREST_CI_NAMESPACES, TIMEOUTS } from '@e2e/constants';
import { getCITokenFromLocalStorage } from '@e2e/utils/localStorage';

const testPrefix = 'pr-db-lst',
  namespace = EVEREST_CI_NAMESPACES.EVEREST_UI;
let token: string;

test.describe.parallel('DB clusters list', () => {
  test.beforeAll(async ({}) => {
    token = await getCITokenFromLocalStorage();
    expect(token).not.toHaveLength(0);
  });

  test.beforeEach(async ({ page }) => {
    await goToUrl(page, '/databases');
  });

  test('DB clusters Delete Action', async ({ page, request }) => {
    const dbClusterName = limitedSuffixedName(testPrefix + '-del');

    try {
      await test.step(`Create ${dbClusterName} DB cluster`, async () => {
        await createDbClusterFn(
          request,
          {
            dbName: dbClusterName,
            dbType: 'postgresql',
            numberOfNodes: '1',
          },
          namespace
        );
      });

      await test.step(`Wait for DB cluster ${dbClusterName} creation`, async () => {
        await expect(async () => {
          // new DB cluster appears in response not immediately.
          // wait for new DB cluster to appear.
          const dbCluster = await getDbClusterAPI(
            dbClusterName,
            namespace,
            request,
            token
          );
          expect(dbCluster).toBeDefined();
        }).toPass({
          intervals: [1000],
          timeout: TIMEOUTS.TenSeconds,
        });
      });

      await test.step(`Delete DB cluster ${dbClusterName}`, async () => {
        await goToUrl(page, '/databases');
        await expect(page.getByText(dbClusterName)).toBeVisible({
          timeout: TIMEOUTS.TenSeconds,
        });

        await findDbAndClickActions(page, dbClusterName, 'Delete');

        // Delete action
        await page.getByTestId(`${dbClusterName}-form-dialog`).waitFor();
        await expect(
          page.getByTestId('irreversible-action-alert')
        ).toBeVisible();
        const deleteConfirmationButton = page
          .getByRole('button')
          .filter({ hasText: 'Delete' });
        await expect(deleteConfirmationButton).toBeDisabled();
        await page.getByTestId('text-input-confirm-input').fill(dbClusterName);
        await expect(deleteConfirmationButton).toBeEnabled();

        const delResp = page.waitForResponse(
          (resp) =>
            resp.request().method() === 'DELETE' &&
            resp
              .url()
              .includes(
                `/v1/namespaces/${namespace}/database-clusters/${dbClusterName}`
              ) &&
            resp.status() === 204
        );
        await deleteConfirmationButton.click();
        await delResp;
      });
    } finally {
      await deleteDbClusterFn(request, dbClusterName, namespace);
    }
  });

  test.skip('DB cluster Paused/Resume', async () => {
    // TODO check update of Paused/Resume Status
  });

  test.skip('DB cluster Restarting', async () => {
    // TODO check updating of status
  });
});
