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
import { expect, Page, test } from '@playwright/test';
import { getTokenFromLocalStorage } from '@e2e/utils/localStorage';
import {
  deleteDbCluster,
  findDbAndClickRow,
} from '@e2e/utils/db-clusters-list';
import { getClusterDetailedInfo } from '@e2e/utils/storage-class';
import {
  moveForward,
  populateBasicInformation,
  submitWizard,
} from '@e2e/utils/db-wizard';
import { waitForInitializingState } from '@e2e/utils/table';
import { selectDbEngine } from '@e2e/pr/db-cluster/db-wizard/db-wizard-utils';
import {
  createDbClusterFn,
  deleteDbClusterFn,
  getDbClusterAPI,
} from '@e2e/utils/db-cluster';
import { getNamespacesFn } from '@e2e/utils/namespaces';

let token: string;
const openResourcesModal = async (page: Page) => {
  const editResourcesButton = page.getByTestId('edit-resources-button');
  await editResourcesButton.waitFor();
  await editResourcesButton.click();
  expect(page.getByTestId('edit-resources-form-dialog')).toBeVisible();
};

[
  { db: 'psmdb', size: 1 },
  { db: 'pxc', size: 1 },
  // { db: 'postgresql', size: 1 },
].forEach(
  ({ db, size }: { db: 'pxc' | 'psmdb' | 'postgresql'; size: number }) => {
    test.describe(`Overview page: ${db} resources editing`, () => {
      test.describe.configure({ timeout: 1000000 });

      const clusterName = `${db}-${size}-resources-edit`;

      let storageClasses = [];

      test.beforeAll(async ({ request }) => {
        token = await getTokenFromLocalStorage();

        const { storageClassNames = [] } = await getClusterDetailedInfo(
          token,
          request
        );
        storageClasses = storageClassNames;
      });

      test(`Creation and waiting of ready ${size} node ${db} ${db == 'psmdb' ? 'with sharding' : ''} for resources edit tests`, async ({
        page,
      }) => {
        expect(storageClasses.length).toBeGreaterThan(0);
        await page.goto('/databases');
        await selectDbEngine(page, db);

        await test.step('Populate basic info', async () => {
          await page.getByTestId('text-input-db-name').fill(clusterName);
          // sharding enabling
          if (db == 'psmdb') {
            await page.getByTestId('switch-input-sharding').click();
          }

          // go to resources page
          await moveForward(page);
        });

        await test.step('Populate resources', async () => {
          await page
            .getByRole('button')
            .getByText(size + ' node')
            .click();
          const numberOfNodes = size * (db !== 'psmdb' ? 1 : 2);
          await expect(
            page.getByText(
              numberOfNodes + ` node${numberOfNodes === 1 ? '' : 's'}:`
            )
          ).toBeVisible();
        });

        await test.step('Move forward form with default values', async () => {
          //go to backups page
          await moveForward(page);
          //go to advanced configuration
          await moveForward(page);
          //go to monitoring
          await moveForward(page);
        });

        await test.step('Submit form', async () => {
          await submitWizard(page);
        });
      });

      test(`Show the correct default values during editing of ${db}`, async ({
        page,
      }) => {
        await page.goto('/databases');

        await findDbAndClickRow(page, clusterName);
        await test.step('Open edit resource modal', async () => {
          openResourcesModal(page);
        });

        await test.step('Check default values', async () => {
          //TODO improve setting during creation the CPU, memory, etc.
          //TODO customize sharding number, number of config servers
          await expect(
            page.getByTestId(`toggle-button-nodes-${size}`)
          ).toHaveAttribute('aria-pressed', 'true');
          await expect(
            page.getByTestId('node-resources-toggle-button-small')
          ).toHaveAttribute('aria-pressed', 'true');

          await page.getByTestId('proxies-accordion').click();
          if (db != 'psmdb') {
            await expect(
              page.getByTestId(`toggle-button-proxies-${size}`)
            ).toHaveAttribute('aria-pressed', 'true');
          } else {
            // sharding
            await expect(
              page.getByTestId(`toggle-button-routers-${size}`)
            ).toHaveAttribute('aria-pressed', 'true');
            expect(
              await page.getByTestId('text-input-shard-nr').inputValue()
            ).toBe('2');
            await expect(
              page.getByTestId('shard-config-servers-1')
            ).toHaveAttribute('aria-pressed', 'true');
          }
        });
      });

      test(`Disk resize during edition for ${db} should be disabled`, async ({
        page,
      }) => {
        await page.goto('/databases');
        await findDbAndClickRow(page, clusterName);

        await test.step('Open edit resource modal', async () => {
          openResourcesModal(page);
        });
        await expect(page.getByTestId('text-input-disk')).toBeDisabled();
      });

      test('Set custom resources to nodes and proxies during editing', async ({
        page,
      }) => {
        await page.goto('/databases');
        await findDbAndClickRow(page, clusterName);
        await test.step('Open edit resource modal', async () => {
          openResourcesModal(page);
        });

        await test.step('Set custom resources size per node', async () => {
          await expect(
            page.getByTestId('node-resources-toggle-button-custom')
          ).toHaveAttribute('aria-pressed', 'false');
          page.getByTestId('text-input-cpu').fill('2');
          await expect(
            page.getByTestId('node-resources-toggle-button-custom')
          ).toHaveAttribute('aria-pressed', 'true');
        });

        //TODO can be better customizable between different dbTypes
        await page.getByTestId('proxies-accordion').click();
        await test.step(`Set custom number of ${db != 'psmdb' ? 'proxies' : 'routers'}`, async () => {
          await page
            .getByTestId(
              `toggle-button-${db != 'psmdb' ? 'proxies' : 'routers'}-custom`
            )
            .click();
          await page.getByTestId('text-input-custom-nr-of-proxies').fill('2');
        });

        await test.step(`Set custom resources size per ${db != 'psmdb' ? 'proxies' : 'routers'}`, async () => {
          await expect(
            page.getByTestId(
              `${db != 'psmdb' ? 'proxy' : 'router'}-resources-toggle-button-custom`
            )
          ).toHaveAttribute('aria-pressed', 'false');
          page
            .getByTestId('text-input-proxy-cpu')
            .fill(`${db != 'psmdb' ? '0.4' : '2'}`);
          await expect(
            page.getByTestId(
              `${db != 'psmdb' ? 'proxy' : 'router'}-resources-toggle-button-custom`
            )
          ).toHaveAttribute('aria-pressed', 'true');
        });

        expect(page.getByTestId('form-dialog-save')).not.toBeDisabled();
        await page.getByTestId('form-dialog-save').click();

        //check result
        await expect(
          page
            .getByTestId('node-cpu-overview-section-row')
            .filter({ hasText: '2' })
        ).toBeVisible();

        await expect(
          page
            .getByTestId(
              `${db != 'psmdb' ? 'proxies' : 'routers'}-cpu-overview-section-row`
            )
            .filter({
              hasText: `${db != 'psmdb' ? '0.8' : '4'}`,
            })
        ).toBeVisible();
      });

      test(`Delete cluster [${db} size ${size}]`, async ({ page }) => {
        await deleteDbCluster(page, clusterName);
        // We do not wait for total deletion for timeout purposes (costing more than 15m on the CI)
      });
    });
  }
);

