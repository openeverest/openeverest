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
import { deleteDbCluster } from '@e2e/utils/db-clusters-list';
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
import {
  waitForStatus,
  waitForDelete,
  findRowAndClickActions,
} from '@e2e/utils/table';
import { getDbClusterAPI } from '@e2e/utils/db-cluster';
import { shouldExecuteDBCombination } from '@e2e/utils/generic';
import {
  createPodSchedulingPolicyWithValues,
  createRuleForPodSchedulingPolicyWithValues,
  deletePodSchedulingPolicy,
  getDefaultPodSchedulingPolicyNameForDbType,
} from '@e2e/utils/policies';
import {
  addLabelToK8sNode,
  deleteAllK8sPVCs,
  getK8sNodes,
  getK8sResource,
  removeLabelFromK8sNode,
} from '@e2e/utils/kubernetes';

let token: string;

test.describe.configure({ retries: 0 });

[
  { db: 'psmdb', size: 3, nProxies: 5 },
  { db: 'psmdb', size: 3, nShards: 2 },
  { db: 'pxc', size: 3, nProxies: 5 },
  { db: 'postgresql', size: 2, nProxies: 3 },
].forEach(({ db, size, nProxies, nShards }) => {
  test.describe(
    'Default Pod Scheduling Policies',
    {
      tag: '@release',
    },
    () => {
      test.skip(!shouldExecuteDBCombination(db, size));
      test.describe.configure({ timeout: 720000 });

      const clusterName = `${db}-${size}${nProxies ? `-${nProxies}` : ''}${nShards ? `-${nShards}` : ''}`;
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

      test(`Cluster creation [${db}, size ${size}, nProxies ${nProxies}, nShards ${nShards}]`, async ({
        page,
        request,
      }) => {
        expect(storageClasses.length).toBeGreaterThan(0);

        await page.goto('/databases');
        await page.getByTestId('add-db-cluster-button').waitFor();
        await page.getByTestId('add-db-cluster-button').click();
        await page.getByTestId(`add-db-cluster-button-${db}`).click();

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
          await populateResources(
            page,
            0.6,
            1,
            1,
            size,
            nProxies,
            0.6,
            1,
            nShards
          );
          await moveForward(page);
        });

        await test.step('Populate backups', async () => {
          await moveForward(page);
        });

        await test.step('Populate advanced db config', async () => {
          await populateAdvancedConfig(page, db, false, '', true, '');
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

        // go to db list and check status
        await test.step('Check db list and status', async () => {
          await page.goto('/databases');
          await waitForStatus(page, clusterName, 'Initializing', 30000);
          await waitForStatus(page, clusterName, 'Up', 660000);
        });

        await test.step('Check db cluster k8s object options', async () => {
          const addedCluster = await getDbClusterAPI(
            clusterName,
            EVEREST_CI_NAMESPACES.EVEREST_UI,
            request,
            token
          );

          expect(addedCluster?.spec.podSchedulingPolicyName).toBe(
            getDefaultPodSchedulingPolicyNameForDbType(db)
          );
        });
      });

      test(`Delete cluster [${db}, size ${size}, nProxies ${nProxies}, nShards ${nShards}]`, async ({
        page,
      }) => {
        await deleteDbCluster(page, clusterName);
        await waitForStatus(page, clusterName, 'Deleting', 15000);
        await waitForDelete(page, clusterName, 240000);
      });
    }
  );
});

[
  { db: 'pxc', size: 3 },
  { db: 'psmdb', size: 3 },
  { db: 'postgresql', size: 2 },
].forEach(
  ({
    db,
    size,
    nProxies,
    nShards,
  }: {
    db: string;
    size: number;
    nProxies?: number;
    nShards?: number;
  }) => {
    test.describe(
      'Disabled Pod Scheduling Policies',
      {
        tag: '@release',
      },
      () => {
        test.skip(!shouldExecuteDBCombination(db, size));
        test.describe.configure({ timeout: 720000 });

        const clusterName = `${db}-${size}${nProxies ? `-${nProxies}` : ''}${nShards ? `-${nShards}` : ''}`;

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

        test(`Cluster creation [${db}, size ${size}, nProxies ${nProxies}, nShards ${nShards}]`, async ({
          page,
          request,
        }) => {
          expect(storageClasses.length).toBeGreaterThan(0);

          await page.goto('/databases');
          await page.getByTestId('add-db-cluster-button').waitFor();
          await page.getByTestId('add-db-cluster-button').click();
          await page.getByTestId(`add-db-cluster-button-${db}`).click();

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
            await populateResources(
              page,
              0.6,
              1,
              1,
              size,
              nProxies,
              0.6,
              1,
              nShards
            );
            await moveForward(page);
          });

          await test.step('Populate backups', async () => {
            await moveForward(page);
          });

          await test.step('Populate advanced db config', async () => {
            await populateAdvancedConfig(page, db, false, '', true, '', false);
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

          // go to db list and check status
          await test.step('Check db list and status', async () => {
            await page.goto('/databases');
            await waitForStatus(page, clusterName, 'Initializing', 30000);
            await waitForStatus(page, clusterName, 'Up', 660000);
          });

          await test.step('Check db cluster k8s object options', async () => {
            const addedCluster = await getDbClusterAPI(
              clusterName,
              EVEREST_CI_NAMESPACES.EVEREST_UI,
              request,
              token
            );

            expect(addedCluster?.spec.podSchedulingPolicyName).toBeUndefined();
          });
        });

        test(`Delete cluster [${db} size ${size}]`, async ({ page }) => {
          await deleteDbCluster(page, clusterName);
          await waitForStatus(page, clusterName, 'Deleting', 15000);
          await waitForDelete(page, clusterName, 240000);
        });
      }
    );
  }
);

