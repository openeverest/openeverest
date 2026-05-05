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
  getEnginesLatestRecommendedVersions,
  getEnginesVersions,
} from '@e2e/utils/database-engines';
import { createDbClusterFn, deleteDbClusterFn } from '@e2e/utils/db-cluster';
import { getTokenFromLocalStorage } from '@e2e/utils/localStorage';
import { advancedConfigurationStepCheck } from './steps/advanced-configuration-step';
import { backupsStepCheck } from './steps/backups-step';
import {
  basicInformationStepCheck,
  DEFAULT_CLUSTER_VERSION,
} from './steps/basic-information-step';
import { resourcesStepCheck } from './steps/resources-step';
import {
  goToLastAndSubmit,
  goToStep,
  moveBack,
  moveForward,
  setPitrEnabledStatus,
  submitWizard,
} from '@e2e/utils/db-wizard';
import {
  addFirstScheduleInDBWizard,
  checkAmountOfDbEngines,
  fillScheduleModalForm,
  openCreateScheduleDialogFromDBWizard,
  selectDbEngine,
} from '../db-wizard-utils';
import { findDbAndClickActions } from '@e2e/utils/db-clusters-list';
import { waitForInitializingState } from '@e2e/utils/table';
import { EVEREST_CI_NAMESPACES } from '@e2e/constants';

