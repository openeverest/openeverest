import { test, expect } from '@playwright/test';
import { getCITokenFromLocalStorage } from '@e2e/utils/localStorage';
import { getNamespacesFn } from '@e2e/utils/namespaces';
import { goToUrl } from '@e2e/utils/generic';

const generateMockEngineData = ({
  pg,
  mongo,
  mysql,
}: {
  [key in 'pg' | 'mysql' | 'mongo']: {
    status: string;
    version: string;
  };
}) => ({
  items: [
    {
      spec: {
        type: 'postgresql',
      },
      status: {
        status: pg.status,
        operatorVersion: pg.version,
        availableVersions: {
          engine: {},
        },
      },
      metadata: {
        name: 'percona-postgresql-operator',
      },
    },
    {
      spec: {
        type: 'pxc',
      },
      status: {
        status: mysql.status,
        operatorVersion: mysql.version,
        availableVersions: {
          engine: {},
        },
      },
      metadata: {
        name: 'percona-xtradb-cluster-operator',
      },
    },
    {
      spec: {
        type: 'psmdb',
      },
      status: {
        status: mongo.status,
        operatorVersion: mongo.version,
        availableVersions: {
          engine: {},
        },
      },
      metadata: {
        name: 'percona-server-mongodb-operator',
      },
    },
  ],
});

