import { EVEREST_CI_NAMESPACES, TIMEOUTS } from '@e2e/constants';
import { getCITokenFromLocalStorage } from '@e2e/utils/localStorage';
import { expect, test } from '@playwright/test';
import { selectDbEngine } from '../db-wizard-utils';
import { moveBack, moveForward, submitWizard } from '@e2e/utils/db-wizard';
import { findDbAndClickRow } from '@e2e/utils/db-clusters-list';
import { waitForDelete } from '@e2e/utils/table';
import { goToUrl, limitedSuffixedName } from '@e2e/utils/generic';
import { basicInformationSelectNamespaceCheck } from '@e2e/pr/db-cluster/db-wizard/create-db-cluster/steps/basic-information-step';
import {
  deleteDbClusterFn,
  getDbClusterAPI,
  patchPSMDBFinalizers,
} from '@e2e/utils/db-cluster';
import { advancedConfigurationSelectFirstStorageClass } from '@e2e/pr/db-cluster/db-wizard/create-db-cluster/steps/advanced-configuration-step';

const namespace = EVEREST_CI_NAMESPACES.PSMDB_ONLY;
let token: string;

test.describe
  .parallel('DB cluster wizard creation with sharding (PSMDB)', () => {
  test.beforeAll(async ({}) => {
    token = await getCITokenFromLocalStorage();
    expect(token).not.toHaveLength(0);
  });

  test.beforeEach(async ({ page }) => {
    await goToUrl(page, '/databases');
  });

  test('Sharding toggle is presented for MongoDb during creation of database', async ({
    page,
  }) => {
    // expect(storageClasses.length).toBeGreaterThan(0);
    await test.step('Start DB cluster creation wizard', async () => {
      await selectDbEngine(page, 'psmdb');
    });

    const shardingToggle = page
      .getByTestId('switch-input-sharding')
      .getByRole('checkbox');
    // await shardingToggle.waitFor();

    await expect(shardingToggle).toBeVisible();
    await expect(shardingToggle).not.toBeDisabled();
    await expect(shardingToggle).not.toBeChecked();
  });

  test('Config servers and Nº of shards is presented if sharding is enabled', async ({
    page,
  }) => {
    // expect(storageClasses.length).toBeGreaterThan(0);
    await test.step('Start DB cluster creation wizard', async () => {
      await selectDbEngine(page, 'psmdb');
    });

    await test.step('Basic Info step', async () => {
      const shardingToggle = page
        .getByTestId('switch-input-sharding')
        .getByRole('checkbox');
      await expect(shardingToggle).not.toBeChecked();
      // await shardingToggle.waitFor();

      const basicInfo = page.getByTestId('section-basic-information');
      await expect(basicInfo.getByText('1. Basic Information')).toBeVisible();
      // there are several 'preview-content' elements in 'Basic info' section
      const previewContents = basicInfo.getByTestId('preview-content');
      await expect(
        previewContents.getByText('Sharding: disabled')
      ).toBeVisible();

      await shardingToggle.click();
      await expect(shardingToggle).toBeChecked();
      await expect(
        previewContents.getByText('Sharding: enabled')
      ).toBeVisible();
    });

    await test.step('Resources step', async () => {
      await moveForward(page);

      const shardNr = page.getByTestId('text-input-shard-nr');
      await expect(shardNr).toBeVisible();
      await expect(shardNr).toHaveValue('2');

      const resourcesInfo = page.getByTestId('section-resources');
      await expect(resourcesInfo.getByText('2. Resources')).toBeVisible();

      const previewContents = resourcesInfo.getByTestId('preview-content');
      await expect(previewContents.getByText('2 shards')).toBeVisible();
      await expect(
        previewContents.getByText('3 configuration servers')
      ).toBeVisible();

      const configServersNr = page.getByTestId(
        'toggle-button-group-input-shard-config-servers'
      );
      await expect(configServersNr).toBeVisible();

      const configServers3Btn = page.getByTestId('shard-config-servers-3');
      await expect(configServers3Btn).toBeVisible();
      await expect(configServers3Btn).toHaveAttribute('aria-pressed', 'true');
    });

    await test.step('Back to Basic Info step', async () => {
      await moveBack(page);
      const shardingToggle = page
        .getByTestId('switch-input-sharding')
        .getByRole('checkbox');

      const basicInfo = page.getByTestId('section-basic-information');
      await expect(basicInfo.getByText('1. Basic Information')).toBeVisible();
      const previewContents = basicInfo.getByTestId('preview-content');

      await expect(shardingToggle).toBeChecked();
      await expect(
        previewContents.getByText('Sharding: enabled')
      ).toBeVisible();

      await shardingToggle.click();
      await expect(shardingToggle).not.toBeChecked();
      await expect(
        previewContents.getByText('Sharding: disabled')
      ).toBeVisible();
    });

    await test.step('Resources step', async () => {
      await moveForward(page);

      const shardNr = page.getByTestId('text-input-shard-nr');
      await expect(shardNr).not.toBeVisible();

      const configServersNr = page.getByTestId(
        'toggle-button-group-input-shard-config-servers'
      );
      await expect(configServersNr).not.toBeVisible();

      const resourcesInfo = page.getByTestId('section-resources');
      await expect(resourcesInfo.getByText('2. Resources')).toBeVisible();

      const previewContents = resourcesInfo.getByTestId('preview-content');
      await expect(previewContents.getByText('2 shards')).not.toBeVisible();
      await expect(
        previewContents.getByText('3 configuration servers')
      ).not.toBeVisible();
    });
  });

  test('Sharding should be correctly displayed on the overview page', async ({
    page,
    request,
  }) => {
    test.setTimeout(120 * 1000);
    const dbName = limitedSuffixedName('pr-db-wzd-shr');

    try {
      await test.step('Start DB cluster creation wizard', async () => {
        await selectDbEngine(page, 'psmdb');
      });

      await test.step('Basic Info step', async () => {
        const shardingToggle = page
          .getByTestId('switch-input-sharding')
          .getByRole('checkbox');
        await expect(shardingToggle).not.toBeChecked();

        // namespace
        await basicInformationSelectNamespaceCheck(page, namespace);

        // db cluster name
        await page.getByTestId('text-input-db-name').fill(dbName);

        await shardingToggle.click();
      });

      await test.step('Resources step', async () => {
        await moveForward(page);
        const nodesAccordion = page.getByTestId('nodes-accordion');
        await nodesAccordion.waitFor({ timeout: TIMEOUTS.ThirtySeconds });

        await nodesAccordion.getByTestId('text-input-cpu').fill('1');
        await nodesAccordion.getByTestId('text-input-memory').fill('1');
        await nodesAccordion.getByTestId('text-input-disk').fill('1');
      });

      await test.step('Backup Schedules step', async () => {
        await moveForward(page);
      });

      await test.step('Advanced Configuration step', async () => {
        await moveForward(page);
        await advancedConfigurationSelectFirstStorageClass(page);
      });

      await test.step('Monitoring step', async () => {
        await moveForward(page);
      });

      await test.step('Submit wizard', async () => {
        await submitWizard(page);
      });

      await test.step(`Wait for DB cluster ${dbName} creation`, async () => {
        await expect(async () => {
          // new DB cluster appears in response not immediately.
          // wait for new DB cluster to appear.
          const dbCluster = await getDbClusterAPI(
            dbName,
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

      await test.step('Check DB overview page', async () => {
        await goToUrl(page, '/databases');
        await expect(page.getByText(dbName)).toBeVisible({
          timeout: TIMEOUTS.TenSeconds,
        });

        await findDbAndClickRow(page, dbName);
        await expect(
          page
            .getByTestId('sharding-status-overview-section-row')
            .filter({ hasText: 'Enabled' })
        ).toBeVisible();

        await expect(
          page
            .getByTestId('number-of-shards-overview-section-row')
            .filter({ hasText: '2' })
        ).toBeVisible();

        await expect(
          page
            .getByTestId('config-servers-overview-section-row')
            .filter({ hasText: '3' })
        ).toBeVisible();
      });
    } finally {
      await deleteDbClusterFn(request, dbName, namespace);
      // TODO: This function should be removed after fix for: https://perconadev.atlassian.net/browse/K8SPSMDB-1208
      await patchPSMDBFinalizers(dbName, namespace);
      await waitForDelete(page, dbName, TIMEOUTS.OneMinute);
    }
  });

  test('Mongo with sharding should not pass multinode cluster creation if config servers = 1', async ({
    page,
  }) => {
    await test.step('Start DB cluster creation wizard', async () => {
      await selectDbEngine(page, 'psmdb');
    });

    await test.step('Basic Info step', async () => {
      // namespace
      await basicInformationSelectNamespaceCheck(page, namespace);
      await page.getByTestId('switch-input-sharding').click();
    });

    await test.step('Resources step', async () => {
      await moveForward(page);

      await expect(page.getByTestId(`toggle-button-routers-3`)).toHaveAttribute(
        'aria-pressed',
        'true'
      );

      await expect(page.getByTestId('shard-config-servers-3')).toHaveAttribute(
        'aria-pressed',
        'true'
      );

      await page.getByTestId('shard-config-servers-1').click();
      await expect(
        page.getByTestId('shard-config-servers-error')
      ).toBeVisible();

      await page.getByTestId('toggle-button-nodes-1').click();
      await expect(
        page.getByTestId('shard-config-servers-error')
      ).not.toBeVisible();
      await expect(
        page.getByTestId('db-wizard-continue-button')
      ).not.toBeDisabled();

      await page.getByTestId('toggle-button-nodes-3').click();
      await expect(
        page.getByTestId('shard-config-servers-error')
      ).toBeVisible();

      await page.getByTestId('shard-config-servers-3').click();
      await expect(
        page.getByTestId('shard-config-servers-error')
      ).not.toBeVisible();
      await expect(
        page.getByTestId('db-wizard-continue-button')
      ).not.toBeDisabled();
    });
  });

  test('0 value of the Nº of shards causes an error', async ({ page }) => {
    await test.step('Start DB cluster creation wizard', async () => {
      await selectDbEngine(page, 'psmdb');
    });

    await test.step('Basic Info step', async () => {
      const shardingToggle = page
        .getByTestId('switch-input-sharding')
        .getByRole('checkbox');
      await expect(shardingToggle).not.toBeChecked();
      await shardingToggle.click();
      await expect(shardingToggle).toBeChecked();
    });

    await test.step('Resources step', async () => {
      await moveForward(page);

      await page.getByTestId('text-input-shard-nr').fill('0');
      await expect(
        page.getByText('The value cannot be less than 1')
      ).toBeVisible();
      await page.getByTestId('text-input-shard-nr').fill('1');
      await expect(
        page.getByTestId('db-wizard-continue-button')
      ).not.toBeDisabled();
    });
  });

  test('Changing the Nº of nodes causes changing the confing servers Nº automatically, until the user touches confing servers Nº', async ({
    page,
  }) => {
    await test.step('Start DB cluster creation wizard', async () => {
      await selectDbEngine(page, 'psmdb');
    });

    await test.step('Basic Info step', async () => {
      const shardingToggle = page
        .getByTestId('switch-input-sharding')
        .getByRole('checkbox');
      await expect(shardingToggle).not.toBeChecked();
      await shardingToggle.click();
      await expect(shardingToggle).toBeChecked();
    });

    await test.step('Resources step', async () => {
      await moveForward(page);
      await expect(page.getByTestId(`toggle-button-routers-3`)).toHaveAttribute(
        'aria-pressed',
        'true'
      );
      await expect(page.getByTestId('shard-config-servers-3')).toHaveAttribute(
        'aria-pressed',
        'true'
      );

      await page.getByTestId('toggle-button-nodes-1').click();
      await expect(page.getByTestId('shard-config-servers-1')).toHaveAttribute(
        'aria-pressed',
        'true'
      );
      await page.getByTestId('toggle-button-nodes-5').click();
      await expect(page.getByTestId('shard-config-servers-5')).toHaveAttribute(
        'aria-pressed',
        'true'
      );
      await page.getByTestId('toggle-button-nodes-custom').click();
      await page.getByTestId('text-input-custom-nr-of-nodes').fill('7');
      await expect(page.getByTestId('shard-config-servers-7')).toHaveAttribute(
        'aria-pressed',
        'true'
      );
      await page.getByTestId('text-input-custom-nr-of-nodes').fill('9');
      await expect(page.getByTestId('shard-config-servers-7')).toHaveAttribute(
        'aria-pressed',
        'true'
      );

      await page.getByTestId('shard-config-servers-3').click();
      await page.getByTestId('toggle-button-nodes-1').click();
      await expect(page.getByTestId('shard-config-servers-1')).toHaveAttribute(
        'aria-pressed',
        'false'
      );
      await expect(page.getByTestId('shard-config-servers-3')).toHaveAttribute(
        'aria-pressed',
        'true'
      );
    });
  });

  test('Sharding is not reset to default when returning to the previous step of the form in dbWizard', async ({
    page,
  }) => {
    await test.step('Start DB cluster creation wizard', async () => {
      await selectDbEngine(page, 'psmdb');
    });

    await test.step('Basic Info step', async () => {
      const shardingToggle = page
        .getByTestId('switch-input-sharding')
        .getByRole('checkbox');
      await expect(shardingToggle).not.toBeChecked();
      await shardingToggle.click();
      await expect(shardingToggle).toBeChecked();
    });

    await test.step('Resources step', async () => {
      await moveForward(page);
    });

    await test.step('Back to Basic Info step', async () => {
      await moveBack(page);

      const shardingToggle = page
        .getByTestId('switch-input-sharding')
        .getByRole('checkbox');
      await expect(shardingToggle).toBeChecked();
    });

    await test.step('Resources step', async () => {
      await moveForward(page);
    });

    await test.step('Backup Schedules step', async () => {
      await moveForward(page);
    });

    await test.step('Back to Basic Info step', async () => {
      await page.getByTestId('edit-section-1').click();

      const shardingToggle = page
        .getByTestId('switch-input-sharding')
        .getByRole('checkbox');
      await expect(shardingToggle).toBeChecked();
    });
  });
});
