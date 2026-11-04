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

import { expect, Locator, Page, test } from '@playwright/test';
import { EVEREST_CI_NAMESPACES, TIMEOUTS } from '@e2e/constants';
import {
  moveForwardTo,
  openDbCreationForm,
  populateBasicInformation,
  populateEngineResources,
  submitWizard,
} from '@e2e/utils/db-wizard';
import { waitForStatus } from '@e2e/utils/table';
import { ensureLoggedInOnDatabasesPage } from '@e2e/utils/db-clusters-list';
import {
  deleteInstanceAndWait,
  deleteInstanceQuietly,
  getInstance,
} from '@e2e/utils/instance';
import {
  addLabelToK8sNode,
  getK8sNodes,
  getK8sResource,
  getK8sResourcesByLabel,
  removeLabelFromK8sNode,
} from '@e2e/utils/kubernetes';

const PXC_PROVIDER_NAME = 'percona-xtradb-cluster';
const namespace = EVEREST_CI_NAMESPACES.EVEREST_UI;
const NODE_LABEL_KEY = 'e2e.openeverest.io/pod-scheduling';
const NODE_LABEL_VALUE = 'affinity';
const MISSING_NODE_LABEL_KEY = 'e2e.openeverest.io/no-such-node';
const HOSTNAME_TOPOLOGY_KEY = 'kubernetes.io/hostname';

type Operator = 'In' | 'Exists';

type Condition = { key: string; operator: Operator; values?: string[] };

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
const OPERATOR_LABELS: Record<Operator, string> = {
  In: 'in',
  Exists: 'exists',
};

const getPxcSpec = async (name: string) =>
  (await getK8sResource('pxc', name, namespace)).spec;

const getPods = (name: string, component: string): Promise<Pod[]> =>
  getK8sResourcesByLabel(
    'pods',
    `app.kubernetes.io/instance=${name},app.kubernetes.io/component=${component}`,
    namespace
  );

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
  await ensureLoggedInOnDatabasesPage(page);
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

  await moveForwardTo(
    page,
    page.getByTestId('scheduling-component-tab-engine')
  );

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

    test.beforeAll(async () => {
      [targetNode] = await getK8sNodes();
      expect(targetNode).toBeTruthy();
      await addLabelToK8sNode(
        targetNode,
        NODE_LABEL_KEY,
        NODE_LABEL_VALUE,
        true
      );
    });

    // Serial mode skips the rest after a failure; the cluster must not starve the shared node.
    test.afterAll(async () => {
      await deleteInstanceQuietly(
        test.info().project.use.baseURL,
        namespace,
        clusterName
      );
      if (targetNode) {
        await removeLabelFromK8sNode(targetNode, NODE_LABEL_KEY);
      }
    });

    test('Create an instance whose engine requires the labelled node', async ({
      page,
      request,
    }) => {
      await deleteInstanceAndWait(request, namespace, clusterName);
      await createPxcWithEngineRule(page, clusterName, {
        condition: engineCondition,
      });

      await page.goto('/databases');
      await waitForStatus(page, clusterName, 'Ready', TIMEOUTS.FifteenMinutes);
    });

    test('The rule reaches the Instance and the PXC resource', async ({
      request,
    }) => {
      const instance = await getInstance(request, namespace, clusterName);
      expect(
        instance.spec.components?.engine?.schedulingPolicy?.affinity
          ?.nodeAffinity
      ).toEqual(nodeAffinityTerm(engineCondition));

      expect(
        (await getPxcSpec(clusterName)).pxc?.affinity?.advanced?.nodeAffinity
      ).toEqual(nodeAffinityTerm(engineCondition));
    });

    test('Engine pods run only on the labelled node', async () => {
      const pods = await getPods(clusterName, 'pxc');
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
            (await getInstance(request, namespace, clusterName)).spec.components
              ?.proxy?.schedulingPolicy?.affinity?.podAntiAffinity
              ?.preferredDuringSchedulingIgnoredDuringExecution,
          { timeout: TIMEOUTS.OneMinute }
        )
        .toEqual([proxyAntiAffinity]);

      const instance = await getInstance(request, namespace, clusterName);
      expect(
        instance.spec.components?.engine?.schedulingPolicy?.affinity
          ?.nodeAffinity
      ).toEqual(nodeAffinityTerm(engineCondition));

      await expect
        .poll(
          async () =>
            (await getPxcSpec(clusterName)).haproxy?.affinity?.advanced
              ?.podAntiAffinity
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
      await deleteInstanceQuietly(
        test.info().project.use.baseURL,
        namespace,
        clusterName
      );
    });

    test('Create an instance whose engine requires a node that does not exist', async ({
      page,
      request,
    }) => {
      await deleteInstanceAndWait(request, namespace, clusterName);
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
          async () =>
            (await getPods(clusterName, 'pxc')).some((pod) =>
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

    test('Deleting the rule group removes the rule', async ({
      page,
      request,
    }) => {
      const modal = await openAdvancedEditModal(page, clusterName);
      await modal.getByTestId('scheduling-component-tab-engine').click();
      await modal
        .getByTestId('delete-editable-item-button-affinity-group-0')
        .click();
      await expect(
        modal.getByText(MISSING_NODE_LABEL_KEY, { exact: true })
      ).toBeHidden();
      await saveSection(page, modal);

      await expect
        .poll(
          async () =>
            (await getInstance(request, namespace, clusterName)).spec.components
              ?.engine?.schedulingPolicy?.affinity?.nodeAffinity,
          { timeout: TIMEOUTS.OneMinute }
        )
        .toBeUndefined();
      await expect
        .poll(
          async () =>
            (await getPxcSpec(clusterName)).pxc?.affinity?.advanced
              ?.nodeAffinity,
          { timeout: TIMEOUTS.ThreeMinutes }
        )
        .toBeUndefined();
    });

    // The Pending engine pod is never recreated:
    // https://github.com/openeverest/provider-percona-xtradb-cluster/issues/49
    test.fixme(
      'The instance recovers once the rule is gone',
      async ({ page }) => {
        await ensureLoggedInOnDatabasesPage(page);
        await waitForStatus(
          page,
          clusterName,
          'Ready',
          TIMEOUTS.FifteenMinutes
        );

        await gotoOverview(page, clusterName);
        await expect(page.getByTestId('pods-alert')).toBeHidden({
          timeout: TIMEOUTS.ThreeMinutes,
        });
      }
    );
  }
);