test.describe('DB Cluster creation', () => {
  // IST is UTC+5h30, with or without DST
  test.use({
    timezoneId: 'IST',
  });

  let engineVersions = {
    pxc: [],
    psmdb: [],
    postgresql: [],
  };
  // let monitoringInstancesList = [];
  const namespace = EVEREST_CI_NAMESPACES.EVEREST_UI;

  test.beforeAll(async ({ request }) => {
    const token = await getTokenFromLocalStorage();
    engineVersions = await getEnginesVersions(token, namespace, request);
  });

  test.beforeEach(async ({ page }) => {
    await page.goto('/databases');
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
      await dbEnginesButtons.nth(i).click();

      await page.getByTestId('select-input-db-version').waitFor();
      expect(
        await page.getByTestId('select-input-db-version').inputValue()
      ).toBeDefined();

      await moveForward(page);

      expect(
        await page.getByTestId(`toggle-button-nodes-${expectedNodesOrder[i]}`)
      ).toHaveAttribute('aria-pressed', 'true');

      // We click on the first button to make sure it always goes back to defaults afterwards
      await page.getByTestId('toggle-button-nodes-1').click();

      // We return to databases page to choose other db
      await page.goto('/databases');
      await page.getByTestId('add-db-cluster-button').waitFor();
      await page.getByTestId('add-db-cluster-button').click();
    }
  });

  test('Cluster creation', async ({ page, request }) => {
    const clusterName = 'db-cluster-ui-test';
    const recommendedEngineVersions = await getEnginesLatestRecommendedVersions(
      namespace,
      request
    );

    await selectDbEngine(page, 'psmdb');

    await basicInformationStepCheck(
      page,
      engineVersions,
      recommendedEngineVersions,
      clusterName
    );

    const dbName = await page.getByTestId('text-input-db-name').inputValue();

    await moveForward(page);
    await expect(page.getByText('3 nodes - CPU')).toBeVisible();

    await resourcesStepCheck(page);

    // Sharding off, no routers available
    await expect(page.getByText('Routers (3)')).not.toBeVisible();

    await moveBack(page);
    await page
      .getByTestId('switch-input-sharding-label')
      .getByRole('checkbox')
      .check();
    await moveForward(page);

    await expect(page.getByText('Routers (3)')).toBeVisible();
    await page.getByTestId('proxies-accordion').getByRole('button').click();
    await page.getByTestId('toggle-button-routers-1').click();
    await expect(page.getByText('Routers (1)')).toBeVisible();
    await page.getByTestId('nodes-accordion').getByRole('button').click();
    await page.getByTestId('toggle-button-nodes-1').click();
    await page.getByTestId('toggle-button-nodes-3').click();
    // After used changed the number of routers, it should no more follow the number of nodes
    await expect(page.getByText('Routers (1)')).toBeVisible();

    await moveForward(page);

    await backupsStepCheck(page);

    await moveForward(page);

    await advancedConfigurationStepCheck(page);
    await moveForward(page);

    // Test the mechanism for default number of nodes
    await page.getByTestId('button-edit-preview-basic-information').click();
    // Here we test that version wasn't reset to default
    await expect(
      page.getByText(`Version: ${DEFAULT_CLUSTER_VERSION}`)
    ).toBeVisible();

    // Make sure name doesn't change when we go back to first step
    expect(await page.getByTestId('text-input-db-name').inputValue()).toBe(
      dbName
    );

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

    await page.getByTestId('button-edit-preview-monitoring').click();

    // await monitoringStepCheck(page, monitoringInstancesList);
    await submitWizard(page);

    const response = await request.get(
      `/v1/namespaces/${namespace}/database-clusters`,
      {
        headers: {
          Authorization: `Bearer ${await getTokenFromLocalStorage()}`,
        },
      }
    );

    expect(response.ok()).toBeTruthy();
    // TODO replace with correct payload typings from GET DB Clusters
    const { items: clusters } = await response.json();

    const addedCluster = clusters.find(
      (cluster) => cluster.metadata.name === clusterName
    );
    await deleteDbClusterFn(request, addedCluster?.metadata.name, namespace);
    //TODO: Add check for PITR ones backend is ready

    expect(addedCluster).not.toBeUndefined();
    expect(addedCluster?.spec.engine.type).toBe('psmdb');
    expect(addedCluster?.spec.engine.replicas).toBe(3);
    expect(addedCluster?.spec.engine.resources?.cpu.toString()).toBe('600m');
    expect(addedCluster?.spec.engine.resources?.memory.toString()).toBe('1G');
    expect(addedCluster?.spec.engine.storage.size.toString()).toBe('1Gi');
    expect(addedCluster?.spec.proxy.expose.type).toBe('ClusterIP');
    // TODO commented, because we use only psmdb in this test
    // expect(addedCluster?.spec.proxy.replicas).toBe(1);
    // expect(addedCluster?.spec.proxy.resources.cpu).toBe('1');
    // expect(addedCluster?.spec.proxy.resources.memory).toBe('2G');

    // expect(addedCluster?.spec.proxy.expose.ipSourceRanges).toEqual([
    //   '192.168.1.1/24',
    //   '192.168.1.0',
    // ]);
    expect(addedCluster?.spec.backup.schedules[0].retentionCopies).toBe(1);
    // Verify timezone conversion was applied to the schedule cron
    // Day 10, 1h05 in IST timezone is day 9, 19h35 UTC
    expect(addedCluster?.spec.backup.schedules[0].schedule).toBe('35 19 9 * *');
  });

  test('PITR should be disabled when backups has no schedules checked', async ({
    page,
  }) => {
    await selectDbEngine(page, 'pxc');
    // go to resources page
    await moveForward(page);
    // go to backups page
    await moveForward(page);
    await expect(
      page.getByText('You currently do not have any backup schedules set up.')
    ).toBeVisible();
    const enabledPitrCheckbox = page
      .getByTestId('switch-input-pitr-enabled-label')
      .getByRole('checkbox');

    await expect(enabledPitrCheckbox).not.toBeChecked();
    await expect(enabledPitrCheckbox).toBeDisabled();
    await addFirstScheduleInDBWizard(page, 'testFirst');
    await expect(enabledPitrCheckbox).not.toBeChecked();
    await expect(enabledPitrCheckbox).not.toBeDisabled();
    await enabledPitrCheckbox.setChecked(true);
    await expect(
      page.getByTestId('text-input-pitr-storage-location')
    ).toBeVisible();
  });

  test.skip('Cancel wizard', async ({ page }) => {
    await page.getByTestId('mongodb-toggle-button').click();
    await page.getByTestId('text-input-db-name').fill('new-cluster');
    await page.getByRole('option').first().click();
    await moveForward(page);

    await expect(
      page.getByRole('heading', {
        name: 'Configure the resources your new database will have access to.',
      })
    ).toBeVisible();

    await page.getByTestId('toggle-button-nodes-3').click();
    await page.getByTestId('node-resources-toggle-button-large').click();
    await page.getByTestId('text-input-disk').fill('150');
    await moveForward(page);

    // await expect(
    //   page.getByRole('heading', {
    //     name: 'Specify how often you want to run backup jobs for your database.',
    //   })
    // ).toBeVisible();
    //
    // await page.getByTestId('text-input-storage-location').click();
    //
    // const storageOptions = page.getByRole('option');
    //
    // expect(storageOptions.filter({ hasText: 'ui-dev' })).toBeVisible();
    // await storageOptions.first().click();
    //
    // await await moveForward(page);

    await expect(
      page.getByRole('heading', { name: 'Advanced Configurations' })
    ).toBeVisible();

    await page.getByTestId('db-wizard-cancel-button').click();
    await expect(page.getByRole('dialog')).toBeVisible();
    await page.getByText('Yes, cancel').click();

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

    await basicInformationStepCheck(
      page,
      engineVersions,
      recommendedEngineVersions,
      clusterName
    );
    await moveForward(page);
    await resourcesStepCheck(page);
    await moveForward(page);
    await backupsStepCheck(page);
    await page
      .getByTestId('switch-input-pitr-enabled-label')
      .getByRole('checkbox')
      .check();
    await moveForward(page);
    await advancedConfigurationStepCheck(page);
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

  test('Warning should appears for schedule with the same date', async ({
    page,
  }) => {
    await selectDbEngine(page, 'psmdb');

    // Resources Step
    await moveForward(page);
    // Backups step
    await moveForward(page);

    await addFirstScheduleInDBWizard(page, 'testFirst');
    await openCreateScheduleDialogFromDBWizard(page);
    await expect(page.getByTestId('same-schedule-warning')).not.toBeVisible();
    await fillScheduleModalForm(page, undefined, '1', undefined, undefined);
    await expect(page.getByTestId('same-schedule-warning')).toBeVisible();
  });

  test('Reset schedules, PITR and monitoring when changing namespace', async ({
    page,
  }) => {
    await selectDbEngine(page, 'psmdb');
    await moveForward(page);
    await moveForward(page);
    await addFirstScheduleInDBWizard(page, 'testFirst');
    await page
      .getByTestId('switch-input-pitr-enabled-label')
      .getByRole('checkbox')
      .check();
    await moveForward(page);
    await moveForward(page);
    await page
      .getByTestId('switch-input-monitoring-label')
      .getByRole('checkbox')
      .check();

    const monitoringPreviewContent = page
      .getByTestId('section-monitoring')
      .getByTestId('preview-content');

    expect(await monitoringPreviewContent.textContent()).toBe('Enabled');
    await expect(page.getByText('PITR Enabled')).toBeVisible();
    await expect(page.getByText('Backups disabled')).not.toBeVisible();

    await goToStep(page, 'basic-information');
    await page.getByTestId('text-input-k8s-namespace').click();

    const namespacesOptions = page.getByRole('option');
    await namespacesOptions.nth(1).click();
    expect(await monitoringPreviewContent.textContent()).toBe('Disabled');
    await expect(page.getByText('Backups disabled')).toBeVisible();
    await expect(page.getByText('PITR disabled')).toBeVisible();
  });

  test('Duplicate name should throw an error', async ({ page, request }) => {
    await createDbClusterFn(
      request,
      {
        dbName: 'mysql-1',
        dbType: 'mysql',

        numberOfNodes: '1',
        backup: {
          enabled: false,
          schedules: [],
        },
      },
      'pxc-only'
    );
    await selectDbEngine(page, 'pxc');

    const nameInput = page.getByTestId('text-input-db-name');
    await page.getByTestId('k8s-namespace-autocomplete').click();
    await page.getByRole('option', { name: 'pxc-only' }).click();
    await nameInput.fill('mysql-1');
    await expect(
      page.getByText('You already have a database with the same name.')
    ).toBeVisible();

    await deleteDbClusterFn(request, 'mysql-1', 'pxc-only');
  });
});
