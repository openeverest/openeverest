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

import { execSync } from 'child_process';
import {
  APIRequestContext,
  expect,
  Locator,
  Page,
  request as apiRequest,
  test,
} from '@playwright/test';
import {
  EVEREST_CI_CLUSTER,
  EVEREST_CI_NAMESPACES,
  TIMEOUTS,
} from '@e2e/constants';
import { getCITokenFromLocalStorage } from '@e2e/utils/localStorage';
import {
  moveForward,
  openDbCreationForm,
  populateBasicInformation,
  populateEngineResources,
  submitWizard,
} from '@e2e/utils/db-wizard';
import { waitForStatus } from '@e2e/utils/table';
import { ensureLoggedInOnDatabasesPage } from '@e2e/utils/db-clusters-list';

const PXC_PROVIDER_NAME = 'percona-xtradb-cluster';
const namespace = EVEREST_CI_NAMESPACES.EVEREST_UI;
const NODE_LABEL_KEY = 'e2e.openeverest.io/pod-scheduling';
const NODE_LABEL_VALUE = 'affinity';
const MISSING_NODE_LABEL_KEY = 'e2e.openeverest.io/no-such-node';
const HOSTNAME_TOPOLOGY_KEY = 'kubernetes.io/hostname';

type Condition = { key: string; operator: string; values?: string[] };

interface Pod {
  spec: { nodeName?: string };
  status: { conditions?: { type: string; reason?: string }[] };
}

type RuleGroup = {
  type?: 'Pod affinity' | 'Pod anti-affinity';
  weight?: number;
  topologyKey?: string;
  condition: Condition;
};

// Matches the operator labels of the rule editor (AffinityOperatorValue).
const OPERATOR_LABELS: Record<string, string> = {
  In: 'in',
  Exists: 'exists',
};

const instancePath = (name: string) =>
  `/v1/clusters/${EVEREST_CI_CLUSTER}/namespaces/${namespace}/instances/${name}`;

const authHeaders = async () => ({
  Authorization: `Bearer ${await getCITokenFromLocalStorage()}`,
});

const kubectlJson = (args: string) =>
  JSON.parse(execSync(`kubectl ${args} -o json`).toString());

const getInstance = async (request: APIRequestContext, name: string) => {
  const response = await request.get(instancePath(name), {
    headers: await authHeaders(),
  });
  expect(response.ok()).toBeTruthy();
  return response.json();
};

const getPxcSpec = (name: string) =>
  kubectlJson(`-n ${namespace} get pxc ${name}`).spec;

const getPods = (name: string, component: string): Pod[] =>
  kubectlJson(
    `-n ${namespace} get pods -l app.kubernetes.io/instance=${name},app.kubernetes.io/component=${component}`
  ).items;

const deleteInstanceAndWait = async (
  request: APIRequestContext,
  name: string
) => {
  const response = await request.delete(instancePath(name), {
    headers: await authHeaders(),
  });
  expect([204, 404]).toContain(response.status());
  await expect
    .poll(
      async () =>
        (
          await request.get(instancePath(name), {
            headers: await authHeaders(),
          })
        ).status(),
      { timeout: TIMEOUTS.FiveMinutes, intervals: [2000, 5000] }
    )
    .toBe(404);
};

// Best-effort: serial mode skips the remaining tests after a failure, so the
// cluster must not outlive the suite and starve the shared release node.
const deleteInstanceQuietly = async (name: string) => {
  try {
    const ctx = await apiRequest.newContext({
      baseURL: process.env.EVEREST_URL || 'http://localhost:8080',
    });
    await ctx.delete(instancePath(name), { headers: await authHeaders() });
    await ctx.dispose();
  } catch {
    // Leftovers are removed by the next run's pre-create delete.
  }
};

const moveForwardTo = async (page: Page, target: Locator) => {
  for (let step = 0; step < 8; step += 1) {
    if (await target.isVisible().catch(() => false)) {
      return;
    }
    await moveForward(page);
  }
  await expect(target).toBeVisible();
};

