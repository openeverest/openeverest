import { getTokenFromLocalStorage } from '@e2e/utils/localStorage';
import { getNamespacesFn } from '@e2e/utils/namespaces';
import { setRBACPermissionsK8S } from '@e2e/utils/rbac-cmd-line';
import { expect, test } from '@playwright/test';
import {
  MOCK_CLUSTER_NAME,
  mockBackups,
  mockClusters,
  mockStorages,
} from './utils';

test.describe('Backups RBAC', () => {
  let namespace = '';
  test.beforeAll(async ({ request }) => {
    await setRBACPermissionsK8S([['namespaces', 'read', '*']]);
    const token = await getTokenFromLocalStorage();
    const namespaces = await getNamespacesFn(token, request);
    namespace = namespaces[0];
  });

  test('Hide Backups', async ({ page }) => {
    await setRBACPermissionsK8S([
      ['namespaces', 'read', namespace],
      ['database-engines', '*', `${namespace}/*`],
      ['backup-storages', '*', `${namespace}/*`],
      ['database-clusters', '*', `${namespace}/*`],
    ]);
    await mockClusters(page, namespace);
    await mockBackups(page, namespace);
    await mockStorages(page, namespace);
    await page.goto(`/databases/${namespace}/${MOCK_CLUSTER_NAME}/backups`);
    await expect(page.getByRole('table')).toBeVisible();
    const rows = page.locator('.MuiTableRow-root:not(.MuiTableRow-head)');
    expect(await rows.count()).toBe(0);
  });

  test('Show Backups', async ({ page }) => {
    await setRBACPermissionsK8S([
      ['namespaces', 'read', namespace],
      ['database-engines', '*', `${namespace}/*`],
      ['backup-storages', '*', `${namespace}/*`],
      ['database-clusters', '*', `${namespace}/*`],
      ['database-cluster-backups', 'read', `${namespace}/${MOCK_CLUSTER_NAME}`],
    ]);
    await mockClusters(page, namespace);
    await mockBackups(page, namespace);
    await mockStorages(page, namespace);
    await page.goto(`/databases/${namespace}/${MOCK_CLUSTER_NAME}/backups`);
    await expect(page.getByRole('table')).toBeVisible();
    await expect(page.getByTestId('row-actions-menu-button')).not.toBeVisible();
    const rows = page.locator('.MuiTableRow-root:not(.MuiTableRow-head)');
    expect(await rows.count()).toBe(1);
  });

  test('Delete backup', async ({ page }) => {
    await setRBACPermissionsK8S([
      ['namespaces', 'read', namespace],
      ['database-engines', '*', `${namespace}/*`],
      ['backup-storages', '*', `${namespace}/*`],
      ['database-clusters', '*', `${namespace}/*`],
      ['database-cluster-backups', 'read', `${namespace}/${MOCK_CLUSTER_NAME}`],
      [
        'database-cluster-backups',
        'delete',
        `${namespace}/${MOCK_CLUSTER_NAME}`,
      ],
    ]);
    await mockClusters(page, namespace);
    await mockBackups(page, namespace);
    await mockStorages(page, namespace);
    await page.goto(`/databases/${namespace}/${MOCK_CLUSTER_NAME}/backups`);
    await expect(page.getByTestId('row-actions-menu-button')).toBeVisible();
    await page.getByTestId('row-actions-menu-button').click();
    await expect(page.getByText('Delete')).toBeVisible();
    await expect(page.getByText('Restore to this DB')).not.toBeVisible();
    await expect(page.getByText('Create new DB')).not.toBeVisible();
  });

  test('Create on-demand backup', async ({ page }) => {
    await setRBACPermissionsK8S([
      ['namespaces', 'read', namespace],
      ['database-engines', '*', `${namespace}/*`],
      ['backup-storages', '*', `${namespace}/*`],
      ['database-clusters', 'read', `${namespace}/*`],
      [
        'database-cluster-backups',
        'create',
        `${namespace}/${MOCK_CLUSTER_NAME}`,
      ],
    ]);
    await mockClusters(page, namespace);
    await mockBackups(page, namespace);
    await mockStorages(page, namespace);
    await page.goto(`/databases/${namespace}/${MOCK_CLUSTER_NAME}/backups`);
    await expect(page.getByTestId('menu-button')).toBeVisible();
    await page.getByText('Create backup').click();
    await expect(page.getByText('Now', { exact: true })).toBeVisible();
    await expect(page.getByText('Schedule', { exact: true })).not.toBeVisible();
  });

  test('Create scheduled backup', async ({ page }) => {
    await setRBACPermissionsK8S([
      ['namespaces', 'read', namespace],
      ['database-engines', '*', `${namespace}/*`],
      ['backup-storages', '*', `${namespace}/*`],
      ['database-clusters', 'read', `${namespace}/*`],
      ['database-clusters', 'update', `${namespace}/*`],
      [
        'database-cluster-backups',
        'create',
        `${namespace}/${MOCK_CLUSTER_NAME}`,
      ],
    ]);
    await mockClusters(page, namespace);
    await mockBackups(page, namespace);
    await mockStorages(page, namespace);
    await page.goto(`/databases/${namespace}/${MOCK_CLUSTER_NAME}/backups`);
    await expect(page.getByTestId('menu-button')).toBeVisible();
    await page.getByText('Create backup').click();
    await expect(page.getByText('Schedule', { exact: true })).toBeVisible();
  test('TC-08: Minimal RBAC gating for Restore to New DB', async ({ page }) => {
    // Restricted user: lacks database-clusters create permission
    await setRBACPermissionsK8S([
      ['namespaces', 'read', namespace],
      ['database-engines', '*', `${namespace}/*`],
      ['backup-storages', '*', `${namespace}/*`],
      ['database-clusters', 'read', `${namespace}/${MOCK_CLUSTER_NAME}`],
      ['database-cluster-backups', 'read', `${namespace}/${MOCK_CLUSTER_NAME}`],
      ['database-cluster-restores', 'create', `${namespace}/*`],
      [
        'database-cluster-credentials',
        'read',
        `${namespace}/${MOCK_CLUSTER_NAME}`,
      ],
    ]);
    await mockClusters(page, namespace);
    await mockBackups(page, namespace);
    await mockStorages(page, namespace);
    await page.goto(`/databases/${namespace}/${MOCK_CLUSTER_NAME}/backups`);
    await expect(page.getByTestId('row-actions-menu-button')).toBeVisible();
    await page.getByTestId('row-actions-menu-button').click();
    await expect(page.getByText('Create new DB')).not.toBeVisible();

    // Authorized user: has required permissions
    await setRBACPermissionsK8S([
      ['namespaces', 'read', namespace],
      ['database-engines', '*', `${namespace}/*`],
      ['backup-storages', '*', `${namespace}/*`],
      ['database-clusters', 'read', `${namespace}/${MOCK_CLUSTER_NAME}`],
      ['database-clusters', 'create', `${namespace}/*`],
      ['database-cluster-backups', 'read', `${namespace}/${MOCK_CLUSTER_NAME}`],
      ['database-cluster-restores', 'create', `${namespace}/*`],
      [
        'database-cluster-credentials',
        'read',
        `${namespace}/${MOCK_CLUSTER_NAME}`,
      ],
    ]);
    await page.reload();
    await expect(page.getByTestId('row-actions-menu-button')).toBeVisible();
    await page.getByTestId('row-actions-menu-button').click();
    await expect(page.getByText('Create new DB')).toBeVisible();
  });
});
