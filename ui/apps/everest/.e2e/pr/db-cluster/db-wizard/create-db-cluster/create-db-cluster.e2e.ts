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

import { expect, test } from '@playwright/test';
import {
  getEnginesLatestRecommendedVersions,
  getEnginesVersions,
} from '@e2e/utils/database-engines';
import { deleteDbClusterFn, getDbClusterAPI } from '@e2e/utils/db-cluster';
import { getCITokenFromLocalStorage } from '@e2e/utils/localStorage';
import {
  advancedConfigurationStepCheckForPG,
  dbSummaryAdvancedConfigurationCheckForPG,
} from './steps/advanced-configuration-step';
import {
  backupsStepCheckForPG,
  dbSummaryBackupsCheckForPG,
} from './steps/backups-step';
import {
  basicInformationSelectNamespaceCheck,
  basicInformationStepCheckForPG,
  dbSummaryBasicInformationCheckForPG,
} from './steps/basic-information-step';
import {
  resourcesStepCheckForPG,
  dbSummaryResourcesCheckForPG,
} from './steps/resources-step';
import {
  cancelWizard,
  goToLastAndSubmit,
  goToStep,
  moveForward,
  setPitrEnabledStatus,
  submitWizard,
} from '@e2e/utils/db-wizard';
import { checkAmountOfDbEngines, selectDbEngine } from '../db-wizard-utils';
import { findDbAndClickActions } from '@e2e/utils/db-clusters-list';
import { waitForInitializingState } from '@e2e/utils/table';
import { EVEREST_CI_NAMESPACES, TIMEOUTS } from '@e2e/constants';
import { goToUrl, limitedSuffixedName } from '@e2e/utils/generic';
import {
  dbSummaryMonitoringCheck,
  monitoringStepCheck,
} from '@e2e/pr/db-cluster/db-wizard/create-db-cluster/steps/monitoring-step';

const namespace = EVEREST_CI_NAMESPACES.PG_ONLY,
  testPrefix = 'pr-db-wzd';

let availableEngineVersions = {
    pxc: [],
    psmdb: [],
    postgresql: [],
  },
  recommendedEngineVersions = {
    pxc: '',
    psmdb: '',
    postgresql: '',
  },
  token: string;

