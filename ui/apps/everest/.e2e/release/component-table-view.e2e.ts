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

/*
 * Verifies DB components in the "Components → Table View".
 *
 * Checks:
 * - Pods and proxies are listed with correct Status, Ready, Name, Type, Age, and Restarts columns.
 * - Each pod/proxy row expands to show containers in a detail panel.
 * - Containers in detail panels have expected names, are Running, Ready, and have valid restart counts.
 *
 * Preconditions:
 * - The test navigates to the cluster details page.
 * - The "Components" tab and "Table view" are selected by the test.
 * - Database clusters are created as part of the test flow.
 */

import { expect, test, Page } from '@playwright/test';
import {
  findDbAndClickRow,
  deleteDbCluster,
} from '@e2e/utils/db-clusters-list';
import {
  moveForward,
  submitWizard,
  populateBasicInformation,
  populateResources,
  populateAdvancedConfig,
} from '@e2e/utils/db-wizard';
import { clickAddDbClusterBtn } from '@e2e/pr/db-cluster/db-wizard/db-wizard-utils';
import { EVEREST_CI_NAMESPACES } from '@e2e/constants';
import { waitForStatus, waitForDelete } from '@e2e/utils/table';
import { getCITokenFromLocalStorage } from '@e2e/utils/localStorage';
import { getClusterDetailedInfo } from '@e2e/utils/storage-class';
import { shouldExecuteDBCombination } from '@e2e/utils/generic';

let token: string;

test.describe.configure({ retries: 0 });

type Expectations = {
  /** text in Type column */
  expectedType: string;
  /** prefix used to match Name column (pod suffix is variable) */
  nameStartsWith: string;
  /** names of containers we expect to see in expanded view */
  containers: string[];
  /** type of proxy */
  expectedTypeProxy: string;
  /** names of containers we expect to see in expanded view for proxy */
  containersProxy: string[];
};

async function verifyComponentsForDb(
  page: Page,
  {
    db,
    clusterName,
    size,
  }: {
    db: string;
    clusterName: string;
    size: number;
  }
) {
  const defaultsByDb: Record<string, (cluster: string) => Expectations> = {
    postgresql: (cluster) => ({
      expectedType: 'pg',
      nameStartsWith: `${cluster}-instance1`,
      containers: [
        'database',
        'pgbackrest',
        'pgbackrest-config',
        'replication-cert-copy',
      ],
      expectedTypeProxy: 'pgbouncer',
      containersProxy: ['pgbouncer', 'pgbouncer-config'],
    }),
    pxc: (cluster) => ({
      expectedType: 'pxc',
      nameStartsWith: `${cluster}-pxc-`,
      containers: ['pxc'],
      expectedTypeProxy: 'haproxy',
      containersProxy: ['haproxy', 'pxc-monit'],
    }),
    psmdb: (cluster) => ({
      expectedType: 'mongod',
      nameStartsWith: `${cluster}-rs0-`,
      containers: ['backup-agent', 'mongod'],
      expectedTypeProxy: '',
      containersProxy: [],
    }),
  };
  const types = ['db', 'proxy'];
  const exp: Expectations = defaultsByDb[db](clusterName);

  const table = page.getByTestId(`${clusterName}-components`);
  await expect(table).toBeVisible();

  const expandAllButton = table.getByRole('button', { name: 'Expand all' });
  await expect(expandAllButton).toBeVisible();
  await expandAllButton.click();

  // we use this check also to wait until all rows are expanded
  const expNumberRunning =
    size +
    (size * exp.containers.length +
      (exp.containersProxy.length > 0
        ? size + size * exp.containersProxy.length
        : 0));
  const numberRunning = await table
    .getByText('Running', { exact: true })
    .count();
  expect(numberRunning).toEqual(expNumberRunning);

  // Only pick real data rows (skip detail-panel spacer rows)
  const rows = table.locator('tbody tr:not(.Mui-TableBodyCell-DetailPanel)');

  for (const type of types) {
    if (type === 'proxy' && !exp.expectedTypeProxy) {
      // Skip proxy checks if DB does not have proxy
      continue;
    }

    const startOfName =
      type === 'db'
        ? exp.nameStartsWith
        : `${clusterName}-${exp.expectedTypeProxy}-`;

    // Filter rows where Name (3rd cell) starts with the desired prefix
    const targetRows = rows.filter({
      has: page.locator(`td:nth-child(3):text-matches("^${startOfName}")`),
    });

    const count = await targetRows.count();
    expect(
      count,
      `Expected ${size} rows for ${db} with name starting '${exp.nameStartsWith}', got ${count}`
    ).toEqual(size);

    for (let i = 0; i < count; i++) {
      const row = targetRows.nth(i);

      const statusCell = row.locator('td').nth(0);
      const readyCell = row.locator('td').nth(1);
      const nameCell = row.locator('td').nth(2);
      const typeCell = row.locator('td').nth(3);
      //const ageCell      = row.locator('td').nth(4);
      const restartsCell = row.locator('td').nth(5);

      // Status
      await expect(statusCell.getByTestId('status')).toContainText('Running');

      // Ready
      const expReady =
        type === 'db'
          ? exp.containers.length + '/' + exp.containers.length
          : exp.containersProxy.length + '/' + exp.containersProxy.length;
      await expect(readyCell.getByTestId('component-ready-status')).toHaveText(
        expReady
      );

      // Name prefix (pod suffix is variable)
      await expect(nameCell).toHaveText(new RegExp(`^${startOfName}`));

      // Type
      const expType = type === 'db' ? exp.expectedType : exp.expectedTypeProxy;
      await expect(typeCell).toHaveText(expType);

      // Age: non-empty
      // Age only shows after 1 minute and you need to reload page so for now we will not wait for this
      // await expect(ageCell).not.toBeEmpty();

      // Restarts
      const restartCount = parseInt(
        (await restartsCell.textContent())?.trim() ?? '',
        10
      );
      expect(Number.isNaN(restartCount)).toBeFalsy();
      expect(restartCount).toBeGreaterThanOrEqual(0);
      expect(restartCount).toBeLessThanOrEqual(3);

      // === Detail row checks ===
      // The detail row is immediately after this main row in the DOM.
      const detailRow = row.locator(
        'xpath=following-sibling::tr[1][contains(@class,"Mui-TableBodyCell-DetailPanel")]'
      );
      const containerRows = detailRow.locator('tbody tr');

      const containerCount = await containerRows.count();
      const expContainerCount =
        type === 'db' ? exp.containers.length : exp.containersProxy.length;
      expect(
        containerCount,
        'Expected at least one container row in detail panel'
      ).toEqual(expContainerCount);

      for (let j = 0; j < expContainerCount; j++) {
        const containerRow = containerRows.nth(j);
        const containerStatus = containerRow.locator('td').nth(0);
        const containerReady = containerRow.locator('td').nth(1);
        const containerName = containerRow.locator('td').nth(2);
        const containerType = containerRow.locator('td').nth(3);
        const containerRestarts = containerRow.locator('td').nth(5);

        await expect(containerStatus).toContainText('Running');
        await expect(containerReady).toHaveText('true');
        const expContainerNames =
          type === 'db' ? exp.containers : exp.containersProxy;
        const text = await containerName.textContent();
        expect(expContainerNames).toContain(text?.trim());
        await expect(containerType).toBeEmpty();
        const restartCount = parseInt(
          await containerRestarts.textContent(),
          10
        );
        expect(Number.isNaN(restartCount)).toBeFalsy();
        expect(restartCount).toBeGreaterThanOrEqual(0);
        expect(restartCount).toBeLessThanOrEqual(3);
      }
    }
  }
}

