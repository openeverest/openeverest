import { expect, Page, test } from '@playwright/test';
import { moveBack, moveForward, submitWizard } from '@e2e/utils/db-wizard';
import { findDbAndClickRow } from '@e2e/utils/db-clusters-list';
import { createDbClusterFn, deleteDbClusterFn } from '@e2e/utils/db-cluster';
import { selectDbEngine } from '../db-cluster/db-wizard/db-wizard-utils';
import { getTokenFromLocalStorage } from '@e2e/utils/localStorage';
import { STORAGE_STATE_FILE } from '@e2e/constants';

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

const PG_POLICY_NAME = 'policy-pg-test';
const PSMDB_POLICY_NAME = 'policy-psmdb-test';
const DB_CLUSTER_NAME = 'policy-test-cluster';

test.beforeAll(async ({ browser, request }) => {
  const context = await browser.newContext({
    storageState: STORAGE_STATE_FILE,
  });
  const page = await context.newPage();
  await page.goto(POD_SCHEDULING_POLICIES_URL);
  await page.getByTestId('add-policy').click();
  await page.getByTestId('text-input-name').fill(PG_POLICY_NAME);
  await page.getByTestId('select-type-button').click();
  await page.getByRole('option', { name: 'PostgreSQL', exact: true }).click();
  await page.getByTestId('form-dialog-create').click();
  await page.getByRole('button', { name: 'Back' }).click();
  await page.getByTestId('add-policy').click();
  await page.getByTestId('text-input-name').fill(PSMDB_POLICY_NAME);
  await page.getByTestId('select-type-button').click();
  await page.getByRole('option', { name: 'MongoDB', exact: true }).click();
  await page.getByTestId('form-dialog-create').click();

  await createDbClusterFn(request, {
    dbName: DB_CLUSTER_NAME,
    dbType: 'mongodb',
    numberOfNodes: '1',
    cpu: 1,
    memory: 1,
    proxyCpu: 0.5,
    proxyMemory: 0.8,
  });
});

export const POD_SCHEDULING_POLICIES_URL =
  '/settings/policies/details/pod-scheduling';

test.afterAll(async ({ request }) => {
  await deleteDbClusterFn(request, DB_CLUSTER_NAME);
  const promises = [
    request.delete(`/v1/pod-scheduling-policies/${PG_POLICY_NAME}`, {
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${await getTokenFromLocalStorage()}`,
      },
    }),
    request.delete(`/v1/pod-scheduling-policies/${PSMDB_POLICY_NAME}`, {
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${await getTokenFromLocalStorage()}`,
      },
    }),
  ];

  const response = await Promise.all(promises);
});

