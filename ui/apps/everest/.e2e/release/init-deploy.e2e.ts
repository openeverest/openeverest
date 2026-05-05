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
  deleteDbCluster,
  suspendDbCluster,
  resumeDbCluster,
  restartDbCluster,
  findDbAndClickRow,
} from '@e2e/utils/db-clusters-list';
import { getTokenFromLocalStorage } from '@e2e/utils/localStorage';
import { getClusterDetailedInfo } from '@e2e/utils/storage-class';
import {
  moveForward,
  submitWizard,
  populateBasicInformation,
  populateResources,
  populateAdvancedConfig,
} from '@e2e/utils/db-wizard';
import { EVEREST_CI_NAMESPACES } from '@e2e/constants';
import { waitForStatus, waitForDelete } from '@e2e/utils/table';
import { getDbClusterAPI } from '@e2e/utils/db-cluster';
import { shouldExecuteDBCombination } from '@e2e/utils/generic';
import {
  queryPG,
  queryPSMDB,
  queryMySQL,
  getPGStsName,
} from '@e2e/utils/db-cmd-line';
import { getK8sResource } from '@e2e/utils/kubernetes';
import { checkDBMetrics, checkQAN } from '@e2e/utils/monitoring-instance';

let token: string;

test.describe.configure({ retries: 0 });

const zephyrMap: Record<string, string> = {
  'create-psmdb-1': 'T66',
  'create-psmdb-3': 'T67',
  'create-psmdb-5': 'T68',
  'create-pxc-1': 'T72',
  'create-pxc-3': 'T73',
  'create-pxc-5': 'T74',
  'create-postgresql-1': 'T77',
  'create-postgresql-2': 'T78',
  'create-postgresql-3': 'T79',
  'edit-pxc': 'T85',
  'edit-psmdb': 'T86',
  'edit-postgresql': 'T87',
  'external-pxc': 'T70',
  'external-psmdb': 'T76',
  'external-postgresql': 'T82',
  'custom-pxc': 'T20',
  'custom-psmdb': 'T46',
  'custom-postgresql': 'T47',
  'monitoring-pxc': 'T69',
  'monitoring-psmdb': 'T75',
  'monitoring-postgresql': 'T81',
  'suspend-pxc': 'T90',
  'suspend-psmdb': 'T91',
  'suspend-postgresql': 'T92',
  'resume-pxc': 'T93',
  'resume-psmdb': 'T94',
  'resume-postgresql': 'T95',
  'restart-pxc': 'T87',
  'restart-psmdb': 'T88',
  'restart-postgresql': 'T89',
  'delete-pxc': 'T96',
  'delete-psmdb': 'T97',
  'delete-postgresql': 'T98',
};