test.describe(
  'Pods assigment',
  {
    tag: '@release',
  },
  () => {
    const MySQLPolicyName = 'mysql-pol-1';
    const firstClusterName = 'mysql-ac1';
    const secondClusterName = 'mysql-ra8';

    const enginePodName = `${secondClusterName}-pxc-0`;
    const proxyPodName = `${secondClusterName}-haproxy-0`;

    let storageClasses = [];
    let k8sNodes: string[] = [];
    const namespace = EVEREST_CI_NAMESPACES.EVEREST_UI;

    test.describe.configure({ timeout: 1_200_000 });

    test.beforeAll(async ({ request }) => {
      token = await getTokenFromLocalStorage();

      const { storageClassNames = [] } = await getClusterDetailedInfo(
        token,
        request
      );
      storageClasses = storageClassNames;

      k8sNodes = await getK8sNodes();

      if (!k8sNodes.length || k8sNodes.length < 3) {
        throw new Error('3 k8s nodes are required for this test');
      }
    });

    test('Create Pod Scheduling Policies', async ({ page }) => {
      await test.step('Create MySQL policy', async () => {
        await createPodSchedulingPolicyWithValues(page, MySQLPolicyName, 'pxc');
      });

      await test.step('Add engine node affinity rule to MySQL policy', async () => {
        await createRuleForPodSchedulingPolicyWithValues(
          page,
          MySQLPolicyName,
          'engine',
          'nodeAffinity',
          'required',
          1,
          '',
          'size',
          'In',
          'medium,large'
        );
      });

      await test.step('Add proxy node affinity rule to MySQL policy', async () => {
        await createRuleForPodSchedulingPolicyWithValues(
          page,
          MySQLPolicyName,
          'proxy',
          'nodeAffinity',
          'required',
          1,
          '',
          'size',
          'In',
          'small'
        );
      });

      await test.step('Create MySQL cluster with policy', async () => {
        await page.goto('/databases');
        await page.getByTestId('add-db-cluster-button').waitFor();
        await page.getByTestId('add-db-cluster-button').click();
        await page.getByTestId('add-db-cluster-button-pxc').click();
        await populateBasicInformation(
          page,
          EVEREST_CI_NAMESPACES.EVEREST_UI,
          firstClusterName,
          'pxc',
          storageClasses[0],
          false,
          null
        );
        await moveForward(page);

        await page.getByRole('button').getByText('1 node').click();
        await populateResources(page, 0.6, 1, 1, 1, 1, 0.6, 1);

        await moveForward(page);
        await moveForward(page);

        await expect(
          page
            .getByTestId('switch-input-pod-scheduling-policy-enabled')
            .getByRole('checkbox')
        ).toBeEnabled();

        await page
          .getByTestId('switch-input-pod-scheduling-policy-enabled')
          .getByRole('checkbox')
          .check();
        await page.getByTestId('select-pod-scheduling-policy-button').click();
        await page
          .getByRole('option', { name: MySQLPolicyName, exact: true })
          .click();

        await moveForward(page);
        await submitWizard(page);
      });

      await test.step('Assign size=large labels to the two first k8s nodes', async () => {
        for (let i = 0; i < 2; i++) {
          await addLabelToK8sNode(k8sNodes[i], 'size', 'large');
        }
      });

      await test.step('Create MySQL cluster with no policy', async () => {
        await page.goto('/databases');
        await page.getByTestId('add-db-cluster-button').waitFor();
        await page.getByTestId('add-db-cluster-button').click();
        await page.getByTestId('add-db-cluster-button-pxc').click();
        await populateBasicInformation(
          page,
          EVEREST_CI_NAMESPACES.EVEREST_UI,
          secondClusterName,
          'pxc',
          storageClasses[0],
          false,
          null
        );
        await moveForward(page);

        await page.getByRole('button').getByText('1 node').click();
        await populateResources(page, 0.6, 1, 1, 1, 1, 0.6, 1);

        await moveForward(page);
        await moveForward(page);

        await expect(
          page
            .getByTestId('switch-input-pod-scheduling-policy-enabled')
            .getByRole('checkbox')
        ).toBeEnabled();

        await page
          .getByTestId('switch-input-pod-scheduling-policy-enabled')
          .getByRole('checkbox')
          .uncheck();
        await moveForward(page);
        await submitWizard(page);
      });

      await test.step('Check up status', async () => {
        await page.goto('/databases');
        await waitForStatus(page, secondClusterName, 'Initializing', 30000);
      });

      await test.step('Show policies on overview', async () => {
        await page.goto(`/databases/${namespace}/${secondClusterName}`);
        await page.getByText('Advanced configuration').waitFor();
        await expect(
          page.getByTestId('pod-scheduling policy-overview-section-row')
        ).toHaveText('Pod scheduling policyDisabled');
        await page.getByTestId('edit-advanced-configuration-db-btn').click();
        await page
          .getByTestId('switch-input-pod-scheduling-policy-enabled')
          .getByRole('checkbox')
          .check();
        await page.getByTestId('select-pod-scheduling-policy-button').click();
        await page
          .getByRole('option', { name: MySQLPolicyName, exact: true })
          .click();
        await page.getByTestId('form-dialog-save').click();
        await expect(
          page.getByTestId('pod-scheduling policy-overview-section-row')
        ).toHaveText(`Pod scheduling policy${MySQLPolicyName}`);
      });

      await test.step('Check restart status', async () => {
        await page.goto('/databases');
        await waitForStatus(page, secondClusterName, 'Initializing', 30000);
      });

      await test.step('Assign size=small labels to the last k8s node', async () => {
        await addLabelToK8sNode(k8sNodes[2], 'size', 'small');
      });

      await test.step('Disable pod scheduling policies for the second cluster', async () => {
        await page.goto(`/databases/${namespace}/${secondClusterName}`);
        await page.getByText('Advanced configuration').waitFor();
        await page.getByTestId('edit-advanced-configuration-db-btn').click();
        await page
          .getByTestId('switch-input-pod-scheduling-policy-enabled')
          .getByRole('checkbox')
          .uncheck();
        await page.getByTestId('form-dialog-save').click();
      });

      await test.step('Check restart status', async () => {
        await page.goto('/databases');
        await waitForStatus(page, secondClusterName, 'Initializing', 30000);
      });

      await test.step('Re-enable pod scheduling policies for the second cluster', async () => {
        await page.goto(`/databases/${namespace}/${secondClusterName}`);
        await page.getByText('Advanced configuration').waitFor();
        await page.getByTestId('edit-advanced-configuration-db-btn').click();
        await page
          .getByTestId('switch-input-pod-scheduling-policy-enabled')
          .getByRole('checkbox')
          .check();
        await page.getByTestId('select-pod-scheduling-policy-button').click();
        await page
          .getByRole('option', { name: MySQLPolicyName, exact: true })
          .click();
        await page.getByTestId('form-dialog-save').click();
      });

      await test.step('Check restart status', async () => {
        await page.goto('/databases');
        await waitForStatus(page, secondClusterName, 'Initializing', 30000);
        await deleteAllK8sPVCs('everest-ui');
        await waitForStatus(page, secondClusterName, 'Up', 660000);
      });

      await test.step('Check pods assignment', async () => {
        const enginePod = await getK8sResource(
          'pods',
          enginePodName,
          namespace
        );
        const proxyPod = await getK8sResource('pods', proxyPodName, namespace);
        const engineNode = enginePod.spec.nodeName;
        const proxyNode = proxyPod.spec.nodeName;

        // Engines should be on large nodes
        expect(k8sNodes.slice(0, 2).includes(engineNode));
        // Proxies should be on small nodes
        expect(k8sNodes.slice(2, 3).includes(proxyNode));
      });

      await test.step('Try to remove policy in use', async () => {
        await page.goto('/settings/policies/details/pod-scheduling');
        await page.getByTestId('pod-scheduling-policies').waitFor();
        await findRowAndClickActions(page, MySQLPolicyName, 'Delete');
        await page.getByTestId('form-dialog-delete').waitFor();
        await expect(page.getByTestId('form-dialog-delete')).toBeDisabled();
      });
    });

    test(`Delete clusters and policies`, async ({ page, request }) => {
      for (let i = 0; i < 3; i++) {
        await removeLabelFromK8sNode(k8sNodes[i], 'size');
      }
      await deleteDbCluster(page, firstClusterName);
      await waitForStatus(page, firstClusterName, 'Deleting', 15000);
      await waitForDelete(page, firstClusterName, 240000);
      await deleteDbCluster(page, secondClusterName);
      await waitForStatus(page, secondClusterName, 'Deleting', 15000);
      await waitForDelete(page, secondClusterName, 240000);
      await deletePodSchedulingPolicy(request, MySQLPolicyName);
    });
  }
);
