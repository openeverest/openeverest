import { expect, test } from '@playwright/test';
import { mockEngines, MOCK_CLUSTER_NAME, mockClusters } from './utils';
import { getCITokenFromLocalStorage } from '@e2e/utils/localStorage';
import { getNamespacesFn } from '@e2e/utils/namespaces';
import { setRBACPermissionsK8S } from '@e2e/utils/rbac-cmd-line';

const { CI_USER: user } = process.env;

test.describe('Clusters RBAC', () => {
  let namespace = '';

  test.beforeAll(async ({ request }) => {
    await setRBACPermissionsK8S([['namespaces', 'read', '*']]);
    const token = await getCITokenFromLocalStorage();
    const namespaces = await getNamespacesFn(token, request);
    namespace = namespaces[0];
  });

  test('permitted cluster creation with present clusters', async ({ page }) => {
    await mockEngines(page, namespace);
    await setRBACPermissionsK8S([
      ['namespaces', 'read', namespace],
      ['database-engines', '*', `${namespace}/*`],
      ['database-clusters', '*', `${namespace}/*`],
    ]);
    await mockClusters(page, namespace);
    await page.goto('/databases');
    await expect(page.getByText('Create', { exact: true })).toBeVisible();
    await expect(page.getByText('Create', { exact: true })).not.toBeDisabled();
  });

  test('permitted cluster creation without present clusters', async ({
    page,
  }) => {
    await mockEngines(page, namespace);
    await setRBACPermissionsK8S([
      ['namespaces', 'read', namespace],
      ['database-engines', '*', `${namespace}/*`],
      ['database-clusters', '*', `${namespace}/*`],
    ]);
    await page.goto('/databases');
    await expect(page.getByText('Create', { exact: true })).toBeVisible();
    await expect(page.getByText('Create', { exact: true })).not.toBeDisabled();
  });

  test('not permitted cluster creation with present clusters', async ({
    page,
  }) => {
    await mockEngines(page, namespace);
    await mockClusters(page, namespace);
    await setRBACPermissionsK8S([
      ['namespaces', 'read', namespace],
      ['database-engines', '*', `${namespace}/*`],
      ['database-clusters', 'read', `${namespace}/*`],
    ]);
    await page.goto('/databases');
    await expect(page.getByText('Create', { exact: true })).not.toBeVisible();
  });

  test('not permitted cluster creation without present clusters', async ({
    page,
  }) => {
    await mockEngines(page, namespace);
    await setRBACPermissionsK8S([
      ['namespaces', 'read', namespace],
      ['database-engines', '*', `${namespace}/*`],
      ['database-clusters', 'read', `${namespace}/*`],
    ]);
    await page.goto('/databases');
    await expect(page.getByText('Create', { exact: true })).not.toBeVisible();
  });

  test('visible actions', async ({ page }) => {
    await mockEngines(page, namespace);
    await mockClusters(page, namespace);
    await setRBACPermissionsK8S([
      ['namespaces', 'read', namespace],
      ['database-engines', '*', `${namespace}/*`],
      ['database-clusters', '*', `${namespace}/${MOCK_CLUSTER_NAME}`],
    ]);
    await page.goto('/databases');
    await page.getByTestId('actions-menu-button').click();
    await expect(page.getByText('Delete')).toBeVisible();
    await expect(page.getByText('Suspend')).toBeVisible();
    await expect(page.getByText('Restart')).toBeVisible();

    await page.goto(`/databases/${namespace}/${MOCK_CLUSTER_NAME}`);
    await expect(
      page.getByTestId('edit-advanced-configuration-db-btn')
    ).toBeVisible();
    await expect(
      page.getByTestId('edit-advanced-configuration-db-btn')
    ).not.toBeDisabled();
    await expect(page.getByTestId('edit-resources-button')).toBeVisible();
    await expect(page.getByTestId('edit-resources-button')).not.toBeDisabled();
  });

  test('not visible actions', async ({ page }) => {
    await mockEngines(page, namespace);
    await mockClusters(page, namespace);
    await setRBACPermissionsK8S([
      ['namespaces', 'read', namespace],
      ['database-engines', '*', `${namespace}/*`],
      ['database-clusters', 'read', `${namespace}/${MOCK_CLUSTER_NAME}`],
    ]);
    await page.goto('/databases');
    await expect(page.getByTestId('actions-menu-button')).not.toBeVisible();
    await page.goto(`/databases/${namespace}/${MOCK_CLUSTER_NAME}`);
    await expect(
      page.getByTestId('edit-advanced-configuration-db-btn')
    ).not.toBeVisible();
    await expect(page.getByTestId('edit-resources-button')).toBeDisabled();
  });
});
