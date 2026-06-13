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

import { EVEREST_CI_NAMESPACES } from '@e2e/constants';
import { getEnginesVersions } from '@e2e/utils/database-engines';
import { getTokenFromLocalStorage } from '@e2e/utils/localStorage';
import { getClusterDetailedInfo } from '@e2e/utils/storage-class';
import { expect, test } from '@playwright/test';
import { selectDbEngine } from '../db-wizard-utils';
import {
  goToLastStepByStepAndSubmit,
  moveBack,
  moveForward,
} from '@e2e/utils/db-wizard';
import {
  deleteDbCluster,
  findDbAndClickRow,
} from '@e2e/utils/db-clusters-list';
import { waitForDelete } from '@e2e/utils/table';
import { execSync } from 'child_process';

export const patchPSMDBFinalizers = async (
  cluster: string,
  namespace: string
) => {
  try {
    const command = `kubectl patch --namespace ${namespace} psmdb ${cluster} --type='merge' -p '{"metadata":{"finalizers":["percona.com/delete-psmdb-pvc"]}}'`;
    const output = execSync(command).toString();
    return true;
  } catch (error) {
    console.error(`Error executing command: ${error}`);
    throw error;
  }
};

test.describe('Sharding (psmdb)', () => {
  let engineVersions = {
    pxc: [],
    psmdb: [],
    postgresql: [],
  };

  let storageClasses = [];
  const namespace = EVEREST_CI_NAMESPACES.EVEREST_UI;

  test.beforeAll(async ({ request }) => {
    const token = await getTokenFromLocalStorage();
    engineVersions = await getEnginesVersions(token, namespace, request);

    const { storageClassNames = [] } = await getClusterDetailedInfo(
      token,
      request
    );
    storageClasses = storageClassNames;
  });

  test.beforeEach(async ({ page }) => {
    await page.goto('/databases');
  });

  test('Sharding toggle is presented for MongoDb during creation of database', async ({
    page,
  }) => {
    expect(storageClasses.length).toBeGreaterThan(0);
    await selectDbEngine(page, 'psmdb');

    const shardingToggle = page.getByTestId('switch-input-sharding');
    await shardingToggle.waitFor();

    expect(shardingToggle).toBeVisible();
    expect(shardingToggle).toHaveValue('false');
  });

  test('Config servers and Nº of shards is presented if sharding is enabled', async ({
    page,
  }) => {
    expect(storageClasses.length).toBeGreaterThan(0);
    await selectDbEngine(page, 'psmdb');

    const shardingToggle = page.getByTestId('switch-input-sharding');
    await shardingToggle.waitFor();

    const basicSectionSummury = page.getByTestId('section-basic-information');
    expect(basicSectionSummury.getByText('Sharding: disabled')).toBeVisible();
    await shardingToggle.click();
    expect(basicSectionSummury.getByText('Sharding: enabled')).toBeVisible();

    await moveForward(page);

    const resourcesSectionSummury = page.getByTestId('section-resources');
    expect(resourcesSectionSummury.getByText('2 shards')).toBeVisible();
    expect(
      resourcesSectionSummury.getByText('3 configuration servers')
    ).toBeVisible();

    const shardNr = page.getByTestId('text-input-shard-nr');
    expect(shardNr).toBeVisible();
    expect(shardNr).toHaveValue('2');

    const configServersNr = page.getByTestId(
      'toggle-button-group-input-shard-config-servers'
    );
    const configServers3Btn = page.getByTestId('shard-config-servers-3');

    expect(configServersNr).toBeVisible();
    expect(configServers3Btn).toHaveAttribute('aria-pressed', 'true');

    await moveBack(page);
    await shardingToggle.click();
    await moveForward(page);

    expect(shardNr).not.toBeVisible();
    expect(configServersNr).not.toBeVisible();
    expect(resourcesSectionSummury.getByText('2 shards')).not.toBeVisible();
    expect(
      resourcesSectionSummury.getByText('Configuration servers: 3')
    ).not.toBeVisible();
  });

  test.skip('Sharding should be correctly displayed on the overview page', async ({
    page,
  }) => {
    test.setTimeout(120 * 1000);
    const dbName = 'sharding-psmdb';
    expect(storageClasses.length).toBeGreaterThan(0);
    await selectDbEngine(page, 'psmdb');

    await page.getByTestId('switch-input-sharding').click();
    await page.getByTestId('text-input-db-name').fill(dbName);
    await goToLastStepByStepAndSubmit(page);
    await page.goto('/databases');

    await findDbAndClickRow(page, dbName);
    expect(
      page
        .getByTestId('sharding-status-overview-section-row')
        .filter({ hasText: 'Enabled' })
    ).toBeVisible();
    expect(
      page
        .getByTestId('number-of-shards-overview-section-row')
        .filter({ hasText: '2' })
    ).toBeVisible();
    expect(
      page
        .getByTestId('config-servers-overview-section-row')
        .filter({ hasText: '3' })
    ).toBeVisible();

    await deleteDbCluster(page, dbName);
    // TODO: This function should be removed after fix for: https://perconadev.atlassian.net/browse/K8SPSMDB-1208
    await patchPSMDBFinalizers('sharding-psmdb', 'everest-ui');
    await waitForDelete(page, dbName, 60000);
  });

  test('Mongo with sharding should not pass multinode cluster creation if config servers = 1', async ({
    page,
  }) => {
    expect(storageClasses.length).toBeGreaterThan(0);
    await selectDbEngine(page, 'psmdb');

    await page.getByTestId('switch-input-sharding').click();
    await moveForward(page);

    await expect(page.getByTestId(`toggle-button-routers-3`)).toHaveAttribute(
      'aria-pressed',
      'true'
    );

    await expect(page.getByTestId('shard-config-servers-3')).toHaveAttribute(
      'aria-pressed',
      'true'
    );

    await page.getByTestId('shard-config-servers-1').click();
    await expect(page.getByTestId('shard-config-servers-error')).toBeVisible();

    await page.getByTestId('toggle-button-nodes-1').click();
    await expect(
      page.getByTestId('shard-config-servers-error')
    ).not.toBeVisible();
    await expect(
      page.getByTestId('db-wizard-continue-button')
    ).not.toBeDisabled();

    await page.getByTestId('toggle-button-nodes-3').click();
    await expect(page.getByTestId('shard-config-servers-error')).toBeVisible();

    await page.getByTestId('shard-config-servers-3').click();
    await expect(
      page.getByTestId('shard-config-servers-error')
    ).not.toBeVisible();
    await expect(
      page.getByTestId('db-wizard-continue-button')
    ).not.toBeDisabled();
  });

  test('0 value of the Nº of shards causes an error', async ({ page }) => {
    expect(storageClasses.length).toBeGreaterThan(0);
    await selectDbEngine(page, 'psmdb');

    await page.getByTestId('switch-input-sharding').click();
    expect(page.getByTestId('switch-input-sharding')).toBeEnabled();

    await moveForward(page);

    await page.getByTestId('text-input-shard-nr').fill('0');
    await expect(
      page.getByText('The value cannot be less than 1')
    ).toBeVisible();
    await page.getByTestId('text-input-shard-nr').fill('1');
    await expect(
      page.getByTestId('db-wizard-continue-button')
    ).not.toBeDisabled();
  });

  test('Changing the Nº of nodes causes changing the confing servers Nº automatically, until the user touches confing servers Nº', async ({
    page,
  }) => {
    expect(storageClasses.length).toBeGreaterThan(0);
    await selectDbEngine(page, 'psmdb');

    await page.getByTestId('switch-input-sharding').click();
    await expect(page.getByTestId('switch-input-sharding')).toBeEnabled();

    await moveForward(page);

    await expect(page.getByTestId(`toggle-button-routers-3`)).toHaveAttribute(
      'aria-pressed',
      'true'
    );
    await expect(page.getByTestId('shard-config-servers-3')).toHaveAttribute(
      'aria-pressed',
      'true'
    );

    await page.getByTestId('toggle-button-nodes-1').click();
    await expect(page.getByTestId('shard-config-servers-1')).toHaveAttribute(
      'aria-pressed',
      'true'
    );
    await page.getByTestId('toggle-button-nodes-5').click();
    await expect(page.getByTestId('shard-config-servers-5')).toHaveAttribute(
      'aria-pressed',
      'true'
    );
    await page.getByTestId('toggle-button-nodes-custom').click();
    await page.getByTestId('text-input-custom-nr-of-nodes').fill('7');
    await expect(page.getByTestId('shard-config-servers-7')).toHaveAttribute(
      'aria-pressed',
      'true'
    );
    await page.getByTestId('text-input-custom-nr-of-nodes').fill('9');
    await expect(page.getByTestId('shard-config-servers-7')).toHaveAttribute(
      'aria-pressed',
      'true'
    );

    await page.getByTestId('shard-config-servers-3').click();
    await page.getByTestId('toggle-button-nodes-1').click();
    await expect(page.getByTestId('shard-config-servers-1')).toHaveAttribute(
      'aria-pressed',
      'false'
    );
    await expect(page.getByTestId('shard-config-servers-3')).toHaveAttribute(
      'aria-pressed',
      'true'
    );
  });

  test('Sharding is not reset to default when returning to the previous step of the form in dbWizard', async ({
    page,
  }) => {
    expect(storageClasses.length).toBeGreaterThan(0);
    await selectDbEngine(page, 'psmdb');

    await page.getByTestId('switch-input-sharding').click();
    await expect(page.getByTestId('switch-input-sharding')).toBeEnabled();

    await moveForward(page);
    await moveBack(page);

    await expect(page.getByTestId('switch-input-sharding')).toBeEnabled();

    await moveForward(page);
    await moveForward(page);

    await page.getByTestId('edit-section-1').click();

    await page.getByTestId('switch-input-sharding').waitFor();
    await expect(page.getByTestId('switch-input-sharding')).toBeEnabled();
  });
});
