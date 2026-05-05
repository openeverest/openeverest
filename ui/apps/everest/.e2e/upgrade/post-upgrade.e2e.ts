import { expect, test } from '@playwright/test';
import fs from 'fs';
import { EVEREST_CI_NAMESPACES, TIMEOUTS } from '@e2e/constants';
import {
  expectedEverestUpgradeAPILine,
  expectedEverestUpgradeCatalogLine,
  expectedEverestUpgradeCRDLine,
  expectedEverestUpgradeFirstLine,
  expectedEverestUpgradeHelmLine,
  expectedEverestUpgradeLastLine,
  expectedEverestUpgradeOperatorLine,
  mongoDBCluster,
  postgresDBCluster,
  pxcDBCluster,
} from './testData';
import { waitForStatus, waitForDelete } from '@e2e/utils/table';
import { getTokenFromLocalStorage } from '@e2e/utils/localStorage';
import { getNamespacesFn } from '@e2e/utils/namespaces';
import { getExpectedOperatorVersions } from '@e2e/upgrade/helper';
import { getDbAvailableUpgradeVersionK8S } from '@e2e/utils/db-cluster';
import {
  deleteDbCluster,
  getDbClustersListAPI,
} from '@e2e/utils/db-clusters-list';
import { queryTestDB } from '@e2e/utils/db-cmd-line';
import { getK8sObjectsNamespaceYaml } from '@e2e/utils/kubernetes';

let namespace: string;
let token: string;
let upgradeClustersInfo: {
  name: string;
  namespace: string;
  dbType: string;
  currentVersion: string;
  upgradeVersion: string | null;
}[] = [];

test.describe.configure({ retries: 0 });
test.describe.configure({ timeout: TIMEOUTS.FifteenMinutes });

