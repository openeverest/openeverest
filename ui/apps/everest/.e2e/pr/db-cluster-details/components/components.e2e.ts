import { findDbAndClickRow } from '@e2e/utils/db-clusters-list';
import { expect, test } from '@playwright/test';
import { DBClusterDetailsTabs } from '../../../../src/pages/db-cluster-details/db-cluster-details.types';
import { goToUrl } from '@e2e/utils/generic';
import { EVEREST_CI_NAMESPACES, TIMEOUTS } from '@e2e/constants';

const namespace = EVEREST_CI_NAMESPACES.PG_ONLY,
  dbClusterName = 'pr-db-det-comp';

test.describe.serial('Cluster components', async () => {
  test.describe.configure({ timeout: TIMEOUTS.ThirtySeconds });

  test.beforeEach(async ({ page }) => {
    await goToUrl(page, '/databases');
  });

  test('Same components on table and diagram', async ({ page }) => {
    const componentsList: Array<{
      name: string;
      type: string;
      containers: Array<{
        name: string;
      }>;
    }> = [];

    await test.step('Open DB cluster overview page', async () => {
      await expect(page.getByText(dbClusterName)).toBeVisible({
        timeout: TIMEOUTS.TenSeconds,
      });
      await findDbAndClickRow(page, dbClusterName);
    });

    await test.step('Open Components tab', async () => {
      const getResp = page.waitForResponse(
        (resp) =>
          resp.request().method() === 'GET' &&
          resp
            .url()
            .includes(
              `/v1/namespaces/${namespace}/database-clusters/${dbClusterName}/components`
            ) &&
          resp.status() === 200
      );

      await page.getByTestId(DBClusterDetailsTabs.components).click();
      await getResp;

      const switchInput = page
        .getByTestId('switch-input-table-view')
        .getByRole('checkbox');
      await switchInput.check();
      await expect(page.getByRole('table')).toBeVisible();
      expect(
        await page
          .locator(
            '.MuiTableRow-root:not(.MuiTableRow-head):not(.Mui-TableBodyCell-DetailPanel)'
          )
          .count()
      ).not.toBe(0);
    });

    await test.step('Check Table View', async () => {
      await page.getByLabel('Expand all').getByRole('button').click();
      await expect(page.getByLabel('Collapse all')).toBeVisible();
      const allComponents = await page
        .locator(
          `table[data-testid="${dbClusterName}-components"] > tbody > tr.MuiTableRow-root:not(.MuiTableRow-head):not(.Mui-TableBodyCell-DetailPanel)`
        )
        .all();
      const allContainers = await page
        .locator('tr.Mui-TableBodyCell-DetailPanel')
        .all();

      for (const component of allComponents) {
        // We just read static data, as the other data might change while we check things around
        const name = await component.locator('td').nth(2).innerText();
        const type = await component.locator('td').nth(3).innerText();
        componentsList.push({ name, type, containers: [] });
      }

      for (const [index, container] of allContainers.entries()) {
        const innerText = await container.innerText();

        if (innerText !== 'No containers') {
          const innerTable = container.locator('table');
          const innerTableRows = await innerTable.locator('tr').all();

          for (const innerTableRow of innerTableRows) {
            const name = await innerTableRow.locator('td').nth(2).innerText();
            componentsList[index].containers.push({ name });
          }
        }
      }
    });

    await test.step('Check Diagram view', async () => {
      const switchInput = page
        .getByTestId('switch-input-table-view')
        .getByRole('checkbox');
      await switchInput.uncheck();
      await expect(page.getByRole('table')).not.toBeVisible();

      await expect(page.getByTitle('zoom in')).toBeVisible();

      for (const component of componentsList) {
        const { name, type, containers } = component;
        const correspondingNode = page.getByTestId(
          new RegExp(`component-node-${name}(-selected)?`)
        );
        const correspondingNodeSelected = page.getByTestId(
          `component-node-${name}-selected`
        );
        const isSelected = await correspondingNodeSelected.isVisible();

        await expect(correspondingNode).toBeVisible();
        expect(
          await correspondingNode.getByTestId('component-name').innerText()
        ).toBe(name);
        expect(
          await correspondingNode.getByTestId('component-type').innerText()
        ).toBe(type);

        if (!isSelected) {
          await correspondingNode.click();
        }
        // Wait for the diagram to be updated
        await page.waitForTimeout(300);

        if (containers.length) {
          const correspondingContainers = await page
            .getByTestId(/container-node-.+/)
            .all();
          expect(correspondingContainers.length).toBe(containers.length);

          for (const container of containers) {
            const { name } = container;
            const correspondingContainer = page.getByTestId(
              `container-node-${name}`
            );

            await expect(correspondingContainer).toBeVisible();
            expect(
              await correspondingContainer
                .getByTestId('container-name')
                .innerText()
            ).toBe(name);
          }
        }
      }
    });
  });
});
