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

import { expect, Page, test } from '@playwright/test';
import { moveForward } from '@e2e/utils/db-wizard';
import { findDbAndClickRow } from '@e2e/utils/db-clusters-list';
import { selectDbEngine } from '@e2e/pr/db-cluster/db-wizard/db-wizard-utils';
import { goToUrl, limitedSuffixedName } from '@e2e/utils/generic';
import { EVEREST_CI_NAMESPACES, technologyMap, TIMEOUTS } from '@e2e/constants';
import {
  createDbClusterFn,
  deleteDbClusterFn,
  getDbClusterAPI,
} from '@e2e/utils/db-cluster';
import { getCITokenFromLocalStorage } from '@e2e/utils/localStorage';
import { DbEngineType, DbType } from '@percona/types';
import { deletePodSchedulingPolicy } from '@e2e/utils/policies';

const PG_DEFAULT_PSP_NAME = 'everest-default-postgresql';
const PSMDB_DEFAULT_PSP_NAME = 'everest-default-mongodb';
const PXC_DEFAULT_PSP_NAME = 'everest-default-mysql';

type AffinityRuleFormArgs = {
  component?: 'DB Node' | 'Proxy' | 'Router' | 'PG Bouncer' | 'Config Server';
  type?: 'Node affinity' | 'Pod affinity' | 'Pod anti-affinity';
  preference?: 'preferred' | 'required';
  weight?: string;
  topologyKey?: string;
  key?: string;
  operator?: 'in' | 'not in' | 'exists' | 'does not exist';
  values?: string;
};

const fillAffinityRuleForm = async (
  page: Page,
  {
    weight,
    topologyKey,
    key,
    operator,
    values,
    component,
    type,
    preference,
  }: AffinityRuleFormArgs
) => {
  if (preference) {
    await page.getByTestId(`toggle-button-${preference}`).click();
  }

  if (type) {
    await page.getByTestId('select-type-button').click();
    await page.getByRole('option', { name: type, exact: true }).click();
  }

  if (component) {
    await page.getByTestId('select-component-button').click();
    await page.getByRole('option', { name: component, exact: true }).click();
  }

  if (weight) {
    await page.getByTestId('text-input-weight').fill(weight);
  }

  if (topologyKey) {
    await page.getByTestId('text-input-topology-key').fill(topologyKey);
  }

  if (key) {
    await page.getByTestId('text-input-key').fill(key);
  }

  if (operator) {
    await page.getByTestId('select-operator-button').click();
    await page.getByRole('option', { name: operator, exact: true }).click();
  }

  if (values) {
    await page.getByTestId('text-input-values').fill(values);
  }
};