const selectOption = async (page: Page, combobox: Locator, name: string) => {
  await combobox.click();
  await page.getByRole('option', { name, exact: true }).click();
};

// Adds one rule group through the editor dialog. Defaults of a new group:
// Node affinity, Required, a single condition.
const addRuleGroup = async (
  page: Page,
  scope: Locator,
  component: 'engine' | 'proxy',
  group: RuleGroup
) => {
  await scope.getByTestId(`scheduling-component-tab-${component}`).click();
  await scope.getByRole('button', { name: 'Add rule group' }).click();

  const dialog = page.getByTestId('affinity-group-form-dialog');
  await expect(dialog).toBeVisible();

  if (group.type) {
    await selectOption(
      page,
      dialog.getByRole('combobox', { name: 'Type' }),
      group.type
    );
  }
  if (group.weight !== undefined) {
    await dialog.getByTestId('toggle-button-preferred').click();
    await dialog
      .getByRole('spinbutton', { name: 'Weight' })
      .fill(String(group.weight));
  }
  if (group.topologyKey) {
    await dialog
      .getByRole('textbox', { name: 'Topology Key' })
      .fill(group.topologyKey);
  }

  const { key, operator, values } = group.condition;
  await dialog.getByRole('textbox', { name: 'Key', exact: true }).fill(key);
  await selectOption(
    page,
    dialog.getByRole('combobox', { name: 'Operator' }),
    OPERATOR_LABELS[operator]
  );
  if (values) {
    await dialog
      .getByRole('textbox', { name: 'Values' })
      .fill(values.join(', '));
  }

  await dialog.getByTestId('form-dialog-add').click();
  await expect(dialog).toBeHidden();
  await expect(scope.getByText(key, { exact: true })).toBeVisible();
};

// Creates a 1-node PXC (1 proxy) whose engine carries the given rule group.
const createPxcWithEngineRule = async (
  page: Page,
  name: string,
  group: RuleGroup
) => {
  await page.goto('/databases');
  await openDbCreationForm(page, PXC_PROVIDER_NAME);

  await populateBasicInformation(page, namespace, name, 'pxc', '', false, '');

  const replicas = page.getByTestId(
    'text-input-spec.components.engine.replicas'
  );
  await moveForwardTo(page, replicas);
  await replicas.fill('1');
  // The topology defaults (4 CPU / 8Gi / 100Gi per node) do not fit a CI runner.
  await populateEngineResources(page, 1, 1, 1);

  const proxyReplicas = page.getByTestId(
    'text-input-spec.components.proxy.replicas'
  );
  await moveForwardTo(page, proxyReplicas);
  await proxyReplicas.fill('1');

  const policySwitch = page.getByLabel('Enable Pod scheduling policy');
  await moveForwardTo(page, policySwitch);
  await policySwitch.check();

  await addRuleGroup(page, page.locator('main'), 'engine', group);
  await submitWizard(page);
};

// The stored session expires while a cluster comes up; re-login if needed.
const gotoOverview = async (page: Page, name: string) => {
  await ensureLoggedInOnDatabasesPage(page);
  await page.goto(`/databases/${namespace}/${name}/overview`);
};

const openAdvancedEditModal = async (page: Page, name: string) => {
  await gotoOverview(page, name);
  // An edit made before cluster-info loads leaves Save disabled for good.
  const clusterInfo = page.waitForResponse((response) =>
    response.url().endsWith('/v1/cluster-info')
  );
  await page.getByTestId('advanced-edit-button').click();
  await clusterInfo;
  const modal = page.getByTestId('form-dialog');
  await expect(modal).toBeVisible();
  return modal;
};

const saveSection = async (page: Page, modal: Locator) => {
  await modal.getByTestId('form-dialog-save').click();
  await expect(page.getByText('Section updated successfully')).toBeVisible();
  await expect(modal).toBeHidden();
};

const nodeAffinityTerm = (condition: Condition) => ({
  requiredDuringSchedulingIgnoredDuringExecution: {
    nodeSelectorTerms: [{ matchExpressions: [condition] }],
  },
});

test.describe.configure({ retries: 0 });
// Fail a stuck click in a minute instead of at the 30-minute test timeout.
test.use({ actionTimeout: TIMEOUTS.OneMinute });