test.describe.parallel('DB cluster wizard creation', () => {
  test.describe.configure({ timeout: TIMEOUTS.FiveMinutes });

  // IST is UTC+5h30, with or without DST
  test.use({
    timezoneId: 'IST',
  });

  test.beforeAll(async ({ request }) => {
    token = await getCITokenFromLocalStorage();
    expect(token).not.toHaveLength(0);

    availableEngineVersions = await getEnginesVersions(
      token,
      namespace,
      request
    );
    // pg-only namespace has only 1 operator installed
    expect(availableEngineVersions.postgresql).not.toHaveLength(0);
    expect(availableEngineVersions.pxc).toHaveLength(0);
    expect(availableEngineVersions.psmdb).toHaveLength(0);

    recommendedEngineVersions = await getEnginesLatestRecommendedVersions(
      namespace,
      request
    );
    // pg-only namespace has only 1 operator installed
    expect(recommendedEngineVersions.postgresql).not.toHaveLength(0);
    expect(recommendedEngineVersions.pxc).toHaveLength(0);
    expect(recommendedEngineVersions.psmdb).toHaveLength(0);
  });

  test.beforeEach(async ({ page }) => {
    await goToUrl(page, '/databases');
  });

  test.skip('Cluster creation with an incomplete list of DBEngines', () => {
    // TODO after the https://jira.percona.com/browse/EVEREST-203 is ready
    // 1) rewrite the starting pipeline of launching everest to launch everest without clusters
    // 2) add 2 clusters using new methods from EVEREST-203
    // 3) check that the default parameters for MySQL are changed with parameters for the first available dbEngine
  });

  test('Cluster defaults for namespaces with all operators', async ({
    page,
  }) => {
    const expectedNodesOrder = [3, 3, 2];
    const dbEnginesButtons = await checkAmountOfDbEngines(page);
    expect(await dbEnginesButtons.count()).toBe(3);
    // TODO expect all buttons available and not disabled

    for (let i = 0; i < 3; i++) {
      await test.step(`Run wizard for ${await dbEnginesButtons.nth(i).textContent()} DB engine`, async () => {
        await dbEnginesButtons.nth(i).click();

        await test.step('Basic Info step', async () => {
          await page
            .getByTestId('select-input-db-version')
            .waitFor({ timeout: TIMEOUTS.TenSeconds });
          expect(
            await page.getByTestId('select-input-db-version').inputValue()
          ).toBeDefined();
        });

        await test.step('Resources step', async () => {
          // move to Resources step
          await moveForward(page);

          await expect(
            page.getByTestId(`toggle-button-nodes-${expectedNodesOrder[i]}`)
          ).toHaveAttribute('aria-pressed', 'true');

          // We click on the first button to make sure it always goes back to defaults afterwards
          await page.getByTestId('toggle-button-nodes-1').click();
        });

        await test.step('Back to /databases page', async () => {
          // We return to databases page to choose other db
          await goToUrl(page, '/databases');

          await page
            .getByTestId('add-db-cluster-button')
            .waitFor({ timeout: TIMEOUTS.TenSeconds });
          await page.getByTestId('add-db-cluster-button').click();
        });
      });
    }
  });

  test('Cluster creation successful', async ({ page, request }) => {
    const clusterName = limitedSuffixedName(testPrefix + '-crt-psm');

    await test.step('Start DB cluster creation wizard', async () => {
      await selectDbEngine(page, 'postgresql');
    });

    await test.step('Basic Info step', async () => {
      await basicInformationStepCheckForPG(
        page,
        namespace,
        availableEngineVersions,
        recommendedEngineVersions,
        clusterName
      );
    });

    await test.step('Resources step', async () => {
      // Move to Resources step
      await moveForward(page);
      await resourcesStepCheckForPG(page);
    });

    await test.step('Backup Schedules step', async () => {
      // Move to "Scheduled Backups" step
      await moveForward(page);
      await backupsStepCheckForPG(page);
    });

    await test.step('Advanced Configuration step', async () => {
      // Move to "Advanced Configuration" step
      await moveForward(page);
      await advancedConfigurationStepCheckForPG(page);
    });

    await test.step('Monitoring step', async () => {
      // Go to "Monitoring" step
      await moveForward(page);
      await monitoringStepCheck(page);
    });

    await test.step('Submit wizard', async () => {
      await submitWizard(page);
    });

    try {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      let addedCluster: any;
      await test.step('Wait for DB cluster creation', async () => {
        await expect(async () => {
          // new DB cluster appears in response not immediately
          addedCluster = await getDbClusterAPI(
            clusterName,
            namespace,
            request,
            token
          );
          expect(addedCluster).toBeDefined();
        }).toPass({
          intervals: [1000],
          timeout: TIMEOUTS.TenSeconds,
        });
      });

      expect(addedCluster?.spec.engine.type).toBe('postgresql');
      expect(addedCluster?.spec.engine.replicas).toBe(1);
      expect(addedCluster?.spec.engine.resources?.cpu.toString()).toBe('600m');
      expect(addedCluster?.spec.engine.resources?.memory.toString()).toBe('1G');
      expect(addedCluster?.spec.engine.storage.size.toString()).toBe('1Gi');

      expect(addedCluster?.spec.proxy.expose.type).toBe('ClusterIP');
      expect(addedCluster?.spec.proxy.replicas).toBe(1);
      expect(addedCluster?.spec.proxy.resources.cpu).toBe('1');
      expect(addedCluster?.spec.proxy.resources.memory).toBe('30M');

      // expect(addedCluster?.spec.proxy.expose.ipSourceRanges).toEqual([
      //   '192.168.1.1/24',
      //   '192.168.1.0',
      // ]);
      expect(addedCluster?.spec.backup.schedules[0].retentionCopies).toBe(1);
      // Verify timezone conversion was applied to the schedule cron
      // Day 10, 1h05 in IST timezone is day 9, 19h35 UTC
      expect(addedCluster?.spec.backup.schedules[0].schedule).toBe(
        '35 19 9 * *'
      );
    } finally {
      await deleteDbClusterFn(request, clusterName, namespace);
    }
  });

  test('Cluster creation with back operations successful', async ({
    page,
    request,
  }) => {
    const clusterName = limitedSuffixedName(testPrefix + '-crt-psm');
    await selectDbEngine(page, 'postgresql');

    await test.step('Basic Info step', async () => {
      await basicInformationStepCheckForPG(
        page,
        namespace,
        availableEngineVersions,
        recommendedEngineVersions,
        clusterName
      );
    });

    await test.step('Resources step', async () => {
      // Move to Resources step
      await moveForward(page);
      await resourcesStepCheckForPG(page);
    });

    await test.step('Backup Schedules step', async () => {
      // Move to "Scheduled Backups" step
      await moveForward(page);
      await backupsStepCheckForPG(page);
    });

    await test.step('Advanced Configuration step', async () => {
      // Move to "Advanced Configuration" step
      await moveForward(page);
      await advancedConfigurationStepCheckForPG(page);
    });

    await test.step('Monitoring step', async () => {
      // Go to "Monitoring" step
      await moveForward(page);
      await monitoringStepCheck(page);
    });

    await test.step('Back to Basic Info step', async () => {
      // Test the mechanism for default number of nodes
      await goToStep(page, 'basic-information');
      // Here we test that version wasn't reset to default
      await expect(
        page
          .getByTestId('section-basic-information')
          .getByTestId('preview-content')
          .getByText('Version: ' + recommendedEngineVersions.postgresql)
      ).toBeVisible();

      // Make sure name doesn't change when we go back to first step
      expect(await page.getByTestId('text-input-db-name').inputValue()).toBe(
        clusterName
      );
    });

    // TODO should we move next lines to separate PG/Mongo tests or we already have it in release folder?
    // Now we change the number of nodes

    // await page.getByTestId('button-edit-preview-resources').click();
    // await page.getByTestId('toggle-button-nodes-3').click();
    // await expect(page.getByText('PG Bouncers (3)')).toBeVisible();
    // await page.getByTestId('toggle-button-nodes-2').click();

    // Because 2 nodes is not valid for MongoDB, the default will be picked
    // await page.getByTestId('mongodb-toggle-button').click();
    // await page
    //   .getByTestId('switch-input-sharding-label')
    //   .getByRole('checkbox')
    //   .check();
    // await expect(page.getByText('Nº nodes: 3')).toBeVisible();

    // await page.getByTestId('button-edit-preview-backups').click();

    // await expect(page.getByTestId('radio-option-logical')).not.toBeVisible();

    // await page.getByTestId('button-edit-preview-monitoring').click();
    // await monitoringStepCheck(page, monitoringInstancesList);

    // Go back to Monitoring step in DB creation wizard
    await test.step('Resources step', async () => {
      // Move to Resources step
      await moveForward(page);
    });

    await test.step('Backup Schedules step', async () => {
      // Move to "Scheduled Backups" step
      await moveForward(page);
    });

    await test.step('Advanced Configuration step', async () => {
      // Move to "Advanced Configuration" step
      await moveForward(page);
    });

    await test.step('Monitoring step', async () => {
      // Go to "Monitoring" step
      await moveForward(page);
    });

    await test.step('Check DB Summary', async () => {
      await dbSummaryBasicInformationCheckForPG(
        page,
        namespace,
        recommendedEngineVersions,
        clusterName
      );
      await dbSummaryResourcesCheckForPG(page);
      await dbSummaryBackupsCheckForPG(page);
      await dbSummaryAdvancedConfigurationCheckForPG(page);
      await dbSummaryMonitoringCheck(page);
    });

    await test.step('Submit wizard', async () => {
      await submitWizard(page);
    });

    try {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      let addedCluster: any;
      await test.step('Wait for DB cluster creation', async () => {
        await expect(async () => {
          // new DB cluster appears in response not immediately
          addedCluster = await getDbClusterAPI(
            clusterName,
            namespace,
            request,
            token
          );
          expect(addedCluster).toBeDefined();
        }).toPass({
          intervals: [1000],
          timeout: TIMEOUTS.TenSeconds,
        });
      });

      expect(addedCluster?.spec.engine.type).toBe('postgresql');
      expect(addedCluster?.spec.engine.replicas).toBe(1);
      expect(addedCluster?.spec.engine.resources?.cpu.toString()).toBe('600m');
      expect(addedCluster?.spec.engine.resources?.memory.toString()).toBe('1G');
      expect(addedCluster?.spec.engine.storage.size.toString()).toBe('1Gi');

      expect(addedCluster?.spec.proxy.expose.type).toBe('ClusterIP');
      expect(addedCluster?.spec.proxy.replicas).toBe(1);
      expect(addedCluster?.spec.proxy.resources.cpu).toBe('1');
      expect(addedCluster?.spec.proxy.resources.memory).toBe('30M');

      // expect(addedCluster?.spec.proxy.expose.ipSourceRanges).toEqual([
      //   '192.168.1.1/24',
      //   '192.168.1.0',
      // ]);
      expect(addedCluster?.spec.backup.schedules[0].retentionCopies).toBe(1);
      // Verify timezone conversion was applied to the schedule cron
      // Day 10, 1h05 in IST timezone is day 9, 19h35 UTC
      expect(addedCluster?.spec.backup.schedules[0].schedule).toBe(
        '35 19 9 * *'
      );
    } finally {
      await deleteDbClusterFn(request, clusterName, namespace);
    }
  });

  test('Cancel wizard', async ({ page }) => {
    const clusterName = limitedSuffixedName(testPrefix + '-crt-psm');

    await test.step('Start DB cluster creation wizard', async () => {
      await selectDbEngine(page, 'postgresql');
    });

    await test.step('Basic Info step', async () => {
      await basicInformationStepCheckForPG(
        page,
        namespace,
        availableEngineVersions,
        recommendedEngineVersions,
        clusterName
      );
    });

    await test.step('Resources step', async () => {
      // Move to Resources step
      await moveForward(page);
      await resourcesStepCheckForPG(page);
    });

    await test.step('Backup Schedules step', async () => {
      // Move to "Scheduled Backups" step
      await moveForward(page);
      // skip this step
    });

    await test.step('Advanced Configuration step', async () => {
      // Move to "Advanced Configuration" step
      await moveForward(page);
      await advancedConfigurationStepCheckForPG(page);
    });

    await test.step('Monitoring step', async () => {
      // Go to "Monitoring" step
      await moveForward(page);
      await monitoringStepCheck(page);
    });

    await test.step('Cancel wizard', async () => {
      await cancelWizard(page);
    });

    await expect(page).toHaveURL('/databases');
  });

  // TODO uncomment when DB Cluster PATCH is available
  test.skip('Multiple Mongo schedules', async ({ page, request }) => {
    test.slow();
    const clusterName = 'multi-schedule-test';
    const recommendedEngineVersions = await getEnginesLatestRecommendedVersions(
      namespace,
      request
    );

    await basicInformationStepCheckForPG(
      page,
      namespace,
      availableEngineVersions,
      recommendedEngineVersions,
      clusterName
    );
    await moveForward(page);
    await resourcesStepCheckForPG(page);
    await moveForward(page);
    await backupsStepCheckForPG(page);
    await page
      .getByTestId('switch-input-pitr-enabled-label')
      .getByRole('checkbox')
      .check();
    await moveForward(page);
    await advancedConfigurationStepCheckForPG(page);
    await moveForward(page);
    await submitWizard(page);

    await page.goto('/databases');
    await waitForInitializingState(page, clusterName);
    await page.goto(`/databases/${namespace}/${clusterName}/backups`);
    await page.getByTestId('menu-button').click();
    await page.getByTestId('schedule-menu-item').click();
    await page.getByTestId('form-dialog-create').click();
    await expect(page.getByText('2 active schedules')).toBeVisible();

    // We disable PITR
    await page.goto('/databases');
    await waitForInitializingState(page, clusterName);
    await findDbAndClickActions(page, clusterName, 'Edit');
    await goToStep(page, 'backups');
    await setPitrEnabledStatus(page, false);
    await goToLastAndSubmit(page);

    // We turn it back on to make sure nothing breaks
    await page.goto('/databases');
    await findDbAndClickActions(page, clusterName, 'Edit');
    await goToStep(page, 'backups');
    await setPitrEnabledStatus(page, true);

    await deleteDbClusterFn(request, clusterName, namespace);
  });

  test('Reset schedules, PITR and monitoring when changing namespace', async ({
    page,
  }) => {
    const clusterName = limitedSuffixedName(testPrefix + '-crt-psm');

    await test.step('Start DB cluster creation wizard', async () => {
      await selectDbEngine(page, 'postgresql');
    });

    await test.step('Basic Info step', async () => {
      await basicInformationStepCheckForPG(
        page,
        namespace,
        availableEngineVersions,
        recommendedEngineVersions,
        clusterName
      );
    });

    await test.step('Resources step', async () => {
      // Move to Resources step
      await moveForward(page);
      await resourcesStepCheckForPG(page);
    });

    await test.step('Backup Schedules step', async () => {
      // Move to "Scheduled Backups" step
      await moveForward(page);
      await backupsStepCheckForPG(page);
    });

    await test.step('Advanced Configuration step', async () => {
      // Move to "Advanced Configuration" step
      await moveForward(page);
      await advancedConfigurationStepCheckForPG(page);
    });

    await test.step('Monitoring step', async () => {
      // Go to "Monitoring" step
      await moveForward(page);
      await monitoringStepCheck(page);
    });

    await test.step('Change Namespace', async () => {
      await goToStep(page, 'basic-information');
      await basicInformationSelectNamespaceCheck(
        page,
        EVEREST_CI_NAMESPACES.EVEREST_UI
      );

      // Check "Monitoring" panel in "Database Summary" section
      const monitoringInfo = page.getByTestId('section-monitoring');
      const monitoringPreviewContents =
        monitoringInfo.getByTestId('preview-content');
      await expect(
        monitoringPreviewContents.getByText('Disabled')
      ).toBeVisible();

      // Check "Backup" panel in "Database Summary" section
      const backupInfo = page.getByTestId('section-backups');
      await expect(
        backupInfo
          .getByTestId('empty-backups-preview-content')
          .getByText('Backups disabled')
      ).toBeVisible();
      await expect(
        backupInfo.getByTestId('preview-content').getByText('PITR disabled')
      ).toBeVisible();
    });
  });
});