[
  { db: 'psmdb', size: 3 },
  { db: 'pxc', size: 3 },
  { db: 'postgresql', size: 3 },
].forEach(({ db, size }) => {
  test.describe(`DB Table View [${db} size ${size}]`, () => {
    test.skip(!shouldExecuteDBCombination(db, size));
    test.describe.configure({ timeout: 720000 });

    const clusterName = `${db}-${size}-view`;
    const namespace = EVEREST_CI_NAMESPACES.EVEREST_UI;
    let storageClasses: string[] = [];

    test.beforeAll(async ({ request }) => {
      token = await getCITokenFromLocalStorage();
      const { storageClassNames = [] } = await getClusterDetailedInfo(
        token,
        request
      );
      storageClasses = storageClassNames;
    });

    test(`Create DB cluster [${db} size ${size}]`, async ({ page }) => {
      expect(storageClasses.length).toBeGreaterThan(0);

      await page.goto('/databases');
      await clickAddDbClusterBtn(page, db);

      await test.step('Populate basic information', async () => {
        await populateBasicInformation(
          page,
          namespace,
          clusterName,
          db,
          storageClasses[0],
          false,
          null
        );
        await moveForward(page);
      });

      await test.step('Populate resources', async () => {
        await page.getByRole('button').getByText('3 node').click();
        await expect(page.getByText('Nodes (3)')).toBeVisible();
        await populateResources(page, 0.6, 1, 1, 3);
        await moveForward(page);
      });

      await test.step('Populate backups', async () => {
        await moveForward(page);
      });

      await test.step('Populate advanced db config', async () => {
        await populateAdvancedConfig(page, db, false, '', true, '');
        await moveForward(page);
      });

      await test.step('Submit wizard', async () => {
        await submitWizard(page);
      });

      await test.step('Wait for cluster to be Up', async () => {
        await page.goto('/databases');
        await waitForStatus(page, clusterName, 'Initializing', 30000);
        await waitForStatus(page, clusterName, 'Up', 660000);
      });
    });

    test('Test headers contain proper columns', async ({ page }) => {
      await page.goto('/databases');
      await findDbAndClickRow(page, clusterName);
      await page.getByTestId('components').click();
      await page.getByTestId('switch-input-table-view').click();

      const table = page.getByTestId(`${clusterName}-components`);
      await expect(table).toBeVisible({ timeout: 5000 });

      // --- 1) Headers ---
      // Assert we have exactly 7 column headers (including the expand column)
      await expect(table.getByRole('columnheader')).toHaveCount(7);

      // Assert presence of required headers by accessible name
      for (const name of [
        'Status',
        'Ready',
        'Name',
        'Type',
        'Age',
        'Restarts',
      ]) {
        await expect(table.getByRole('columnheader', { name })).toBeVisible();
      }
    });

    test('Test pods and containers have expected values', async ({ page }) => {
      await page.goto('/databases');
      await findDbAndClickRow(page, clusterName);
      await page.getByTestId('components').click();
      await page.getByTestId('switch-input-table-view').click();

      await verifyComponentsForDb(page, {
        db: `${db}`,
        clusterName: `${clusterName}`,
        size: size,
      });
    });

    test(`Delete cluster [${db} size ${size}]`, async ({ page }) => {
      await deleteDbCluster(page, clusterName);
      await waitForStatus(page, clusterName, 'Deleting', 15000);
      await waitForDelete(page, clusterName, 240000);
    });
  });
});