test.describe.serial(
  'Pod scheduling policy: pin the engine to a node, then edit the proxy rules [pxc size 1]',
  { tag: '@release' },
  () => {
    test.describe.configure({ timeout: TIMEOUTS.ThirtyMinutes });

    const clusterName = 'pxc-1-affinity';
    const engineCondition: Condition = {
      key: NODE_LABEL_KEY,
      operator: 'In',
      values: [NODE_LABEL_VALUE],
    };
    const proxyAntiAffinity = {
      weight: 100,
      podAffinityTerm: {
        topologyKey: HOSTNAME_TOPOLOGY_KEY,
        labelSelector: {
          matchExpressions: [
            {
              key: 'app.kubernetes.io/component',
              operator: 'In',
              values: ['haproxy'],
            },
          ],
        },
      },
    };
    let targetNode: string;

    test.beforeAll(() => {
      targetNode = execSync(
        `kubectl get nodes -l '!node-role.kubernetes.io/control-plane' -o jsonpath='{.items[0].metadata.name}'`
      )
        .toString()
        .trim();
      expect(targetNode).toBeTruthy();
      execSync(
        `kubectl label node ${targetNode} ${NODE_LABEL_KEY}=${NODE_LABEL_VALUE} --overwrite`
      );
    });

    test.afterAll(async () => {
      await deleteInstanceQuietly(clusterName);
      if (targetNode) {
        execSync(`kubectl label node ${targetNode} ${NODE_LABEL_KEY}-`);
      }
    });

    test('Create a database whose engine requires the labelled node', async ({
      page,
      request,
    }) => {
      await deleteInstanceAndWait(request, clusterName);
      await createPxcWithEngineRule(page, clusterName, {
        condition: engineCondition,
      });

      await page.goto('/databases');
      await waitForStatus(page, clusterName, 'Ready', TIMEOUTS.FifteenMinutes);
    });

    test('The rule reaches the Instance and the PXC resource', async ({
      request,
    }) => {
      const instance = await getInstance(request, clusterName);
      expect(
        instance.spec.components.engine.schedulingPolicy.affinity.nodeAffinity
      ).toEqual(nodeAffinityTerm(engineCondition));

      expect(
        getPxcSpec(clusterName).pxc.affinity.advanced.nodeAffinity
      ).toEqual(nodeAffinityTerm(engineCondition));
    });

    test('Engine pods run only on the labelled node', async () => {
      const pods = getPods(clusterName, 'pxc');
      expect(pods.length).toBe(1);
      for (const pod of pods) {
        expect(pod.spec.nodeName).toBe(targetNode);
      }
    });

    test('Editing adds a preferred pod anti-affinity rule to the proxy', async ({
      page,
      request,
    }) => {
      const modal = await openAdvancedEditModal(page, clusterName);
      await expect(
        modal.getByLabel('Enable Pod scheduling policy')
      ).toBeChecked();
      await expect(
        modal.getByTestId('scheduling-component-tab-engine')
      ).toHaveText('engine (1)');

      await addRuleGroup(page, modal, 'proxy', {
        type: 'Pod anti-affinity',
        weight: proxyAntiAffinity.weight,
        topologyKey: HOSTNAME_TOPOLOGY_KEY,
        condition: {
          key: 'app.kubernetes.io/component',
          operator: 'In',
          values: ['haproxy'],
        },
      });
      await saveSection(page, modal);

      await expect
        .poll(
          async () =>
            (await getInstance(request, clusterName)).spec.components.proxy
              .schedulingPolicy?.affinity?.podAntiAffinity
              ?.preferredDuringSchedulingIgnoredDuringExecution,
          { timeout: TIMEOUTS.OneMinute }
        )
        .toEqual([proxyAntiAffinity]);

      const instance = await getInstance(request, clusterName);
      expect(
        instance.spec.components.engine.schedulingPolicy.affinity.nodeAffinity
      ).toEqual(nodeAffinityTerm(engineCondition));

      await expect
        .poll(
          () =>
            getPxcSpec(clusterName).haproxy.affinity?.advanced?.podAntiAffinity
              ?.preferredDuringSchedulingIgnoredDuringExecution,
          { timeout: TIMEOUTS.ThreeMinutes }
        )
        .toEqual([proxyAntiAffinity]);
    });

    test('The overview shows both rules', async ({ page }) => {
      await gotoOverview(page, clusterName);
      const card = page.getByTestId('advanced-details');
      await card.locator('[data-testid$="-toggle"]').click();

      await expect(
        card.getByTestId('scheduling-component-tab-engine')
      ).toHaveText('engine (1)');
      await card.getByTestId('scheduling-component-tab-engine').click();
      await expect(
        card.getByText(NODE_LABEL_KEY, { exact: true })
      ).toBeVisible();

      await card.getByTestId('scheduling-component-tab-proxy').click();
      await expect(card.getByText(/pod anti-affinity/i)).toBeVisible();
      await expect(
        card.getByText('app.kubernetes.io/component', { exact: true })
      ).toBeVisible();
    });
  }
);