// A "legacy" cluster is one created before the requests/limits split: it stores
// a single flat cpu/memory value with no explicit limits/requests.
// `createDbClusterFn` intentionally writes the legacy flat shape, so we use it
// to reproduce the scenario.
//
// PXC's operator does NOT default absent requests to the limits, so editing a
// legacy PXC cluster must always migrate it to explicit requests (equal to the
// limits when synced) to preserve the effective resource configuration.

test.describe
  .serial('Legacy PXC cluster resources editing always writes requests', () => {
  test.describe.configure({ timeout: 1000000 });

  const clusterName = 'legacy-pxc-res-edit';
  let namespace: string;

  const openResourcesModal = async (page: Page) => {
    const editResourcesButton = page.getByTestId('edit-resources-button');
    await editResourcesButton.waitFor();
    await editResourcesButton.click();
    await expect(page.getByTestId('edit-resources-form-dialog')).toBeVisible();
  };

  test.beforeAll(async ({ request }) => {
    token = await getTokenFromLocalStorage();
    const namespaces = await getNamespacesFn(token, request);
    namespace = namespaces[0];

    await createDbClusterFn(
      request,
      {
        dbName: clusterName,
        dbType: 'mysql',
        numberOfNodes: '1',
        cpu: 1,
        disk: 1,
        memory: 1,
        proxyCpu: 1,
        proxyMemory: 1,
      },
      namespace
    );
  });

  test.beforeEach(async ({ page }) => {
    await page.goto('/databases');
    await waitForInitializingState(page, clusterName);
  });

  test.afterAll(async ({ request }) => {
    await deleteDbClusterFn(request, clusterName, namespace);
  });

  test('writes requests equal to limits when saved while synced', async ({
    page,
    request,
  }) => {
    await findDbAndClickRow(page, clusterName);

    await test.step('Open edit resource modal', async () => {
      await openResourcesModal(page);
    });

    await test.step('Requests are synced and hidden for a legacy PXC cluster', async () => {
      const syncSwitch = page
        .getByTestId('switch-input-node-requests-synced-label')
        .getByRole('checkbox');
      await expect(syncSwitch).toBeChecked();
      await expect(
        page.getByTestId('text-input-cpu-requests')
      ).not.toBeVisible();
    });

    await test.step('Save without changes', async () => {
      await expect(page.getByTestId('form-dialog-save')).not.toBeDisabled();
      await page.getByTestId('form-dialog-save').click();
    });

    await test.step('The CR has both limits and requests (equal)', async () => {
      await expect(async () => {
        const cluster = await getDbClusterAPI(
          clusterName,
          namespace,
          request,
          token
        );
        expect(cluster.spec.engine.resources.limits).toBeDefined();
        expect(cluster.spec.engine.resources.requests).toBeDefined();
        expect(cluster.spec.engine.resources.requests.cpu.toString()).toBe(
          cluster.spec.engine.resources.limits.cpu.toString()
        );
        expect(cluster.spec.engine.resources.requests.memory.toString()).toBe(
          cluster.spec.engine.resources.limits.memory.toString()
        );
      }).toPass({ timeout: 30000 });
    });
  });

  test('writes a lower request when the user consciously desyncs it', async ({
    page,
    request,
  }) => {
    await findDbAndClickRow(page, clusterName);

    await test.step('Open edit resource modal', async () => {
      await openResourcesModal(page);
    });

    await test.step('Turn off sync and set a lower CPU request', async () => {
      const syncSwitch = page
        .getByTestId('switch-input-node-requests-synced-label')
        .getByRole('checkbox');
      await syncSwitch.uncheck();

      const cpuRequest = page.getByTestId('text-input-cpu-requests');
      await expect(cpuRequest).toBeVisible();
      await cpuRequest.fill('0.5');
    });

    await test.step('Save the form', async () => {
      await expect(page.getByTestId('form-dialog-save')).not.toBeDisabled();
      await page.getByTestId('form-dialog-save').click();
    });

    await test.step('The CR now has explicit requests', async () => {
      await expect(async () => {
        const cluster = await getDbClusterAPI(
          clusterName,
          namespace,
          request,
          token
        );
        expect(cluster.spec.engine.resources.requests).toBeDefined();
        expect(cluster.spec.engine.resources.requests.cpu).toBeDefined();
      }).toPass({ timeout: 30000 });
    });
  });
});