test.describe.parallel('Operator upgrades', () => {
  let namespaces = [];

  test.beforeAll(async ({ request }) => {
    const token = await getCITokenFromLocalStorage();
    namespaces = await getNamespacesFn(token, request);
  });

  test('show operator versions without any upgrade header above table', async ({
    page,
  }) => {
    await page.route(
      `/v1/namespaces/${namespaces[0]}/database-engines`,
      async (route) => {
        await route.fulfill({
          json: generateMockEngineData({
            pg: { status: 'installed', version: '2.3.1' },
            mongo: { status: 'installed', version: '1.15.0' },
            mysql: { status: 'installed', version: '1.13.0' },
          }),
        });
      }
    );
    await goToUrl(page, `/settings/namespaces/${namespaces[0]}`);
    await expect(page.getByText('v2.3.1')).toBeVisible();
    await expect(page.getByText('v1.15.0')).toBeVisible();
    await expect(page.getByText('v1.13.0')).toBeVisible();
    await expect(
      page.getByRole('button', { name: 'Upgrade Operators' })
    ).not.toBeVisible();
  });

  test('show new version available with upgrade button', async ({ page }) => {
    await page.route(
      `/v1/namespaces/${namespaces[0]}/database-engines`,
      async (route) => {
        await route.fulfill({
          json: generateMockEngineData({
            pg: { status: 'installed', version: '2.3.1' },
            mongo: {
              status: 'installed',
              version: '1.15.0',
            },
            mysql: { status: 'installed', version: '1.13.0' },
          }),
        });
      }
    );
    await page.route(
      `/v1/namespaces/${namespaces[0]}/database-engines/upgrade-plan`,
      async (route) => {
        await route.fulfill({
          json: {
            upgrades: [
              {
                name: 'percona-server-mongodb-operator',
                currentVersion: '1.15.0',
                targetVersion: '1.16.0',
              },
            ],
            pendingActions: [],
          },
        });
      }
    );
    await goToUrl(page, `/settings/namespaces/${namespaces[0]}`);
    await expect(
      page.getByRole('button', { name: 'Upgrade Operators' })
    ).toBeVisible();
    await expect(page.locator('tbody tr')).toHaveCount(1);
    await page.getByRole('button', { name: 'Upgrade Operators' }).click();
    await expect(
      page.getByText(
        `Are you sure you want to upgrade your operators in ${namespaces[0]}?`
      )
    ).toBeVisible();
    await expect(
      page.getByText('v1.15.0 will be upgraded to v1.16.0')
    ).toBeVisible();
  });

  test('Pending tasks', async ({ page }) => {
    await page.route(
      `/v1/namespaces/${namespaces[0]}/database-engines`,
      async (route) => {
        await route.fulfill({
          json: generateMockEngineData({
            pg: { status: 'installed', version: '2.3.1' },
            mongo: {
              status: 'installed',
              version: '1.15.0',
            },
            mysql: { status: 'installed', version: '1.13.0' },
          }),
        });
      }
    );
    await page.route(
      `/v1/namespaces/${namespaces[0]}/database-engines/upgrade-plan`,
      async (route) => {
        await route.fulfill({
          json: {
            upgrades: [
              {
                name: '',
                currentVersion: '1.15.0',
                targetVersion: '1.16.0',
              },
            ],
            pendingActions: [
              {
                name: 'db1',
                message: 'some message',
                pendingTask: 'notReady',
              },
              {
                name: 'db2',
                message: 'some message',
                pendingTask: 'ready',
              },
            ],
          },
        });
      }
    );
    await goToUrl(page, `/settings/namespaces/${namespaces[0]}`);
    await expect(
      page.getByRole('button', { name: 'Upgrade Operators' })
    ).toBeVisible();
    await expect(page.locator('tbody tr')).toHaveCount(2);
    await expect(
      page.getByRole('button', { name: 'Upgrade Operators' })
    ).toBeDisabled();
    await expect(
      page.getByText(
        'A new version of the operators is available.Start upgrading by performing all the pending tasks in the Actions column.'
      )
    ).toBeVisible();
  });

  test('Post-upgrade tasks', async ({ page }) => {
    await page.route(
      `/v1/namespaces/${namespaces[0]}/database-engines`,
      async (route) => {
        await route.fulfill({
          json: generateMockEngineData({
            pg: { status: 'installed', version: '2.3.1' },
            mongo: { status: 'installed', version: '1.15.0' },
            mysql: { status: 'installed', version: '1.13.0' },
          }),
        });
      }
    );

    await page.route(
      `/v1/namespaces/${namespaces[0]}/database-engines/upgrade-plan`,
      async (route) => {
        await route.fulfill({
          json: {
            upgrades: [],
            pendingActions: [
              {
                name: 'db1',
                message: 'Upgrade CRD',
                pendingTask: 'restart',
              },
            ],
          },
        });
      }
    );

    await goToUrl(page, `/settings/namespaces/${namespaces[0]}`);
    await expect(page.getByTestId('update-db-button')).toBeVisible();
  });

  test('Upgrade engine version via message parsing', async ({ page }) => {
    const mockData = generateMockEngineData({
      pg: { status: 'installed', version: '2.3.1' },
      mongo: {
        status: 'installed',
        version: '1.15.0',
      },
      mysql: { status: 'installed', version: '1.13.0' },
    });

    mockData.items[2].status.availableVersions.engine = {
      '8.0.29-21.1': {
        status: 'available',
      },
    };
    await page.route(
      `/v1/namespaces/${namespaces[0]}/database-engines`,
      async (route) => {
        await route.fulfill({
          json: mockData,
        });
      }
    );
    await page.route(
      `/v1/namespaces/${namespaces[0]}/database-clusters`,
      async (route) => {
        await route.fulfill({
          json: {
            items: [
              {
                metadata: {
                  name: 'db1',
                },
                spec: {
                  engine: {
                    version: '8.0.25-15.1',
                  },
                },
              },
            ],
          },
        });
      }
    );
    await page.route(
      `/v1/namespaces/${namespaces[0]}/database-engines/upgrade-plan`,
      async (route) => {
        await route.fulfill({
          json: {
            upgrades: [
              {
                name: 'percona-server-mongodb-operator',
                currentVersion: '1.15.0',
                targetVersion: '1.16.0',
              },
            ],
            pendingActions: [
              {
                name: 'db1',
                message: 'Upgrade DB version to 8.0.29-21.1 or higher',
                pendingTask: 'upgradeEngine',
              },
              {
                name: 'db2',
                message: 'some message',
                pendingTask: 'ready',
              },
            ],
          },
        });
      }
    );
    await goToUrl(page, `/settings/namespaces/${namespaces[0]}`);
    await expect(
      page.getByRole('button', { name: 'Upgrade Operators' })
    ).toBeVisible();
    await expect(
      page.getByRole('button', { name: 'Upgrade Operators' })
    ).toBeDisabled();
    await expect(
      page.getByText('Upgrade DB version to 8.0.29-21.1 or higher')
    ).toBeVisible();
    await page.getByTestId('update-db-button').click();
    await expect(
      page.getByText(
        'Your DB engine will be upgraded to version 8.0.29-21.1 in db1 cluster.'
      )
    ).toBeVisible();
  });

  test('Upgrade engine version via recommended versions', async ({ page }) => {
    const mockData = generateMockEngineData({
      pg: { status: 'installed', version: '2.3' },
      mongo: {
        status: 'installed',
        version: '1.15.0',
      },
      mysql: { status: 'installed', version: '1.13.0' },
    });

    mockData.items[0].status.availableVersions.engine = {
      '15.1': {
        status: 'available',
      },
      '15.2': {
        status: 'available',
      },
      '15.3': {
        status: 'recommended',
      },
      '15.4': {
        status: 'recommended',
      },
      '16.7': {
        status: 'recommended',
      },
    };
    await page.route(
      `/v1/namespaces/${namespaces[0]}/database-engines`,
      async (route) => {
        await route.fulfill({
          json: mockData,
        });
      }
    );
    await page.route(
      `/v1/namespaces/${namespaces[0]}/database-clusters`,
      async (route) => {
        await route.fulfill({
          json: {
            items: [
              {
                metadata: {
                  name: 'db1',
                },
                spec: {
                  engine: {
                    version: '15.0',
                    type: 'postgresql',
                  },
                },
              },
            ],
          },
        });
      }
    );
    await page.route(
      `/v1/namespaces/${namespaces[0]}/database-engines/upgrade-plan`,
      async (route) => {
        await route.fulfill({
          json: {
            upgrades: [
              {
                name: 'percona-server-postgresql-operator',
                currentVersion: '2.3',
                targetVersion: '2.4',
              },
            ],
            pendingActions: [
              {
                name: 'db1',
                message: 'Upgrade DB version',
                pendingTask: 'upgradeEngine',
              },
              {
                name: 'db2',
                message: 'some message',
                pendingTask: 'ready',
              },
            ],
          },
        });
      }
    );
    await goToUrl(page, `/settings/namespaces/${namespaces[0]}`);
    await expect(
      page.getByRole('button', { name: 'Upgrade Operators' })
    ).toBeVisible();
    await expect(
      page.getByRole('button', { name: 'Upgrade Operators' })
    ).toBeDisabled();
    await expect(page.getByText('Upgrade DB version')).toBeVisible();
    await page.getByTestId('update-db-button').click();
    await expect(
      page.getByText(
        'Your DB engine will be upgraded to version 15.4 in db1 cluster.'
      )
    ).toBeVisible();
  });
});