test.describe('Post upgrade tests', { tag: '@post-upgrade' }, async () => {
  test.beforeAll(async ({ request }) => {
    token = await getTokenFromLocalStorage();
    [namespace] = await getNamespacesFn(token, request);
  });

  test('Verify upgrade.log file', async () => {
    let expectedText: string;
    const filePath = `/tmp/everest-upgrade.log`;
    const data = fs.readFileSync(filePath, 'utf8');

    expectedText = expectedEverestUpgradeFirstLine();
    expect(data.trim()).toContain(expectedText);

    expectedText = expectedEverestUpgradeCRDLine();
    expect(data.trim()).toContain(expectedText);

    expectedText = expectedEverestUpgradeHelmLine();
    expect(data.trim()).toContain(expectedText);

    expectedText = expectedEverestUpgradeAPILine();
    expect(data.trim()).toContain(expectedText);

    expectedText = expectedEverestUpgradeOperatorLine();
    expect(data.trim()).toContain(expectedText);

    expectedText = expectedEverestUpgradeCatalogLine();
    expect(data.trim()).toContain(expectedText);

    expectedText = expectedEverestUpgradeLastLine();
    expect(data.trim()).toContain(expectedText);
  });

  test('Verify DB clusters are running', async ({ page }) => {
    await page.goto('/databases');

    await test.step('Verify clusters are up', async () => {
      await waitForStatus(
        page,
        mongoDBCluster.name,
        'Up',
        TIMEOUTS.FiveSeconds
      );
      await waitForStatus(
        page,
        postgresDBCluster.name,
        'Up',
        TIMEOUTS.FiveSeconds
      );
      await waitForStatus(page, pxcDBCluster.name, 'Up', TIMEOUTS.FiveSeconds);
    });
  });

  test('Collect info about k8s objects after Everest upgrade', async () => {
    await getK8sObjectsNamespaceYaml(
      EVEREST_CI_NAMESPACES.EVEREST_UI,
      '2-everest-upgrade'
    );
  });

  test('Verify operators upgrade', async ({ page }) => {
    const upgradeOperatorsButton = page.getByRole('button', {
      name: 'Upgrade Operators',
    });
    const upgradeOperatorsModal = page.getByRole('dialog');
    const operatorsVersions = await getExpectedOperatorVersions();

    test.skip(operatorsVersions.length === 0, 'No operators to upgrade');

    await test.step(`Open ${namespace} namespace settings`, async () => {
      await page.goto(`/settings/namespaces/${namespace}`);
      await expect(upgradeOperatorsButton).toBeVisible();
    });

    await test.step(`Verify "upgrade available" text is present in the header`, async () => {
      for (const operator of operatorsVersions) {
        await expect(
          page.getByText(
            `${operator.shortName} ${operator.oldVersion} (Upgrade available)`
          )
        ).toBeVisible();
      }
    });

    await test.step(`Click upgrade button and verify modal contains correct versions and operators`, async () => {
      await upgradeOperatorsButton.click();
      await expect(
        upgradeOperatorsModal.getByText(
          `Are you sure you want to upgrade your operators in ${namespace}?`
        )
      ).toBeVisible();

      for (const operatorVersion of operatorsVersions) {
        await expect(
          upgradeOperatorsModal.locator('li').filter({
            hasText: `${operatorVersion.name} ${operatorVersion.oldVersion} will be upgraded to ${operatorVersion.version}`,
          })
        ).toBeVisible();
      }
    });

    await test.step(`Click Upgrade and wait for upgrade success`, async () => {
      await upgradeOperatorsModal
        .getByRole('button', { name: 'Upgrade' })
        .click();

      for (const operator of operatorsVersions) {
        await expect(async () => {
          await expect(
            page.getByText(`${operator.shortName} ${operator.version}`, {
              exact: true,
            })
          ).toBeVisible();
        }).toPass({ timeout: TIMEOUTS.ThreeMinutes });
      }
    });

    for (const operator of operatorsVersions) {
      await test.step(`Upgrade CR version for [${operator.shortName}]`, async () => {
        await page.goto(`/settings/namespaces/${namespace}`);
        await waitForStatus(
          page,
          `${operator.shortName}-db-cluster`,
          'Up',
          TIMEOUTS.FiveSeconds
        );

        await page
          .getByRole('button', {
            name: `Database needs restart to use CRVersion '${operator.version.replace('v', '')}'`,
          })
          .click();

        const upgradeCRDModal = page.getByRole('dialog');

        await expect(
          upgradeCRDModal.getByText(
            `Are you sure you want to upgrade your CRD (Custom Resource Definition) to version ${operator.version.replace('v', '')} in ${operator.shortName}-db-cluster cluster?`
          )
        ).toBeVisible();

        await upgradeCRDModal.getByRole('button', { name: 'Upgrade' }).click();

        await waitForStatus(
          page,
          `${operator.shortName}-db-cluster`,
          'Initializing',
          TIMEOUTS.ThreeMinutes
        );
        await waitForStatus(
          page,
          `${operator.shortName}-db-cluster`,
          'Up',
          TIMEOUTS.FifteenMinutes
        );
      });
    }
  });

  test('Collect info about k8s objects after operators upgrade', async () => {
    await getK8sObjectsNamespaceYaml(
      EVEREST_CI_NAMESPACES.EVEREST_UI,
      '3-operators-upgrade'
    );
  });

  test('Verify databases upgrade', async ({ page, request }) => {
    await page.goto('/databases');

    const dbClusters = (
      await getDbClustersListAPI(
        EVEREST_CI_NAMESPACES.EVEREST_UI,
        request,
        token
      )
    ).items;

    upgradeClustersInfo = dbClusters.map((c) => {
      return {
        name: c.metadata.name,
        namespace: c.metadata.namespace,
        dbType: c.spec.engine.type,
        currentVersion: c.spec.engine.version,
      };
    });

    for (const c of upgradeClustersInfo) {
      c.upgradeVersion = await getDbAvailableUpgradeVersionK8S(
        `${c.dbType}-db-cluster`,
        EVEREST_CI_NAMESPACES.EVEREST_UI,
        request,
        token
      );
    }
    const upgradeCount = upgradeClustersInfo.filter(
      (item) => item.upgradeVersion !== null
    ).length;

    test.skip(upgradeCount === 0, 'No databases to upgrade');

    for (const c of upgradeClustersInfo.filter(
      (item) => item.upgradeVersion !== null
    )) {
      await test.step(`Upgrade ${c.name} database`, async () => {
        await page.goto(`/databases/${c.namespace}/${c.name}/overview`);
        await page.getByTestId('upgrade-db-btn').click();
        const upgradeDbVersionModal = page.getByRole('dialog');

        await expect(
          upgradeDbVersionModal.getByText(`Upgrade DB version`)
        ).toBeVisible();
        await upgradeDbVersionModal
          .getByTestId('select-db-version-button')
          .click();
        await page.getByRole('option', { name: `${c.upgradeVersion}` }).click();

        await upgradeDbVersionModal.getByTestId('form-dialog-upgrade').click();
      });

      await test.step(`Wait for database [${c.name}] status after upgrade`, async () => {
        await page.goto('/databases');
        await waitForStatus(
          page,
          `${c.name}`,
          'Upgrading',
          TIMEOUTS.ThreeMinutes
        );
        await waitForStatus(page, `${c.name}`, 'Up', TIMEOUTS.FifteenMinutes);
      });
    }

    await test.step(`Check databases were upgraded`, async () => {
      await page.goto('/databases');

      for (const c of upgradeClustersInfo) {
        let tech: string = '';

        switch (c.dbType) {
          case 'pxc': {
            tech = 'MySQL';
            break;
          }
          case 'psmdb': {
            tech = 'MongoDB';
            break;
          }
          case 'postgresql': {
            tech = 'PostgreSQL';
            break;
          }
        }

        const version =
          c.upgradeVersion === null ? c.currentVersion : c.upgradeVersion;
        await expect(
          page.getByRole('row').filter({ hasText: `${c.name}` })
        ).toContainText(`${tech} ${version}`);
      }
    });
  });

  test(`Check data after upgrade`, async ({ page }) => {
    for (const c of upgradeClustersInfo) {
      const result = await queryTestDB(c.name, c.namespace);

      switch (c.dbType) {
        case 'pxc':
          expect(result.trim()).toBe('1\n2\n3');
          break;
        case 'psmdb':
          expect(result.trim()).toBe('[{"a":1},{"a":2},{"a":3}]');
          break;
        case 'postgresql':
          expect(result.trim()).toBe('1\n 2\n 3');
          break;
      }
    }
  });

  test(`Delete clusters`, async ({ page }) => {
    for (const c of upgradeClustersInfo) {
      await deleteDbCluster(page, c.name);
      await waitForStatus(page, c.name, 'Deleting', TIMEOUTS.FifteenSeconds);
      await waitForDelete(page, c.name, TIMEOUTS.FiveMinutes);
    }
  });
});