// PostgreSQL (like PSMDB) defaults absent requests to the limits, so editing a
// legacy PostgreSQL cluster while keeping requests synced must persist limits
// only. Adding explicit requests here would trigger an unnecessary restart.

test.describe
  .serial('Legacy PostgreSQL cluster resources editing keeps limits only', () => {
  test.describe.configure({ timeout: 1000000 });

  const clusterName = 'legacy-pg-res-edit';
  let namespace: string;

  const openResourcesModal = async (page: Page) => {
    const editResourcesButton = page.getByTestId('edit-resources-button');
    await editResourcesButton.waitFor();
    await editResourcesButton.click();
    await expect(page.getByTestId('edit-resources-form-dialog')).toBeVisible();
  };

  test.beforeAll(async ({ request }) => {
    token = await getTokenFromLocalStorage();
    const namespaces = await getNamespacesFn(token, request);
    namespace = namespaces[0];

    await createDbClusterFn(
      request,
      {
        dbName: clusterName,
        dbType: 'postgresql',
        numberOfNodes: '1',
        cpu: 1,
        disk: 1,
        memory: 1,
        proxyCpu: 1,
        proxyMemory: 1,
      },
      namespace
    );
  });

  test.beforeEach(async ({ page }) => {
    await page.goto('/databases');
    await waitForInitializingState(page, clusterName);
  });

  test.afterAll(async ({ request }) => {
    await deleteDbClusterFn(request, clusterName, namespace);
  });

  test('starts synced and persists limits only when saved without changes', async ({
    page,
    request,
  }) => {
    await findDbAndClickRow(page, clusterName);

    await test.step('Open edit resource modal', async () => {
      await openResourcesModal(page);
    });

    await test.step('Requests are synced and hidden for a legacy cluster', async () => {
      const syncSwitch = page
        .getByTestId('switch-input-node-requests-synced-label')
        .getByRole('checkbox');
      await expect(syncSwitch).toBeChecked();
      await expect(
        page.getByTestId('text-input-cpu-requests')
      ).not.toBeVisible();
    });

    await test.step('Save without changes', async () => {
      await expect(page.getByTestId('form-dialog-save')).not.toBeDisabled();
      await page.getByTestId('form-dialog-save').click();
    });

    await test.step('The CR still has limits only (no requests)', async () => {
      await expect(async () => {
        const cluster = await getDbClusterAPI(
          clusterName,
          namespace,
          request,
          token
        );
        expect(cluster.spec.engine.resources.limits).toBeDefined();
        expect(cluster.spec.engine.resources.requests).toBeUndefined();
      }).toPass({ timeout: 30000 });
    });
  });
});