[
  { db: 'psmdb', size: 1 },
  { db: 'psmdb', size: 3 },
  { db: 'psmdb', size: 5 },
  { db: 'pxc', size: 1 },
  { db: 'pxc', size: 3 },
  { db: 'pxc', size: 5 },
  { db: 'postgresql', size: 1 },
  { db: 'postgresql', size: 2 },
  { db: 'postgresql', size: 3 },
].forEach(({ db, size }) => {
  test.describe(
    'Initial deployment',
    {
      tag: '@release',
    },
    () => {
      test.skip(!shouldExecuteDBCombination(db, size));
      test.describe.configure({ timeout: 720000 });

      const clusterName = `${db}-${size}-deploy`;
      let zephyrId: string;

      let storageClasses = [];
      const namespace = EVEREST_CI_NAMESPACES.EVEREST_UI;
      const monitoringName = 'e2e-endpoint-0';

      test.beforeAll(async ({ request }) => {
        token = await getTokenFromLocalStorage();

        const { storageClassNames = [] } = await getClusterDetailedInfo(
          token,
          request
        );
        storageClasses = storageClassNames;
      });

      zephyrId = zephyrMap[`create-${db}-${size}`];
      test(`${zephyrId} - Cluster creation [${db} size ${size}]`, async ({
        page,
        request,
      }) => {
        expect(storageClasses.length).toBeGreaterThan(0);

        await page.goto('/databases');
        await page.getByTestId('add-db-cluster-button').waitFor();
        await page.getByTestId('add-db-cluster-button').click();
        await page.getByTestId(`add-db-cluster-button-${db}`).click();

        // await page.getByTestId('toggle-button-group-input-db-type').waitFor();
        await page.getByTestId('select-input-db-version').waitFor();

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
          await page
            .getByRole('button')
            .getByText(size + ' node')
            .click();

          await expect(page.getByText('Nodes (' + size + ')')).toBeVisible();
          await populateResources(page, 0.6, 1, 1, size);
          await moveForward(page);
        });

        await test.step('Populate backups', async () => {
          await moveForward(page);
        });

        await test.step('Populate advanced db config', async () => {
          // For clusters of size 3 we test with external access enabled
          if (size === 3) {
            await populateAdvancedConfig(page, db, true, '', true, '');
          } else {
            await populateAdvancedConfig(page, db, false, '', true, '');
          }
          await moveForward(page);
        });

        await test.step('Populate monitoring', async () => {
          await page.getByTestId('switch-input-monitoring').click();
          await page
            .getByTestId('text-input-monitoring-instance')
            .fill(monitoringName);
          await expect(
            page.getByTestId('text-input-monitoring-instance')
          ).toHaveValue(monitoringName);
        });

        await test.step('Submit wizard', async () => {
          await submitWizard(page);
        });

        await test.step('Check db list and status', async () => {
          await page.goto('/databases');
          // TODO: try re-enable after fix for: https://perconadev.atlassian.net/browse/EVEREST-1693
          if (size != 1 || db != 'psmdb') {
            await waitForStatus(page, clusterName, 'Initializing', 30000);
          }
          await waitForStatus(page, clusterName, 'Up', 720000);
        });

        await test.step('Check db cluster k8s object options', async () => {
          const addedCluster = await getDbClusterAPI(
            clusterName,
            EVEREST_CI_NAMESPACES.EVEREST_UI,
            request,
            token
          );

          expect(addedCluster?.spec.engine.type).toBe(db);
          expect(addedCluster?.spec.engine.replicas).toBe(size);
          expect(['600m', '0.6']).toContain(
            addedCluster?.spec.engine.resources?.cpu.toString()
          );
          expect(addedCluster?.spec.engine.resources?.memory.toString()).toBe(
            '1G'
          );
          expect(addedCluster?.spec.engine.storage.size.toString()).toBe('1Gi');
          if (size === 3) {
            expect(addedCluster?.spec.proxy.expose.type).toBe('LoadBalancer');
          } else {
            expect(addedCluster?.spec.proxy.expose.type).toBe('ClusterIP');
          }
          if (db != 'psmdb') {
            expect(addedCluster?.spec.proxy.replicas).toBe(size);
          }
        });
      });

      zephyrId = zephyrMap[`external-${db}`];
      test(`${zephyrId} - Check service type is LoadBalancer [${db} size ${size}]`, async () => {
        test.skip(size !== 3, 'We only enable external access for size 3');

        let resourceName: string;

        switch (db) {
          case 'pxc': {
            resourceName = `${clusterName}-haproxy`;
            break;
          }
          case 'psmdb': {
            resourceName = `${clusterName}-rs0-0`;
            break;
          }
          case 'postgresql': {
            resourceName = `${clusterName}-pgbouncer`;
            break;
          }
        }

        let resource = await getK8sResource(
          'svc',
          `${resourceName}`,
          'everest-ui'
        );
        expect(resource?.spec.type).toBe('LoadBalancer');
      });

      zephyrId = zephyrMap[`custom-${db}`];
      test(`${zephyrId} - Check DB custom option [${db} size ${size}]`, async () => {
        let result: string;

        switch (db) {
          case 'pxc': {
            result = await queryMySQL(
              clusterName,
              namespace,
              `SHOW variables LIKE "max_connections";`,
              40 // we retry here because LoadBalancer needs some time to be visible in DNS
            );
            expect(result.trim()).toBe('max_connections	250');
            break;
          }
          case 'psmdb': {
            result = await queryPSMDB(
              clusterName,
              namespace,
              'admin',
              `db.serverCmdLineOpts().parsed.systemLog;`,
              40 // we retry here because LoadBalancer needs some time to be visible in DNS
            );
            expect(result.trim()).toBe('{ quiet: true, verbosity: 1 }');
            break;
          }
          case 'postgresql': {
            result = await queryPG(
              clusterName,
              namespace,
              'postgres',
              `SHOW shared_buffers;`,
              40 // we retry here because LoadBalancer needs some time to be visible in DNS
            );
            expect(result.trim()).toBe('192MB');
            break;
          }
        }
      });

      zephyrId = zephyrMap[`monitoring-${db}`];
      test(`${zephyrId} - Check PMM db metrics and QAN [${db} size ${size}]`, async () => {
        test.step('Check DB metrics', async () => {
          switch (db) {
            case 'psmdb': {
              for (let i = 0; i < size; i++) {
                await checkDBMetrics(
                  'node_boot_time_seconds',
                  `everest-ui-${clusterName}-rs0-${i}`,
                  'admin:admin'
                );
                await checkDBMetrics(
                  'mongodb_connections',
                  `everest-ui-${clusterName}-rs0-${i}`,
                  'admin:admin'
                );
              }
              break;
            }
            case 'pxc': {
              const nodeTypes = ['pxc', 'haproxy'];

              for (const nodeType of nodeTypes) {
                for (let i = 0; i < size; i++) {
                  switch (nodeType) {
                    case 'pxc': {
                      await checkDBMetrics(
                        'node_boot_time_seconds',
                        `everest-ui-${clusterName}-${nodeType}-${i}`,
                        'admin:admin'
                      );
                      await checkDBMetrics(
                        'mysql_global_status_uptime',
                        `everest-ui-${clusterName}-${nodeType}-${i}`,
                        'admin:admin'
                      );
                      break;
                    }
                    case 'haproxy': {
                      await checkDBMetrics(
                        'haproxy_backend_status',
                        `everest-ui-${clusterName}-${nodeType}-${i}`,
                        'admin:admin'
                      );
                      await checkDBMetrics(
                        'haproxy_backend_active_servers',
                        `everest-ui-${clusterName}-${nodeType}-${i}`,
                        'admin:admin'
                      );
                      break;
                    }
                  }
                }
              }
              break;
            }
            case 'postgresql': {
              const pgSts = await getPGStsName(clusterName, namespace);
              for (let i = 0; i < size; i++) {
                await checkDBMetrics(
                  'node_boot_time_seconds',
                  `everest-ui-${pgSts[i]}-0`,
                  'admin:admin'
                );
                await checkDBMetrics(
                  'pg_postmaster_uptime_seconds',
                  `everest-ui-${pgSts[i]}-0`,
                  'admin:admin'
                );
              }
              break;
            }
          }
        });

        await test.step(`Check PMM QAN`, async () => {
          // Wait for 90 seconds for QAN to get data
          await new Promise((resolve) => setTimeout(resolve, 90000));

          switch (db) {
            case 'psmdb': {
              // for PSMDB we see QAN only for the first node (primary)
              await checkQAN(
                'mongodb',
                `everest-ui-${clusterName}-rs0-0`,
                'admin:admin'
              );
              break;
            }
            case 'pxc': {
              for (let i = 0; i < size; i++) {
                await checkQAN(
                  'mysql',
                  `everest-ui-${clusterName}-pxc-${i}`,
                  'admin:admin'
                );
              }
              break;
            }
            case 'postgresql': {
              const pgSts = await getPGStsName(clusterName, namespace);
              for (let i = 0; i < size; i++) {
                await checkQAN(
                  'postgresql',
                  `everest-ui-${pgSts[i]}-0`,
                  'admin:admin'
                );
              }
              break;
            }
          }
        });
      });

      zephyrId = zephyrMap[`suspend-${db}`];
      test(`${zephyrId} - Suspend cluster [${db} size ${size}]`, async ({
        page,
      }) => {
        await suspendDbCluster(page, clusterName);
        // One node clusters and Postgresql don't seem to show Stopping state
        if (size != 1 && db != 'postgresql') {
          await waitForStatus(page, clusterName, 'Stopping', 60000);
        }
        await waitForStatus(page, clusterName, 'Paused', 240000);
      });

      zephyrId = zephyrMap[`resume-${db}`];
      test(`${zephyrId} - Resume cluster [${db} size ${size}]`, async ({
        page,
      }) => {
        await resumeDbCluster(page, clusterName);
        // TODO: try re-enable after fix for: https://perconadev.atlassian.net/browse/EVEREST-1693
        if (size != 1 || db != 'psmdb') {
          await waitForStatus(page, clusterName, 'Initializing', 45000);
        }
        await waitForStatus(page, clusterName, 'Up', 660000);
      });

      zephyrId = zephyrMap[`restart-${db}`];
      test(`${zephyrId} - Restart cluster [${db} size ${size}]`, async ({
        page,
      }) => {
        await restartDbCluster(page, clusterName);
        if (size != 1 && db != 'postgresql') {
          await waitForStatus(page, clusterName, 'Stopping', 45000);
        }
        // TODO: try re-enable after fix for: https://perconadev.atlassian.net/browse/EVEREST-1693
        if (size != 1 || db != 'psmdb') {
          await waitForStatus(page, clusterName, 'Initializing', 120000);
        }
        await waitForStatus(page, clusterName, 'Up', 660000);
      });

      zephyrId = zephyrMap[`edit-${db}`];
      test(`${zephyrId} - Edit cluster/scale up [${db} size ${size}]`, async ({
        page,
      }) => {
        test.skip(size > 3, 'We dont want to scale up 5 node clusters');
        const newSize = size + 2;
        let customProxyTestId = 'toggle-button-proxies-custom';

        await test.step('Change resource options', async () => {
          await page.goto('databases');
          await findDbAndClickRow(page, clusterName);
          await page.getByTestId('edit-resources-button').click();
          if (db !== 'pxc') {
            await page.getByTestId('toggle-button-nodes-custom').click();
            await page
              .getByTestId('text-input-custom-nr-of-nodes')
              .fill(newSize.toString());
          } else {
            await page.getByTestId(`toggle-button-nodes-${newSize}`).click();
          }

          if (db === 'postgresql') {
            customProxyTestId = 'toggle-button-PG Bouncers-custom';
          }
          if (db === 'pxc' || db === 'postgresql') {
            await page.getByTestId('ExpandMoreIcon').last().click();
            await page.getByTestId(customProxyTestId).click();
            await page.getByTestId('text-input-custom-nr-of-proxies').fill('2');
          }
          await page.getByTestId('form-dialog-save').click();
        });

        await test.step('Check new resource values', async () => {
          if (db === 'pxc') {
            await expect(
              page
                .getByTestId('overview-section')
                .filter({ hasText: '2 proxies' })
            ).toBeVisible({ timeout: 10000 });
          } else if (db === 'postgresql') {
            await expect(
              page
                .getByTestId('overview-section')
                .filter({ hasText: '2 PG Bouncers' })
            ).toBeVisible({ timeout: 10000 });
          }

          await expect(
            page
              .getByTestId('overview-section')
              .filter({ hasText: `${newSize} nodes` })
          ).toBeVisible({ timeout: 10000 });
        });

        await test.step('Wait for cluster status', async () => {
          await page.goto('databases');
          // TODO: try re-enable after fix for: https://perconadev.atlassian.net/browse/EVEREST-1693
          if (size != 1 || db != 'psmdb') {
            await waitForStatus(page, clusterName, 'Initializing', 60000);
          }
          await waitForStatus(page, clusterName, 'Up', 300000);
        });
      });

      zephyrId = zephyrMap[`external-${db}`];
      test(`${zephyrId} - Change external access options [${db} size ${size}]`, async ({
        page,
      }) => {
        test.skip(size !== 3, 'We only enable external access for size 3');

        await test.step('Set ipSourceRange', async () => {
          await page.goto('databases');
          await findDbAndClickRow(page, clusterName);
          await page.getByTestId('edit-advanced-configuration-db-btn').click();
          await page
            .getByTestId('text-input-source-ranges.0.source-range')
            .fill('192.168.1.0/32');
          await page.getByTestId('form-dialog-save').click();
        });

        await test.step('Check new external access values in UI', async () => {
          await page.getByTestId('edit-advanced-configuration-db-btn').click();
          const rawValue = await page
            .getByTestId('text-input-source-ranges.0.source-range')
            .inputValue();
          await expect(rawValue).toEqual('192.168.1.0/32');
        });

        await test.step(`Check service in K8s [${db} size ${size}]`, async () => {
          let resourceName: string;

          await page.waitForTimeout(15000); // wait for svc to be updated
          switch (db) {
            case 'pxc': {
              resourceName = `${clusterName}-haproxy`;
              break;
            }
            case 'psmdb': {
              resourceName = `${clusterName}-rs0-0`;
              break;
            }
            case 'postgresql': {
              resourceName = `${clusterName}-pgbouncer`;
              break;
            }
          }

          const resource = await getK8sResource(
            'svc',
            `${resourceName}`,
            'everest-ui'
          );
          expect(resource?.spec.type).toBe('LoadBalancer');
          expect(resource?.spec.loadBalancerSourceRanges).toEqual([
            '192.168.1.0/32',
          ]);
        });
      });

      zephyrId = zephyrMap[`delete-${db}`];
      test(`${zephyrId} - Delete cluster [${db} size ${size}]`, async ({
        page,
      }) => {
        await deleteDbCluster(page, clusterName);
        await waitForStatus(page, clusterName, 'Deleting', 15000);
        await waitForDelete(page, clusterName, 240000);
      });
    }
  );
});