test.describe('Create rules', () => {
  test('Create rules for PG', async ({ page }) => {
    await page.goto(POD_SCHEDULING_POLICIES_URL);
    await page.getByRole('table').waitFor();
    await page
      .locator('.MuiTableRow-root')
      .filter({ hasText: PG_POLICY_NAME })
      .click();
    await page.getByRole('table').waitFor();
    await page.getByRole('button', { name: 'Add rule' }).click();
    await page.getByTestId('select-component-button').click();
    await page.getByRole('option', { name: 'PG Bouncer', exact: true }).click();
    await page.getByTestId('toggle-button-required').click();
    await page.getByTestId('text-input-topology-key').fill('my-topology-key');
    await page.getByTestId('text-input-key').fill('my-key');
    await page.getByTestId('select-operator-button').click();
    await expect(
      page.getByRole('option', { name: 'Config Server', exact: true })
    ).not.toBeVisible();
    await page.getByRole('option', { name: 'exists', exact: true }).click();
    await page.getByTestId('form-dialog-add').click();
    const row = page
      .locator('.MuiTableRow-root')
      .filter({ hasText: 'PG Bouncer' });
    await expect(row).toBeVisible();
    await expect(row.getByText('Pod Anti-Affinity')).toBeVisible();
    await expect(row.getByText('Required')).toBeVisible();
    await expect(
      row.getByText('my-topology-key | my-key | Exists')
    ).toBeVisible();
  });

  test('Create rules for Mongo', async ({ page }) => {
    await page.goto(POD_SCHEDULING_POLICIES_URL);
    await page.getByRole('table').waitFor();
    await page
      .locator('.MuiTableRow-root')
      .filter({ hasText: PSMDB_POLICY_NAME })
      .click();
    await page.getByRole('table').waitFor();
    await page.getByRole('button', { name: 'Add rule' }).click();
    await page.getByTestId('select-component-button').click();
    await page
      .getByRole('option', { name: 'Config Server', exact: true })
      .click();
    await page.getByTestId('text-input-weight').fill('10');
    await page.getByTestId('text-input-topology-key').fill('my-topology-key');
    await page.getByTestId('text-input-key').fill('my-key');
    await page.getByTestId('select-operator-button').click();
    await page.getByRole('option', { name: 'in', exact: true }).click();
    await page.getByTestId('text-input-values').fill('my-value');
    await page.getByTestId('form-dialog-add').click();
    const row = page
      .locator('.MuiTableRow-root')
      .filter({ hasText: 'Config Server' });
    await expect(row).toBeVisible();
    await expect(row.getByText('Pod Anti-Affinity')).toBeVisible();
    await expect(row.getByText('Preferred: 10')).toBeVisible();
    await expect(
      row.getByText('my-topology-key | my-key | In | my-value')
    ).toBeVisible();
  });

  test('Edit rules', async ({ page }) => {
    await page.goto(POD_SCHEDULING_POLICIES_URL);
    await page.getByRole('table').waitFor();
    await page
      .locator('.MuiTableRow-root')
      .filter({ hasText: PSMDB_POLICY_NAME })
      .click();
    await page.getByRole('table').waitFor();
    await page.getByRole('button', { name: 'Add rule' }).click();
    await page.getByTestId('form-dialog-add').click();
    const row = page
      .locator('.MuiTableRow-root')
      .filter({ hasText: 'DB Node' });
    await row.getByTestId('edit-rule-button').click();
    await page.getByTestId('text-input-key').fill('edit-rule-key');
    await page.getByTestId('select-operator-button').click();
    await page.getByRole('option', { name: 'exists', exact: true }).click();
    await page.getByTestId('form-dialog-save').click();
    await expect(
      row.getByText('kubernetes.io/hostname | edit-rule-key | Exists')
    ).toBeVisible();
  });

  test('Show policies on wizard', async ({ page }) => {
    await page.goto('/databases');
    await selectDbEngine(page, 'psmdb');
    await moveForward(page);
    await moveForward(page);
    await moveForward(page);
    await page
      .getByTestId('switch-input-pod-scheduling-policy-enabled')
      .waitFor();
    await page
      .getByTestId('switch-input-pod-scheduling-policy-enabled')
      .getByRole('checkbox')
      .uncheck();

    await moveBack(page);
    await moveForward(page);

    await expect(
      page.getByTestId('switch-input-pod-scheduling-policy-enabled')
    ).not.toBeChecked();

    await page
      .getByTestId('switch-input-pod-scheduling-policy-enabled')
      .getByRole('checkbox')
      .check();

    await page.getByTestId('select-pod-scheduling-policy-button').click();
    await expect(
      page.getByRole('option', { name: PSMDB_POLICY_NAME, exact: true })
    ).toBeVisible();
    await expect(
      page.getByRole('option', { name: PG_POLICY_NAME, exact: true })
    ).not.toBeVisible();
    await page
      .getByRole('option', { name: PSMDB_POLICY_NAME, exact: true })
      .click();
    await expect(
      page.getByText(`Pod scheduling policy: ${PSMDB_POLICY_NAME}`)
    ).toBeVisible();
  });

  test('Show policies on overview', async ({ page }) => {
    await page.goto('/databases');
    await findDbAndClickRow(page, DB_CLUSTER_NAME);
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
      .getByRole('option', { name: PSMDB_POLICY_NAME, exact: true })
      .click();
    await page.getByTestId('form-dialog-save').click();
    await expect(
      page.getByTestId('pod-scheduling policy-overview-section-row')
    ).toHaveText(`Pod scheduling policy${PSMDB_POLICY_NAME}`);
  });
});
