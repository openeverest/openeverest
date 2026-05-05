import { APIRequestContext, expect, Page } from '@playwright/test';
import { getTokenFromLocalStorage } from './localStorage';

export const POD_SCHEDULING_POLICIES_URL =
  '/settings/policies/details/pod-scheduling';

export const createPodSchedulingPolicy = async (
  request: APIRequestContext,
  name: string,
  dbType: string
) => {
  const response = await request.post(`/v1/pod-scheduling-policies`, {
    data: {
      metadata: {
        name,
      },
      spec: {
        engineType: dbType,
      },
    },
    headers: {
      Authorization: `Bearer ${await getTokenFromLocalStorage()}`,
    },
  });

  expect(response.ok()).toBeTruthy();
};

export const deletePodSchedulingPolicy = async (
  request: APIRequestContext,
  name: string
) => {
  const response = await request.delete(`/v1/pod-scheduling-policies/${name}`, {
    headers: {
      Authorization: `Bearer ${await getTokenFromLocalStorage()}`,
    },
  });

  expect(response.ok()).toBeTruthy();
};

export const getDefaultPodSchedulingPolicyNameForDbType = (
  dbType: string
): string => {
  switch (dbType) {
    case 'psmdb':
      return 'everest-default-mongodb';
    case 'pxc':
      return 'everest-default-mysql';
    case 'postgresql':
      return 'everest-default-postgresql';
    default:
      throw new Error(`Unknown database type: ${dbType}`);
  }
};

export const createPodSchedulingPolicyWithValues = async (
  page: Page,
  policyName: string,
  dbType: string
) => {
  const matchingPolicyType =
    dbType === 'psmdb' ? 'MongoDB' : dbType === 'pxc' ? 'MySQL' : 'PostgreSQL';
  await page.goto(POD_SCHEDULING_POLICIES_URL);
  await page.getByTestId('add-policy').waitFor();
  await page.getByTestId('add-policy').click();
  await page.getByTestId('text-input-name').fill(policyName);
  await page.getByTestId('select-type-button').click();
  await page
    .getByRole('option', { name: matchingPolicyType, exact: true })
    .click();
  await page.getByTestId('form-dialog-create').click();
};

export const createRuleForPodSchedulingPolicyWithValues = async (
  page: Page,
  policyName: string,
  component: 'proxy' | 'engine' | 'configServer',
  type: 'podAntiAffinity' | 'podAffinity' | 'nodeAffinity',
  priority: 'required' | 'preferred',
  weight: number = 1,
  topologyKey: string = 'kubernetes.io/hostname',
  key?: string,
  operator?: 'In' | 'NotIn' | 'Exists' | 'DoesNotExist',
  values?: string
) => {
  await page.goto(POD_SCHEDULING_POLICIES_URL);
  await page.getByRole('table').waitFor();
  await page
    .locator('.MuiTableRow-root')
    .filter({ hasText: policyName })
    .click();
  await page.getByRole('table').waitFor();
  await page.getByRole('button', { name: 'Add rule' }).click();

  await page.getByTestId('select-component-button').click();
  await page.getByTestId(component).click();

  await page.getByTestId('select-type-button').click();
  await page.getByTestId(type).click();

  await page.getByTestId(`toggle-button-${priority}`).click();

  if (type !== 'nodeAffinity') {
    await page.getByTestId('text-input-topology-key').fill(topologyKey);
  }

  if (priority === 'preferred') {
    await page.getByTestId('text-input-weight').fill(weight.toString());
  }

  if (key) {
    await page.getByTestId('text-input-key').fill(key);
  }

  if (operator) {
    await page.getByTestId('select-operator-button').click();
    await page.getByTestId(operator).click();
  }

  if (values) {
    await page.getByTestId('text-input-values').fill(values);
  }

  await page.getByTestId('form-dialog-add').click();
};
