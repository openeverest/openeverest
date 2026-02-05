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
import { Messages } from '../../../src/modals/restore-db-modal/restore-db-modal.messages';
import { createDbClusterFn, deleteDbClusterFn } from '@e2e/utils/db-cluster';
import {
  findDbAndClickActions,
  findDbAndClickRow,
} from '@e2e/utils/db-clusters-list';
import { getBucketNamespacesMap } from '@e2e/constants';
import { goToStep, moveForward } from '@e2e/utils/db-wizard';

const dbClusterName = 'restore-to-new-cluster';

test.describe('DB Cluster Restore to the new cluster', () => {
  test.beforeAll(async ({ request }) => {
    await createDbClusterFn(request, {
      dbName: dbClusterName,
      dbType: 'mongodb',
      dbVersion: '8.0.4-1',
      numberOfNodes: '1',
      numberOfProxies: '3',
      cpu: '1',
      storageClass: 'my-storage-class',
      sharding: true,
      shards: 2,
      configServerReplicas: 3,
      backup: {
        enabled: true,
        schedules: [
          {
            backupStorageName: getBucketNamespacesMap()[0][0],
            enabled: true,
            name: 'backup-1',
            schedule: '0 * * * *',
          },
        ],
        pitr: {
          enabled: true,
          backupStorageName: 'minio',
        },
      },
      externalAccess: true,
      sourceRanges: [
        {
          sourceRange: '192.168.1.1/32',
        },
      ],
      monitoringConfigName: 'pmm',
    });
  });

  test.beforeEach(async ({ page }) => {
    await page.goto('/databases');
  });

  test.afterAll(async ({ request }) => {
    await deleteDbClusterFn(request, dbClusterName);
  });
  test('DB cluster list restore action', async ({ page }) => {
    await findDbAndClickActions(page, dbClusterName, 'Create DB from a backup');

    await expect(
      page
        .getByTestId('select-backup-name-button')
        .getByText(Messages.selectBackup)
    ).toBeVisible();
  });

  test('DB cluster detail restore action', async ({ page }) => {
    await page.route(
      '/v1/namespaces/**/database-clusters/**/backups',
      async (route) => {
        await route.fulfill({
          json: {
            items: [
              {
                metadata: {
                  name: 'backup-1',
                },
                spec: {
                  dbClusterName,
                  backupStorageName: getBucketNamespacesMap()[0][0],
                },
                status: {
                  state: 'Succeeded',
                  created: '2024-12-20T11:57:41Z',
                  completed: '2024-12-20T11:58:07Z',
                },
              },
            ],
          },
        });
      }
    );
    await findDbAndClickRow(page, dbClusterName);
    const actionButton = page.getByTestId('actions-button');
    await actionButton.click();

    const restoreButton = page.getByTestId(
      `${dbClusterName}-create-new-db-from-backup`
    );
    await restoreButton.click();

    await expect(
      page
        .getByTestId('select-backup-name-button')
        .getByText(Messages.selectBackup)
    ).toBeVisible();
    await page.getByTestId('select-backup-name-button').click();
    await page.getByText('backup-1').click();
    await page.getByText('Create', { exact: true }).click();
    await expect(
      page.getByText('Basic information', { exact: true })
    ).toBeVisible();
    await expect(page.getByTestId('select-input-db-version')).toBeDisabled();

    const comboboxes = page.getByRole('combobox');
    const dbVersionCombobox = comboboxes.nth(1);
    expect(await dbVersionCombobox.textContent()).toBe('8.0.4-1');
    await page.getByTestId('text-input-db-name').fill('new-db-cluster');
    await moveForward(page);

    await expect(page.getByTestId('text-input-shard-nr')).toHaveValue('2');
    await expect(
      page.getByText(
        '2 nodes - CPU - 2.00 CPU; Memory - 4.00 GB; Disk - 50.00 Gi'
      )
    ).toBeVisible();
    await expect(
      page.getByText('3 routers - CPU - 3.00 CPU; Memory - 3.00 GB')
    ).toBeVisible();
    await expect(page.getByTestId('toggle-button-nodes-1')).toHaveAttribute(
      'aria-pressed',
      'true'
    );
    await expect(
      page.getByTestId('node-resources-toggle-button-custom')
    ).toHaveAttribute('aria-pressed', 'true');
    await expect(page.getByTestId('text-input-cpu')).toHaveValue('1');
    await expect(page.getByTestId('text-input-memory')).toHaveValue('2');
    await expect(page.getByTestId('text-input-disk')).toHaveValue('25');
    await page.getByTestId('proxies-accordion').click();
    await expect(page.getByTestId('toggle-button-routers-3')).toHaveAttribute(
      'aria-pressed',
      'true'
    );
    await expect(page.getByTestId('text-input-proxy-cpu')).toHaveValue('1');
    await expect(page.getByTestId('text-input-proxy-memory')).toHaveValue('1');
    await expect(page.getByTestId('shard-config-servers-3')).toHaveAttribute(
      'aria-pressed',
      'true'
    );

    await moveForward(page);

    const schedule = page.getByTestId('editable-item');
    await expect(schedule).toHaveText(/.*Every hour at minute 0.*/);
    await expect(page.getByRole('checkbox')).toBeChecked();
    await expect(
      page.getByTestId('switch-input-pitr-enabled-label')
    ).toHaveText(new RegExp(`.*Storage: ${getBucketNamespacesMap()[0][0]}.*`));

    await moveForward(page);

    await expect(page.getByTestId('text-input-storage-class')).toHaveValue(
      'my-storage-class'
    );
    await expect(
      page.getByTestId('text-input-source-ranges.0.source-range')
    ).toHaveValue('192.168.1.1/32');

    await moveForward(page);

    await expect(page.getByRole('checkbox')).toBeChecked();
    await expect(page.getByTestId('db-wizard-submit-button')).toBeEnabled();
    await expect(page.getByTestId('preview-error-resources')).not.toBeVisible();
  });

  test('PG Cluster correct schedules restore', async ({ page, request }) => {
    const dbName = 'pg-cluster-restore';
    await createDbClusterFn(request, {
      dbName,
      dbType: 'postgresql',
      dbVersion: '15.13',
      numberOfNodes: '4',
      numberOfProxies: '1',
      storageClass: 'my-storage-class',
      backup: {
        enabled: true,
        schedules: [
          {
            backupStorageName: getBucketNamespacesMap()[0][0],
            enabled: true,
            name: 'backup-1',
            schedule: '0 * * * *',
          },
        ],
        pitr: {
          enabled: true,
          backupStorageName: 'minio',
        },
      },
      externalAccess: true,
      sourceRanges: [
        {
          sourceRange: '192.168.1.1/32',
        },
      ],
      monitoringConfigName: 'pmm',
    });
    await page.route(
      '/v1/namespaces/**/database-clusters/**/backups',
      async (route) => {
        await route.fulfill({
          json: {
            items: [
              {
                metadata: {
                  name: 'backup-1',
                },
                spec: {
                  dbName,
                  backupStorageName: getBucketNamespacesMap()[0][0],
                },
                status: {
                  state: 'Succeeded',
                  created: '2024-12-20T11:57:41Z',
                  completed: '2024-12-20T11:58:07Z',
                },
              },
            ],
          },
        });
      }
    );
    await findDbAndClickRow(page, dbName);
    const actionButton = page.getByTestId('actions-button');
    await actionButton.click();

    const restoreButton = page.getByTestId(
      `${dbName}-create-new-db-from-backup`
    );
    await restoreButton.click();

    await expect(
      page
        .getByTestId('select-backup-name-button')
        .getByText(Messages.selectBackup)
    ).toBeVisible();
    await page.getByTestId('select-backup-name-button').click();
    await page.getByText('backup-1').click();
    await page.getByText('Create', { exact: true }).click();
    await expect(page.getByTestId('select-input-db-version')).toBeDisabled();

    const comboboxes = page.getByRole('combobox');
    const dbVersionCombobox = comboboxes.nth(1);
    expect(await dbVersionCombobox.textContent()).toBe('15.13');

    await moveForward(page);

    await expect(
      page.getByText(
        '4 nodes - CPU - 4.00 CPU; Memory - 8.00 GB; Disk - 100.00 Gi'
      )
    ).toBeVisible();
    await expect(
      page.getByText('1 PG Bouncer - CPU - 1.00 CPU; Memory - 1.00 GB')
    ).toBeVisible();
    await expect(
      page.getByTestId('toggle-button-nodes-custom')
    ).toHaveAttribute('aria-pressed', 'true');
    await expect(page.getByTestId('text-input-custom-nr-of-nodes')).toHaveValue(
      '4'
    );
    await expect(
      page.getByTestId('node-resources-toggle-button-small')
    ).toHaveAttribute('aria-pressed', 'true');
    await page.getByTestId('proxies-accordion').click();
    await expect(
      page.getByTestId('toggle-button-PG Bouncers-1')
    ).toHaveAttribute('aria-pressed', 'true');
    await expect(page.getByTestId('text-input-proxy-cpu')).toHaveValue('1');
    await expect(page.getByTestId('text-input-proxy-memory')).toHaveValue('1');

    await moveForward(page);

    await expect(
      page.getByText('Backup schedules', { exact: true })
    ).toBeVisible();
    const schedule = page.getByTestId('editable-item');
    await expect(schedule).toHaveText(/.*Every hour at minute 0.*/);
    await expect(page.getByRole('checkbox')).toBeChecked();
    await expect(page.getByRole('checkbox')).toBeDisabled();

    await moveForward(page);
    await moveForward(page);
    await expect(page.getByTestId('db-wizard-submit-button')).toBeEnabled();
    await expect(page.getByTestId('preview-error-resources')).not.toBeVisible();

    await deleteDbClusterFn(request, dbName);
  });
});