test.describe.serial(
  'Pod scheduling policy: an unschedulable rule is reported and can be undone [pxc size 1]',
  { tag: '@release' },
  () => {
    test.describe.configure({ timeout: TIMEOUTS.ThirtyMinutes });

    const clusterName = 'pxc-1-unschedulable';

    test.afterAll(async () => {
      await deleteInstanceQuietly(clusterName);
    });

    test('Create a database whose engine requires a node that does not exist', async ({
      page,
      request,
    }) => {
      await deleteInstanceAndWait(request, clusterName);
      await createPxcWithEngineRule(page, clusterName, {
        condition: { key: MISSING_NODE_LABEL_KEY, operator: 'Exists' },
      });

      await page.goto('/databases');
      await expect(
        page.getByRole('row').filter({ hasText: clusterName })
      ).toBeVisible({ timeout: TIMEOUTS.OneMinute });
    });

    test('Engine pods stay Pending and the UI warns about them', async ({
      page,
    }) => {
      await expect
        .poll(
          () =>
            getPods(clusterName, 'pxc').some((pod) =>
              pod.status.conditions?.some(
                (c) => c.type === 'PodScheduled' && c.reason === 'Unschedulable'
              )
            ),
          { timeout: TIMEOUTS.FiveMinutes, intervals: [5000] }
        )
        .toBe(true);

      await gotoOverview(page, clusterName);
      const alert = page.getByTestId('pods-alert');
      await expect(alert).toBeVisible({ timeout: TIMEOUTS.FiveMinutes });
      await expect(alert).toContainText('Some pods cannot be scheduled');
      await expect(alert).toContainText(/node affinity/i);

      await ensureLoggedInOnDatabasesPage(page);
      await page.getByTestId(`${clusterName}-status`).hover();
      await expect(page.getByRole('tooltip')).toHaveText(
        'Some pods need attention.'
      );
    });

    test('Turning the policy off removes the rule and the database recovers', async ({
      page,
      request,
    }) => {
      const modal = await openAdvancedEditModal(page, clusterName);
      const policySwitch = modal.getByLabel('Enable Pod scheduling policy');
      await expect(policySwitch).toBeChecked();
      await policySwitch.uncheck();
      await saveSection(page, modal);

      await expect
        .poll(
          async () =>
            (await getInstance(request, clusterName)).spec.components.engine
              .schedulingPolicy?.affinity,
          { timeout: TIMEOUTS.OneMinute }
        )
        .toBeUndefined();
      await expect
        .poll(() => getPxcSpec(clusterName).pxc.affinity?.advanced, {
          timeout: TIMEOUTS.ThreeMinutes,
        })
        .toBeUndefined();

      await ensureLoggedInOnDatabasesPage(page);
      await waitForStatus(page, clusterName, 'Ready', TIMEOUTS.FifteenMinutes);

      await gotoOverview(page, clusterName);
      await expect(page.getByTestId('pods-alert')).toBeHidden({
        timeout: TIMEOUTS.ThreeMinutes,
      });
    });
  }
);
