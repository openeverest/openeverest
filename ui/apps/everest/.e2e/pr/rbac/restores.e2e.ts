import { getCITokenFromLocalStorage } from '@e2e/utils/localStorage';
import { getNamespacesFn } from '@e2e/utils/namespaces';
import { setRBACPermissionsK8S } from '@e2e/utils/rbac-cmd-line';
import { expect, test } from '@playwright/test';
import {
  MOCK_CLUSTER_NAME,
  MOCK_MONITORING_DESTINATION_NAME,
  mockBackups,
  mockClusters,
} from './utils';

const { CI_USER: user } = process.env;

test.describe('Restores RBAC', () => {
  let namespace = '';
  test.beforeAll(async ({ request }) => {
    await setRBACPermissionsK8S([['namespaces', 'read', '*']]);
    const token = await getCITokenFromLocalStorage();
    const namespaces = await getNamespacesFn(token, request);
    namespace = namespaces[0];
  });

  test('Restore to same DB', async ({ page }) => {
    await setRBACPermissionsK8S([
      ['namespaces', 'read', namespace],
      ['database-engines', 'read', `${namespace}/*`],
      ['backup-storages', 'read', `${namespace}/*`],
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
    await page.goto('/databases');
    await expect(page.getByText(MOCK_CLUSTER_NAME)).toBeVisible();
    await page.getByTestId('actions-menu-button').click();
    await expect(page.getByText('Restore from a backup')).toBeVisible();
  });

  test('Create DB from backup', async ({ page }) => {
    await setRBACPermissionsK8S([
      ['namespaces', 'read', namespace],
      ['database-engines', 'read', `${namespace}/*`],
      ['database-clusters', 'read', `${namespace}/${MOCK_CLUSTER_NAME}`],
      ['database-clusters', 'create', `${namespace}/*`],
      ['backup-storages', 'read', `${namespace}/*`],
      ['database-cluster-backups', '*', `${namespace}/${MOCK_CLUSTER_NAME}`],
      [
        'monitoring-destinations',
        '*',
        `${namespace}/${MOCK_MONITORING_DESTINATION_NAME}`,
      ],
      ['database-cluster-restores', 'create', `${namespace}/*`],
      [
        'database-cluster-credentials',
        'read',
        `${namespace}/${MOCK_CLUSTER_NAME}`,
      ],
    ]);
    await mockClusters(page, namespace);
    await mockBackups(page, namespace);
    await page.goto('/databases');
    await expect(page.getByText(MOCK_CLUSTER_NAME)).toBeVisible();
    await page.getByTestId('actions-menu-button').click();
    await expect(page.getByText('Create DB from a backup')).toBeVisible();
  });

  [
    {
      permissionToRemove: 'database-cluster-credentials',
    },
    {
      permissionToRemove: 'database-cluster-restores',
    },
  ].forEach(({ permissionToRemove }) => {
    test(`Hide Restore to same DB if "${permissionToRemove}" permission is removed`, async ({
      page,
    }) => {
      await setRBACPermissionsK8S(
        //@ts-expect-error
        [
          ['namespaces', 'read', namespace],
          ['database-engines', 'read', `${namespace}/*`],
          ['backup-storages', 'read', `${namespace}/*`],
          ['database-clusters', '*', `${namespace}/${MOCK_CLUSTER_NAME}`],
          [
            'database-cluster-backups',
            'read',
            `${namespace}/${MOCK_CLUSTER_NAME}`,
          ],
          ['database-cluster-restores', 'create', `${namespace}/*`],
          [
            'database-cluster-credentials',
            'read',
            `${namespace}/${MOCK_CLUSTER_NAME}`,
          ],
        ].filter(([permission]) => permission !== permissionToRemove)
      );
      await mockClusters(page, namespace);
      await mockBackups(page, namespace);
      await page.goto('/databases');
      await expect(page.getByText(MOCK_CLUSTER_NAME)).toBeVisible();
      await page.getByTestId('actions-menu-button').click();
      await expect(page.getByText('Restore from a backup')).not.toBeVisible();
    });
  });

  [
    {
      permissionToRemove: 'database-cluster-credentials',
    },
    {
      permissionToRemove: 'database-cluster-restores',
    },
    {
      permissionToRemove: 'database-cluster-backups',
    },
  ].forEach(({ permissionToRemove }) => {
    test(`Hide Create DB from backup if "${permissionToRemove}" permission is removed`, async ({
      page,
    }) => {
      await setRBACPermissionsK8S(
        //@ts-expect-error
        [
          ['namespaces', 'read', namespace],
          ['database-engines', 'read', `${namespace}/*`],
          ['database-clusters', '*', `${namespace}/${MOCK_CLUSTER_NAME}`],
          ['database-clusters', 'create', `${namespace}/*`],
          ['backup-storages', 'read', `${namespace}/*`],
          [
            'database-cluster-backups',
            '*',
            `${namespace}/${MOCK_CLUSTER_NAME}`,
          ],
          ['database-cluster-restores', 'create', `${namespace}/*`],
          [
            'database-cluster-credentials',
            'read',
            `${namespace}/${MOCK_CLUSTER_NAME}`,
          ],
        ].filter(([permission]) => permission !== permissionToRemove)
      );
      await mockClusters(page, namespace);
      await mockBackups(page, namespace);
      await page.goto('/databases');
      await expect(page.getByText(MOCK_CLUSTER_NAME)).toBeVisible();
      await page.getByTestId('actions-menu-button').click();
      await expect(page.getByText('Create DB from a backup')).not.toBeVisible();
    });
  });

  test('Hide Create DB from backup if DB has schedules and not allowed to create backups', async ({
    page,
  }) => {
    await setRBACPermissionsK8S([
      ['namespaces', 'read', namespace],
      ['database-engines', 'read', `${namespace}/*`],
      ['database-clusters', 'read', `${namespace}/${MOCK_CLUSTER_NAME}`],
      ['database-clusters', 'create', `${namespace}/*`],
      ['backup-storages', 'read', `${namespace}/*`],
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
    await page.goto('/databases');

    await expect(page.getByText(MOCK_CLUSTER_NAME)).toBeVisible();
    await page.getByTestId('actions-menu-button').click();
    await expect(page.getByText('Create DB from a backup')).not.toBeVisible();
  });

  test('Show Create DB from backup if DB has no schedules, even if not allowed to create backups', async ({
    page,
  }) => {
    await setRBACPermissionsK8S([
      ['namespaces', 'read', namespace],
      ['database-engines', 'read', `${namespace}/*`],
      ['database-clusters', 'read', `${namespace}/${MOCK_CLUSTER_NAME}`],
      ['database-clusters', 'create', `${namespace}/*`],
      ['backup-storages', 'read', `${namespace}/*`],
      ['database-cluster-backups', 'read', `${namespace}/${MOCK_CLUSTER_NAME}`],
      [
        'monitoring-destinations',
        'read',
        `${namespace}/${MOCK_MONITORING_DESTINATION_NAME}`,
      ],
      ['database-cluster-restores', 'create', `${namespace}/*`],
      [
        'database-cluster-credentials',
        'read',
        `${namespace}/${MOCK_CLUSTER_NAME}`,
      ],
    ]);
    await mockClusters(page, namespace, { enableSchedules: false });
    await mockBackups(page, namespace);
    await page.goto('/databases');

    await expect(page.getByText(MOCK_CLUSTER_NAME)).toBeVisible();
    await page.getByTestId('actions-menu-button').click();
    await expect(page.getByText('Create DB from a backup')).toBeVisible();
  });

  test('Hide Create DB from backup if DB has monitoring enabled and not allowed to read monitoring', async ({
    page,
  }) => {
    await setRBACPermissionsK8S([
      ['namespaces', 'read', namespace],
      ['database-engines', 'read', `${namespace}/*`],
      ['database-clusters', 'read', `${namespace}/${MOCK_CLUSTER_NAME}`],
      ['database-clusters', 'create', `${namespace}/*`],
      ['backup-storages', 'read', `${namespace}/*`],
      [
        'database-cluster-backups',
        'create',
        `${namespace}/${MOCK_CLUSTER_NAME}`,
      ],
      ['database-cluster-restores', 'create', `${namespace}/*`],
      [
        'database-cluster-credentials',
        'read',
        `${namespace}/${MOCK_CLUSTER_NAME}`,
      ],
    ]);
    await mockClusters(page, namespace);
    await mockBackups(page, namespace);
    await page.goto('/databases');

    await expect(page.getByText(MOCK_CLUSTER_NAME)).toBeVisible();
    await page.getByTestId('actions-menu-button').click();
    await expect(page.getByText('Create DB from a backup')).not.toBeVisible();
  });

  test('Show Create DB from backup if DB has monitoring disabled, even if not allowed to read monitoring', async ({
    page,
  }) => {
    await setRBACPermissionsK8S([
      ['namespaces', 'read', namespace],
      ['database-engines', 'read', `${namespace}/*`],
      ['database-clusters', 'read', `${namespace}/${MOCK_CLUSTER_NAME}`],
      ['database-clusters', 'create', `${namespace}/*`],
      ['backup-storages', 'read', `${namespace}/*`],
      [
        'database-cluster-backups',
        'create',
        `${namespace}/${MOCK_CLUSTER_NAME}`,
      ],
      ['database-cluster-restores', 'create', `${namespace}/*`],
      [
        'database-cluster-credentials',
        'read',
        `${namespace}/${MOCK_CLUSTER_NAME}`,
      ],
    ]);
    await mockClusters(page, namespace, { enableMonitoring: false });
    await mockBackups(page, namespace);
    await page.goto('/databases');

    await expect(page.getByText(MOCK_CLUSTER_NAME)).toBeVisible();
    await page.getByTestId('actions-menu-button').click();
    await expect(page.getByText('Create DB from a backup')).toBeVisible();
  });
});
