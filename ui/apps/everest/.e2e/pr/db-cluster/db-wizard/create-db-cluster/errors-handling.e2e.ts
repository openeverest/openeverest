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
import { goToStep, moveForward } from '@e2e/utils/db-wizard';
import {
  fillScheduleModalForm,
  openCreateScheduleDialogFromDBWizard,
  selectDbEngine,
} from '../db-wizard-utils';
import { goToUrl, limitedSuffixedName } from '@e2e/utils/generic';
import { backupsStepCheckForPG } from '@e2e/pr/db-cluster/db-wizard/create-db-cluster/steps/backups-step';
import { EVEREST_CI_NAMESPACES, TIMEOUTS } from '@e2e/constants';
import {
  createDbClusterFn,
  deleteDbClusterFn,
  getDbClusterAPI,
} from '@e2e/utils/db-cluster';
import { basicInformationSelectNamespaceCheck } from '@e2e/pr/db-cluster/db-wizard/create-db-cluster/steps/basic-information-step';
import { getCITokenFromLocalStorage } from '@e2e/utils/localStorage';
import { advancedConfigurationStepCheckForPG } from '@e2e/pr/db-cluster/db-wizard/create-db-cluster/steps/advanced-configuration-step';

test.describe.parallel('DB cluster wizard errors handling', () => {
  test.describe.configure({ timeout: TIMEOUTS.FiveMinutes });

  test.beforeEach(async ({ page }) => {
    await goToUrl(page, '/databases');
  });

  test('Wizard form errors', async ({ page }) => {
    await test.step('Start DB cluster creation wizard', async () => {
      await selectDbEngine(page, 'postgresql');
    });

    await test.step('Resources step', async () => {
      // Resources Step
      await moveForward(page);
    });

    await test.step('Backup Schedules step', async () => {
      // Backups step
      await moveForward(page);
    });

    await test.step('Back to Resources step', async () => {
      await goToStep(page, 'resources');

      await expect(
        page.getByTestId('db-wizard-previous-button')
      ).not.toBeDisabled();
      await expect(
        page.getByTestId('db-wizard-continue-button')
      ).not.toBeDisabled();
      await expect(
        page.getByTestId('db-wizard-cancel-button')
      ).not.toBeDisabled();
      await expect(
        page.getByTestId('button-edit-preview-basic-information')
      ).not.toBeDisabled();
      await expect(
        page.getByTestId('button-edit-preview-backups')
      ).not.toBeDisabled();

      // Introduce an error on resources step
      await page.getByTestId('text-input-memory').fill('');
      await expect(
        page.getByTestId('preview-error-resources')
      ).not.toBeVisible();

      await expect(
        page.getByTestId('db-wizard-cancel-button')
      ).not.toBeDisabled();
    });

    await test.step('Backup Schedules step', async () => {
      // Backups step
      await moveForward(page);
    });

    await test.step('Advanced Configuration step', async () => {
      // Advanced Configurations step
      await moveForward(page);
      await advancedConfigurationStepCheckForPG(page);

      // await page
      //   .getByTestId('switch-input-external-access')
      //   .getByRole('checkbox')
      //   .check();
      await page.getByTestId('select-input-exposure-method').waitFor();
      await page.getByTestId('select-exposure-method-button').click();
      await page.getByRole('option', { name: 'Load balancer' }).click();
      // Introduce an error on advanced configs step: two invalid IPs
      await page
        .getByTestId('text-input-source-ranges.0.source-range')
        .fill('invalid-ip');
      await expect(
        page.getByTestId('preview-error-advanced-configurations')
      ).not.toBeVisible();
    });

    await test.step('Monitoring step', async () => {
      // Monitoring step
      await moveForward(page);
      await expect(page.getByTestId('db-wizard-submit-button')).toBeDisabled();
      await expect(page.getByTestId('preview-error-resources')).toBeVisible();
      await expect(
        page.getByTestId('preview-error-advanced-configurations')
      ).toBeVisible();
    });

    await test.step('Back to Resources step', async () => {
      await goToStep(page, 'resources');
      await page.getByTestId('text-input-memory').fill('1');
    });

    await test.step('Advanced Configuration step', async () => {
      await goToStep(page, 'advanced-configurations');
      await page
        .getByTestId('text-input-source-ranges.0.source-range')
        .fill('192.168.1.1');
    });

    await test.step('Monitoring step', async () => {
      await goToStep(page, 'monitoring');
      await expect(
        page.getByTestId('db-wizard-submit-button')
      ).not.toBeDisabled();
      await expect(
        page.getByTestId('preview-error-resources')
      ).not.toBeVisible();
      await expect(
        page.getByTestId('preview-error-advanced-configurations')
      ).not.toBeVisible();
    });
  });

  test('Duplicate backup schedules', async ({ page }) => {
    await test.step('Start DB cluster creation wizard', async () => {
      await selectDbEngine(page, 'postgresql');
    });

    await test.step('Resources step', async () => {
      // Move to Resources step
      await moveForward(page);
      // skip this step
    });

    await test.step('Backup Schedules step', async () => {
      // Move to "Scheduled Backups" step
      await moveForward(page);
      await backupsStepCheckForPG(page);
    });

    await test.step('Add duplicate backup schedule', async () => {
      await openCreateScheduleDialogFromDBWizard(page);
      await expect(page.getByTestId('same-schedule-warning')).not.toBeVisible();
      await fillScheduleModalForm(page, undefined, '1', undefined, undefined);
      await expect(page.getByTestId('same-schedule-warning')).toBeVisible({
        timeout: TIMEOUTS.TenSeconds,
      });
    });
  });

  test('Duplicate DB cluster name', async ({ page, request }) => {
    const dbClusterName = limitedSuffixedName('pr-db-wzd-err-dup'),
      namespace = EVEREST_CI_NAMESPACES.PG_ONLY,
      token = await getCITokenFromLocalStorage();
    expect(token).not.toHaveLength(0);

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

        await goToUrl(page, '/databases');
        await expect(page.getByText(dbClusterName)).toBeVisible({
          timeout: TIMEOUTS.TenSeconds,
        });
      });

      await test.step(`Create DB cluster with duplicate name`, async () => {
        await test.step('Start DB cluster creation wizard', async () => {
          await selectDbEngine(page, 'postgresql');
        });

        await test.step('Basic Info step', async () => {
          // namespace
          await basicInformationSelectNamespaceCheck(page, namespace);

          // db cluster name
          await page.getByTestId('text-input-db-name').fill(dbClusterName);

          await expect(
            page.getByText('You already have a database with the same name.')
          ).toBeVisible({ timeout: TIMEOUTS.ThirtySeconds });
        });
      });
    } finally {
      await deleteDbClusterFn(request, dbClusterName, namespace);
    }
  });
});