test.describe.parallel('Policies Scheduling Policies', () => {
  let token: string;

  test.beforeAll(async ({}) => {
    token = await getCITokenFromLocalStorage();
    expect(token).not.toHaveLength(0);
  });

  test.describe.parallel('Default Policies', () => {
    test('Show default PG policy in DB creation wizard', async ({ page }) => {
      await goToUrl(page, '/databases');

      await selectDbEngine(page, 'postgresql');

      await test.step('Resources step', async () => {
        // Move to Resources step
        await moveForward(page);
      });

      await test.step('Backup Schedules step', async () => {
        // Move to "Scheduled Backups" step
        await moveForward(page);
      });

      await test.step('Advanced Configuration step', async () => {
        const getStorageClassResp = page.waitForResponse(
          (resp) =>
            resp.request().method() === 'GET' &&
            resp.url().includes('/v1/cluster-info') &&
            resp.status() === 200
        );

        const getPspResp = page.waitForResponse(
          (resp) =>
            resp.request().method() === 'GET' &&
            resp
              .url()
              .includes(
                '/v1/pod-scheduling-policies?engineType=postgresql&hasRules=true'
              ) &&
            resp.status() === 200
        );

        // Move to "Advanced Configuration" step
        await moveForward(page);

        // wait for requests to finish
        await getStorageClassResp;
        await getPspResp;

        const pspCheckBox = page
          .getByTestId('switch-input-pod-scheduling-policy-enabled')
          .getByRole('checkbox');
        await expect(pspCheckBox).toBeChecked();

        await page.getByTestId('select-pod-scheduling-policy-button').click();
        const pspOptions = page.getByRole('option');

        // check for default policy
        await expect(
          pspOptions.filter({ hasText: PG_DEFAULT_PSP_NAME })
        ).toBeVisible();
      });
    });

    test('Show default PSMDB policy in DB creation wizard', async ({
      page,
    }) => {
      await goToUrl(page, '/databases');

      await selectDbEngine(page, 'psmdb');

      await test.step('Resources step', async () => {
        // Move to Resources step
        await moveForward(page);
      });

      await test.step('Backup Schedules step', async () => {
        // Move to "Scheduled Backups" step
        await moveForward(page);
      });

      await test.step('Advanced Configuration step', async () => {
        const getStorageClassResp = page.waitForResponse(
          (resp) =>
            resp.request().method() === 'GET' &&
            resp.url().includes('/v1/cluster-info') &&
            resp.status() === 200
        );

        const getPspResp = page.waitForResponse(
          (resp) =>
            resp.request().method() === 'GET' &&
            resp
              .url()
              .includes(
                '/v1/pod-scheduling-policies?engineType=psmdb&hasRules=true'
              ) &&
            resp.status() === 200
        );

        // Move to "Advanced Configuration" step
        await moveForward(page);

        // wait for requests to finish
        await getStorageClassResp;
        await getPspResp;

        const pspCheckBox = page
          .getByTestId('switch-input-pod-scheduling-policy-enabled')
          .getByRole('checkbox');
        await expect(pspCheckBox).toBeChecked();

        await page.getByTestId('select-pod-scheduling-policy-button').click();
        const pspOptions = page.getByRole('option');

        // check for default policy
        await expect(
          pspOptions.filter({ hasText: PSMDB_DEFAULT_PSP_NAME })
        ).toBeVisible();
      });
    });

    test('Show default PXC policy in DB creation wizard', async ({ page }) => {
      await goToUrl(page, '/databases');

      await selectDbEngine(page, 'pxc');

      await test.step('Resources step', async () => {
        // Move to Resources step
        await moveForward(page);
      });

      await test.step('Backup Schedules step', async () => {
        // Move to "Scheduled Backups" step
        await moveForward(page);
      });

      await test.step('Advanced Configuration step', async () => {
        const getStorageClassResp = page.waitForResponse(
          (resp) =>
            resp.request().method() === 'GET' &&
            resp.url().includes('/v1/cluster-info') &&
            resp.status() === 200
        );

        const getPspResp = page.waitForResponse(
          (resp) =>
            resp.request().method() === 'GET' &&
            resp
              .url()
              .includes(
                '/v1/pod-scheduling-policies?engineType=pxc&hasRules=true'
              ) &&
            resp.status() === 200
        );

        // Move to "Advanced Configuration" step
        await moveForward(page);

        // wait for requests to finish
        await getStorageClassResp;
        await getPspResp;

        const pspCheckBox = page
          .getByTestId('switch-input-pod-scheduling-policy-enabled')
          .getByRole('checkbox');
        await expect(pspCheckBox).toBeChecked();

        await page.getByTestId('select-pod-scheduling-policy-button').click();
        const pspOptions = page.getByRole('option');

        // check for default policy
        await expect(
          pspOptions.filter({ hasText: PXC_DEFAULT_PSP_NAME })
        ).toBeVisible();
      });
    });
  });

  test.describe.serial('Custom PG Policy', () => {
    const customPGPolicyName = limitedSuffixedName('pr-set-psp-pg');

    test.afterAll(async ({ request }) => {
      await expect(async () => {
        await deletePodSchedulingPolicy(request, customPGPolicyName);
      }).toPass({
        intervals: [1000],
        timeout: TIMEOUTS.TenSeconds,
      });
    });

    test('Create policy for PG', async ({ page }) => {
      await goToUrl(page, '/settings/policies/pod-scheduling');

      await test.step(`Create policy ${customPGPolicyName} for PG`, async () => {
        await expect(page.getByTestId('add-policy')).toBeVisible();
        await page.getByTestId('add-policy').click();
        await page.waitForLoadState('load', {
          timeout: TIMEOUTS.ThirtySeconds,
        });

        await expect(page.getByTestId('text-input-name')).toBeVisible();
        await page.getByTestId('text-input-name').fill(customPGPolicyName);

        await expect(page.getByTestId('select-type-button')).toBeVisible();
        await page.getByTestId('select-type-button').click();
        await page
          .getByRole('option', {
            name: technologyMap[DbEngineType.POSTGRESQL],
            exact: true,
          })
          .click();

        // Request to create PSP
        const createPspResp = page.waitForResponse(
          (resp) =>
            resp.request().method() === 'POST' &&
            resp.url().includes('/v1/pod-scheduling-policies') &&
            resp.status() === 200
        );

        await page.getByTestId('form-dialog-create').click();
        await createPspResp;
      });

      await test.step('Create rules', async () => {
        await test.step('Add rule for DB Node', async () => {
          await page.getByTestId('AddIcon').click();

          await page.getByTestId('select-component-button').click();
          await page
            .getByRole('option', { name: 'DB Node', exact: true })
            .click();
          await page.getByTestId('toggle-button-preferred').click();

          await page
            .getByTestId('text-input-topology-key')
            .fill('test-topology-key');

          await page.getByTestId('text-input-key').fill('db-node');

          await page.getByTestId('select-operator-button').click();
          await page
            .getByRole('option', { name: 'exists', exact: true })
            .click();

          // Request to update PSP with new rule
          const addRuleResp = page.waitForResponse(
            (resp) =>
              resp.request().method() === 'PUT' &&
              resp
                .url()
                .includes(
                  `/v1/pod-scheduling-policies/${customPGPolicyName}`
                ) &&
              resp.status() === 200
          );

          await page.getByTestId('form-dialog-add').click();
          await addRuleResp;

          const row = page
            .locator('.MuiTableRow-root')
            .filter({ hasText: 'DB Node' });
          await expect(row).toBeVisible();
          await expect(row.getByText('Pod Anti-Affinity')).toBeVisible();
          await expect(row.getByText('Preferred: 1')).toBeVisible();
          await expect(
            row.getByText('test-topology-key | db-node | Exists')
          ).toBeVisible();
        });

        await test.step('Add rule for PG Bouncer', async () => {
          await page.getByTestId('add-rule-button').click();

          await page.getByTestId('select-component-button').click();
          await page
            .getByRole('option', { name: 'PG Bouncer', exact: true })
            .click();
          await page.getByTestId('toggle-button-preferred').click();

          await page
            .getByTestId('text-input-topology-key')
            .fill('test-topology-key');

          await page.getByTestId('text-input-key').fill('pg-bouncer');

          await page.getByTestId('select-operator-button').click();
          await page
            .getByRole('option', { name: 'exists', exact: true })
            .click();

          // Request to update PSP with new rule
          const addRuleResp = page.waitForResponse(
            (resp) =>
              resp.request().method() === 'PUT' &&
              resp
                .url()
                .includes(
                  `/v1/pod-scheduling-policies/${customPGPolicyName}`
                ) &&
              resp.status() === 200
          );

          await page.getByTestId('form-dialog-add').click();
          await addRuleResp;

          const row = page
            .locator('.MuiTableRow-root')
            .filter({ hasText: 'PG Bouncer' });
          await expect(row).toBeVisible();
          await expect(row.getByText('Pod Anti-Affinity')).toBeVisible();
          await expect(row.getByText('Preferred: 1')).toBeVisible();
          await expect(
            row.getByText('test-topology-key | pg-bouncer | Exists')
          ).toBeVisible();
        });
      });
    });

    // TODO: add test to check policy exists in DB creation wizard

    test('Show custom PG policy on DB overview', async ({ page, request }) => {
      const dbClusterName = customPGPolicyName,
        namespace = EVEREST_CI_NAMESPACES.PG_ONLY;

      try {
        await test.step(`Create ${dbClusterName} DB cluster`, async () => {
          await createDbClusterFn(
            request,
            {
              dbName: dbClusterName,
              dbType: DbType.Postresql,
              numberOfNodes: '1',
              cpu: 1,
              memory: 1,
              proxyCpu: 0.5,
              proxyMemory: 0.8,
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

        await test.step(`Open DB cluster ${dbClusterName} overview page`, async () => {
          await goToUrl(page, '/databases');
          await expect(page.getByText(dbClusterName)).toBeVisible({
            timeout: TIMEOUTS.TenSeconds,
          });
          await findDbAndClickRow(page, dbClusterName);

          // check that policy is not set
          await expect(
            page.getByTestId('pod-scheduling policy-overview-section-row')
          ).toHaveText('Pod scheduling policyDisabled');
        });

        await test.step('Assign custom policy', async () => {
          const getPspResp = page.waitForResponse(
            (resp) =>
              resp.request().method() === 'GET' &&
              resp
                .url()
                .includes(
                  '/v1/pod-scheduling-policies?engineType=postgresql&hasRules=true'
                ) &&
              resp.status() === 200
          );

          // assign policy
          await page.getByTestId('edit-advanced-configuration-db-btn').click();

          // wait for requests to finish
          await getPspResp;

          await page
            .getByTestId('switch-input-pod-scheduling-policy-enabled')
            .getByRole('checkbox')
            .check();
          await page.getByTestId('select-pod-scheduling-policy-button').click();
          const pspOptions = page.getByRole('option');
          // check that custom policy exists in list
          await expect(
            pspOptions.filter({ hasText: customPGPolicyName })
          ).toBeVisible();
          await pspOptions.filter({ hasText: customPGPolicyName }).click();

          await page.getByTestId('form-dialog-save').click();

          // check that policy is assigned
          await expect(
            page.getByTestId('pod-scheduling policy-overview-section-row')
          ).toHaveText(`Pod scheduling policy${customPGPolicyName}`);
        });

        await test.step('Unassign custom policy', async () => {
          // unassign policy
          await page.getByTestId('edit-advanced-configuration-db-btn').click();
          await page
            .getByTestId('switch-input-pod-scheduling-policy-enabled')
            .getByRole('checkbox')
            .uncheck();
          await page.getByTestId('form-dialog-save').click();

          // check that policy is not set
          await expect(
            page.getByTestId('pod-scheduling policy-overview-section-row')
          ).toHaveText('Pod scheduling policyDisabled');
        });
      } finally {
        await deleteDbClusterFn(request, dbClusterName, namespace);
      }
    });
  });

  test.describe.serial('Custom PSMDB Policy', () => {
    const customPSMDBPolicyName = limitedSuffixedName('pr-set-psp-psmdb');

    test.afterAll(async ({ request }) => {
      await expect(async () => {
        await deletePodSchedulingPolicy(request, customPSMDBPolicyName);
      }).toPass({
        intervals: [1000],
        timeout: TIMEOUTS.ThirtySeconds,
      });
    });

    test('Create policy for PSMDB', async ({ page }) => {
      await goToUrl(page, '/settings/policies/pod-scheduling');

      await test.step(`Create policy ${customPSMDBPolicyName} for PSMDB`, async () => {
        await expect(page.getByTestId('add-policy')).toBeVisible();
        await page.getByTestId('add-policy').click();
        await page.waitForLoadState('load', {
          timeout: TIMEOUTS.ThirtySeconds,
        });

        await expect(page.getByTestId('text-input-name')).toBeVisible();
        await page.getByTestId('text-input-name').fill(customPSMDBPolicyName);

        await expect(page.getByTestId('select-type-button')).toBeVisible();
        await page.getByTestId('select-type-button').click();
        await page
          .getByRole('option', {
            name: technologyMap[DbEngineType.PSMDB],
            exact: true,
          })
          .click();

        // Request to create PSP
        const createPspResp = page.waitForResponse(
          (resp) =>
            resp.request().method() === 'POST' &&
            resp.url().includes('/v1/pod-scheduling-policies') &&
            resp.status() === 200
        );

        await page.getByTestId('form-dialog-create').click();
        await createPspResp;
      });

      await test.step('Create rules', async () => {
        await test.step('Add rule for DB Node', async () => {
          await page.getByTestId('AddIcon').click();

          await page.getByTestId('select-component-button').click();
          await page
            .getByRole('option', { name: 'DB Node', exact: true })
            .click();
          await page.getByTestId('toggle-button-preferred').click();

          await page
            .getByTestId('text-input-topology-key')
            .fill('test-topology-key');

          await page.getByTestId('text-input-key').fill('db-node');

          await page.getByTestId('select-operator-button').click();
          await page
            .getByRole('option', { name: 'exists', exact: true })
            .click();

          // Request to update PSP with new rule
          const addRuleResp = page.waitForResponse(
            (resp) =>
              resp.request().method() === 'PUT' &&
              resp
                .url()
                .includes(
                  `/v1/pod-scheduling-policies/${customPSMDBPolicyName}`
                ) &&
              resp.status() === 200
          );

          await page.getByTestId('form-dialog-add').click();
          await addRuleResp;

          const row = page
            .locator('.MuiTableRow-root')
            .filter({ hasText: 'DB Node' });
          await expect(row).toBeVisible();
          await expect(row.getByText('Pod Anti-Affinity')).toBeVisible();
          await expect(row.getByText('Preferred: 1')).toBeVisible();
          await expect(
            row.getByText('test-topology-key | db-node | Exists')
          ).toBeVisible();
        });

        await test.step('Add rule for Config Server', async () => {
          await page.getByTestId('add-rule-button').click();

          await page.getByTestId('select-component-button').click();
          await page
            .getByRole('option', { name: 'Config Server', exact: true })
            .click();
          await page.getByTestId('toggle-button-preferred').click();

          await page
            .getByTestId('text-input-topology-key')
            .fill('test-topology-key');

          await page.getByTestId('text-input-key').fill('config-server');

          await page.getByTestId('select-operator-button').click();
          await page
            .getByRole('option', { name: 'exists', exact: true })
            .click();

          // Request to update PSP with new rule
          const addRuleResp = page.waitForResponse(
            (resp) =>
              resp.request().method() === 'PUT' &&
              resp
                .url()
                .includes(
                  `/v1/pod-scheduling-policies/${customPSMDBPolicyName}`
                ) &&
              resp.status() === 200
          );

          await page.getByTestId('form-dialog-add').click();
          await addRuleResp;

          const row = page
            .locator('.MuiTableRow-root')
            .filter({ hasText: 'Config Server' });
          await expect(row).toBeVisible();
          await expect(row.getByText('Pod Anti-Affinity')).toBeVisible();
          await expect(row.getByText('Preferred: 1')).toBeVisible();
          await expect(
            row.getByText('test-topology-key | config-server | Exists')
          ).toBeVisible();
        });

        await test.step('Add rule for Router', async () => {
          await page.getByTestId('add-rule-button').click();

          await page.getByTestId('select-component-button').click();
          await page
            .getByRole('option', { name: 'Router', exact: true })
            .click();
          await page.getByTestId('toggle-button-preferred').click();

          await page
            .getByTestId('text-input-topology-key')
            .fill('test-topology-key');

          await page.getByTestId('text-input-key').fill('router');

          await page.getByTestId('select-operator-button').click();
          await page
            .getByRole('option', { name: 'exists', exact: true })
            .click();

          // Request to update PSP with new rule
          const addRuleResp = page.waitForResponse(
            (resp) =>
              resp.request().method() === 'PUT' &&
              resp
                .url()
                .includes(
                  `/v1/pod-scheduling-policies/${customPSMDBPolicyName}`
                ) &&
              resp.status() === 200
          );

          await page.getByTestId('form-dialog-add').click();
          await addRuleResp;

          const row = page
            .locator('.MuiTableRow-root')
            .filter({ hasText: 'Router' });
          await expect(row).toBeVisible();
          await expect(row.getByText('Pod Anti-Affinity')).toBeVisible();
          await expect(row.getByText('Preferred: 1')).toBeVisible();
          await expect(
            row.getByText('test-topology-key | router | Exists')
          ).toBeVisible();
        });
      });
    });

    // TODO: add test to check policy exists in DB creation wizard

    test('Show custom PSMDB policy on DB overview', async ({
      page,
      request,
    }) => {
      const dbClusterName = customPSMDBPolicyName,
        namespace = EVEREST_CI_NAMESPACES.PSMDB_ONLY;

      try {
        await test.step(`Create ${dbClusterName} DB cluster`, async () => {
          await createDbClusterFn(
            request,
            {
              dbName: dbClusterName,
              dbType: DbType.Mongo,
              numberOfNodes: '1',
              cpu: 1,
              memory: 1,
              proxyCpu: 0.5,
              proxyMemory: 0.8,
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

        await test.step(`Open DB cluster ${dbClusterName} overview page`, async () => {
          await goToUrl(page, '/databases');
          await expect(page.getByText(dbClusterName)).toBeVisible({
            timeout: TIMEOUTS.TenSeconds,
          });
          await findDbAndClickRow(page, dbClusterName);

          // check that policy is not set
          await expect(
            page.getByTestId('pod-scheduling policy-overview-section-row')
          ).toHaveText('Pod scheduling policyDisabled');
        });

        await test.step('Assign custom policy', async () => {
          const getPspResp = page.waitForResponse(
            (resp) =>
              resp.request().method() === 'GET' &&
              resp
                .url()
                .includes(
                  '/v1/pod-scheduling-policies?engineType=psmdb&hasRules=true'
                ) &&
              resp.status() === 200
          );

          // assign policy
          await page.getByTestId('edit-advanced-configuration-db-btn').click();

          // wait for requests to finish
          await getPspResp;

          await page
            .getByTestId('switch-input-pod-scheduling-policy-enabled')
            .getByRole('checkbox')
            .check();
          await page.getByTestId('select-pod-scheduling-policy-button').click();
          const pspOptions = page.getByRole('option');
          // check that custom policy exists in list
          await expect(
            pspOptions.filter({ hasText: customPSMDBPolicyName })
          ).toBeVisible();
          await pspOptions.filter({ hasText: customPSMDBPolicyName }).click();

          await page.getByTestId('form-dialog-save').click();

          // check that policy is assigned
          await expect(
            page.getByTestId('pod-scheduling policy-overview-section-row')
          ).toHaveText(`Pod scheduling policy${customPSMDBPolicyName}`);
        });

        await test.step('Unassign custom policy', async () => {
          // unassign policy
          await page.getByTestId('edit-advanced-configuration-db-btn').click();
          await page
            .getByTestId('switch-input-pod-scheduling-policy-enabled')
            .getByRole('checkbox')
            .uncheck();
          await page.getByTestId('form-dialog-save').click();

          // check that policy is not set
          await expect(
            page.getByTestId('pod-scheduling policy-overview-section-row')
          ).toHaveText('Pod scheduling policyDisabled');
        });
      } finally {
        await deleteDbClusterFn(request, dbClusterName, namespace);
      }
    });
  });

  test.describe.serial('Custom PXC Policy', () => {
    const customPXCPolicyName = limitedSuffixedName('pr-set-psp-pxc');

    test.afterAll(async ({ request }) => {
      await expect(async () => {
        await deletePodSchedulingPolicy(request, customPXCPolicyName);
      }).toPass({
        intervals: [1000],
        timeout: TIMEOUTS.ThirtySeconds,
      });
    });

    test('Create policy for PXC', async ({ page }) => {
      await goToUrl(page, '/settings/policies/pod-scheduling');

      await test.step(`Create policy ${customPXCPolicyName} for PXC`, async () => {
        await expect(page.getByTestId('add-policy')).toBeVisible();
        await page.getByTestId('add-policy').click();
        await page.waitForLoadState('load', {
          timeout: TIMEOUTS.ThirtySeconds,
        });

        await expect(page.getByTestId('text-input-name')).toBeVisible();
        await page.getByTestId('text-input-name').fill(customPXCPolicyName);

        await expect(page.getByTestId('select-type-button')).toBeVisible();
        await page.getByTestId('select-type-button').click();
        await page
          .getByRole('option', {
            name: technologyMap[DbEngineType.PXC],
            exact: true,
          })
          .click();

        // Request to create PSP
        const createPspResp = page.waitForResponse(
          (resp) =>
            resp.request().method() === 'POST' &&
            resp.url().includes('/v1/pod-scheduling-policies') &&
            resp.status() === 200
        );

        await page.getByTestId('form-dialog-create').click();
        await createPspResp;
      });

      await test.step('Create rules', async () => {
        await test.step('Add rule for DB Node', async () => {
          await page.getByTestId('AddIcon').click();

          await page.getByTestId('select-component-button').click();
          await page
            .getByRole('option', { name: 'DB Node', exact: true })
            .click();
          await page.getByTestId('toggle-button-preferred').click();

          await page
            .getByTestId('text-input-topology-key')
            .fill('test-topology-key');

          await page.getByTestId('text-input-key').fill('db-node');

          await page.getByTestId('select-operator-button').click();
          await page
            .getByRole('option', { name: 'exists', exact: true })
            .click();

          // Request to update PSP with new rule
          const addRuleResp = page.waitForResponse(
            (resp) =>
              resp.request().method() === 'PUT' &&
              resp
                .url()
                .includes(
                  `/v1/pod-scheduling-policies/${customPXCPolicyName}`
                ) &&
              resp.status() === 200
          );

          await page.getByTestId('form-dialog-add').click();
          await addRuleResp;

          const row = page
            .locator('.MuiTableRow-root')
            .filter({ hasText: 'DB Node' });
          await expect(row).toBeVisible();
          await expect(row.getByText('Pod Anti-Affinity')).toBeVisible();
          await expect(row.getByText('Preferred: 1')).toBeVisible();
          await expect(
            row.getByText('test-topology-key | db-node | Exists')
          ).toBeVisible();
        });

        await test.step('Add rule for Proxy', async () => {
          await page.getByTestId('add-rule-button').click();

          await page.getByTestId('select-component-button').click();
          await page
            .getByRole('option', { name: 'Proxy', exact: true })
            .click();
          await page.getByTestId('toggle-button-preferred').click();

          await page
            .getByTestId('text-input-topology-key')
            .fill('test-topology-key');

          await page.getByTestId('text-input-key').fill('proxy');

          await page.getByTestId('select-operator-button').click();
          await page
            .getByRole('option', { name: 'exists', exact: true })
            .click();

          // Request to update PSP with new rule
          const addRuleResp = page.waitForResponse(
            (resp) =>
              resp.request().method() === 'PUT' &&
              resp
                .url()
                .includes(
                  `/v1/pod-scheduling-policies/${customPXCPolicyName}`
                ) &&
              resp.status() === 200
          );

          await page.getByTestId('form-dialog-add').click();
          await addRuleResp;

          const row = page
            .locator('.MuiTableRow-root')
            .filter({ hasText: 'Proxy' });
          await expect(row).toBeVisible();
          await expect(row.getByText('Pod Anti-Affinity')).toBeVisible();
          await expect(row.getByText('Preferred: 1')).toBeVisible();
          await expect(
            row.getByText('test-topology-key | proxy | Exists')
          ).toBeVisible();
        });
      });
    });

    // TODO: add test to check policy exists in DB creation wizard

    test('Show custom PXC policy on DB overview', async ({ page, request }) => {
      const dbClusterName = customPXCPolicyName,
        namespace = EVEREST_CI_NAMESPACES.PXC_ONLY;

      try {
        await test.step(`Create ${dbClusterName} DB cluster`, async () => {
          await createDbClusterFn(
            request,
            {
              dbName: dbClusterName,
              dbType: DbType.Mysql,
              numberOfNodes: '1',
              cpu: 1,
              memory: 1,
              proxyCpu: 0.6,
              proxyMemory: 0.8,
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

        await test.step(`Open DB cluster ${dbClusterName} overview page`, async () => {
          await goToUrl(page, '/databases');
          await expect(page.getByText(dbClusterName)).toBeVisible({
            timeout: TIMEOUTS.TenSeconds,
          });
          await findDbAndClickRow(page, dbClusterName);

          // check that policy is not set
          await expect(
            page.getByTestId('pod-scheduling policy-overview-section-row')
          ).toHaveText('Pod scheduling policyDisabled');
        });

        await test.step('Assign custom policy', async () => {
          const getPspResp = page.waitForResponse(
            (resp) =>
              resp.request().method() === 'GET' &&
              resp
                .url()
                .includes(
                  '/v1/pod-scheduling-policies?engineType=pxc&hasRules=true'
                ) &&
              resp.status() === 200
          );

          // assign policy
          await page.getByTestId('edit-advanced-configuration-db-btn').click();

          // wait for requests to finish
          await getPspResp;

          await page
            .getByTestId('switch-input-pod-scheduling-policy-enabled')
            .getByRole('checkbox')
            .check();
          await page.getByTestId('select-pod-scheduling-policy-button').click();
          const pspOptions = page.getByRole('option');
          // check that custom policy exists in list
          await expect(
            pspOptions.filter({ hasText: customPXCPolicyName })
          ).toBeVisible();
          await pspOptions.filter({ hasText: customPXCPolicyName }).click();

          await page.getByTestId('form-dialog-save').click();

          // check that policy is assigned
          await expect(
            page.getByTestId('pod-scheduling policy-overview-section-row')
          ).toHaveText(`Pod scheduling policy${customPXCPolicyName}`);
        });

        await test.step('Unassign custom policy', async () => {
          // unassign policy
          await page.getByTestId('edit-advanced-configuration-db-btn').click();
          await page
            .getByTestId('switch-input-pod-scheduling-policy-enabled')
            .getByRole('checkbox')
            .uncheck();
          await page.getByTestId('form-dialog-save').click();

          // check that policy is not set
          await expect(
            page.getByTestId('pod-scheduling policy-overview-section-row')
          ).toHaveText('Pod scheduling policyDisabled');
        });
      } finally {
        await deleteDbClusterFn(request, dbClusterName, namespace);
      }
    });
  });
});
