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

import { APIRequestContext, Page } from '@playwright/test';
import { EVEREST_CI_NAMESPACES, TIMEOUTS } from '@e2e/constants';
import { expect, test } from '@e2e/fixtures/auth';
import {
  moveForward,
  openDbCreationForm,
  passMonitoringStep,
  populateBasicInformation,
  populateEngineResources,
  submitWizard,
} from '@e2e/utils/db-wizard';
import { getCITokenFromLocalStorage } from '@e2e/utils/localStorage';
import {
  createPmmServiceAccountToken,
  deleteInOrder,
  destinationPath,
  getDestination,
  getPmmMySqlServices,
  getSecret,
  getStatusCode,
  instancePath,
  PMM_SERVER_URL,
  secretPath,
  updateInstance,
} from '@e2e/utils/monitoring';
import { getClusterDetailedInfo } from '@e2e/utils/storage-class';
import {
  confirmDeletion,
  findRowAndClickActions,
  waitForStatus,
} from '@e2e/utils/table';

test.describe.configure({ retries: 0 });

const PXC_PROVIDER_NAME = 'percona-xtradb-cluster';
const PMM_CLASS_TITLE = 'Percona Monitoring and Management';
const DESTINATIONS_URL = '/settings/monitoring-destinations';

// Golden path of monitoring with the PMM class against a real PMM 3 server:
// a destination is added in Settings, an instance sends to it from the
// wizard, its overview reports the monitoring, and the destination can only
// be deleted once no instance uses it, keeping its credentials for reuse.
test.describe.serial(
  'Monitoring golden (PXC + PMM)',
  {
    tag: '@release',
  },
  () => {
    test.describe.configure({ timeout: TIMEOUTS.ThirtyMinutes });

    const namespace = EVEREST_CI_NAMESPACES.EVEREST_UI;
    const instanceName = 'pxc-1-mon';
    const overviewUrl = `/databases/${namespace}/${instanceName}/overview`;
    const destination = 'e2e-mon-pmm';
    const credentialsSecret = `monitoring-destination-${destination}-credentials`;
    // Added on the credentials Secret that deleting `destination` keeps.
    const reusingDestination = 'e2e-mon-reuse';
    // Listed by the instance without existing, as once deleted behind its back.
    const goneDestination = 'e2e-mon-gone';
    let storageClass: string;

    const cleanUp = (request: APIRequestContext) =>
      deleteInOrder(request, [
        instancePath(namespace, instanceName),
        destinationPath(namespace, destination),
        destinationPath(namespace, reusingDestination),
        secretPath(namespace, credentialsSecret),
      ]);

    test.beforeAll(async ({ request }) => {
      const { storageClassNames = [] } = await getClusterDetailedInfo(
        await getCITokenFromLocalStorage(),
        request
      );
      storageClass = storageClassNames[0];
      await cleanUp(request);
    });

    // Unlike a trailing test, which serial mode skips after a failure.
    test.afterAll(async ({ request }) => {
      await cleanUp(request);
    });

    // Fills the Add destination dialog up to the credentials.
    const startAddingPmmDestination = async (page: Page, name: string) => {
      await page.goto(DESTINATIONS_URL);
      await page.getByTestId('add-monitoring-destination').click();
      const dialog = page.getByTestId('monitoring-destination-form-dialog');
      await dialog.getByTestId('text-input-name').fill(name);
      await dialog.getByTestId('namespace-autocomplete').click();
      await page.getByRole('option', { name: namespace, exact: true }).click();
      await dialog.getByTestId('select-class-name-button').click();
      await page.getByRole('option', { name: PMM_CLASS_TITLE }).click();
      await dialog
        .getByTestId('text-input-spec.parameters.url')
        .fill(PMM_SERVER_URL);
      // The PMM chart serves a self-signed certificate.
      await dialog
        .getByRole('switch', { name: 'Skip TLS certificate verification' })
        .check();
      return dialog;
    };

    // The status reads e.g. "Ready · v3.10.0"; the list polls it.
    const expectDestinationReady = (page: Page, name: string) =>
      expect(
        page
          .getByRole('row')
          .filter({ hasText: name })
          .getByText(/^Ready\b/)
      ).toBeVisible({ timeout: TIMEOUTS.ThreeMinutes });

    test('Add a PMM destination', async ({ page }) => {
      const dialog = await startAddingPmmDestination(page, destination);
      await dialog
        .getByTestId('text-input-credentials.token')
        .fill(createPmmServiceAccountToken());
      await dialog.getByTestId('form-dialog-add').click();
      await expect(dialog).toBeHidden();

      await expectDestinationReady(page, destination);
    });

    test('Create an instance that sends to the destination', async ({
      page,
    }) => {
      await page.goto('/databases');
      await openDbCreationForm(page, PXC_PROVIDER_NAME);
      await populateBasicInformation(
        page,
        namespace,
        instanceName,
        'pxc',
        storageClass,
        false,
        ''
      );
      await moveForward(page);
      await passMonitoringStep(page, [destination]);

      // Database Version, then Resources: one node sized for a CI runner.
      await moveForward(page);
      await page
        .getByTestId('text-input-spec.components.engine.replicas')
        .fill('1');
      await populateEngineResources(page, 1, 1, 1);
      await moveForward(page);
      await page
        .getByTestId('text-input-spec.components.proxy.replicas')
        .fill('1');
      await submitWizard(page);

      // The wizard navigates to the list once the create request succeeds.
      await page.waitForURL('**/databases', {
        timeout: TIMEOUTS.ThirtySeconds,
      });
      await waitForStatus(page, instanceName, 'Ready', TIMEOUTS.FifteenMinutes);
    });

    test('The instance reports its monitoring, and PMM receives it', async ({
      page,
    }) => {
      await page.goto(overviewUrl);
      const entry = page.getByTestId(
        `monitoring-${destination}-overview-section`
      );
      await expect(entry.getByTestId('status-overview-section-row')).toHaveText(
        /Ready$/,
        { timeout: TIMEOUTS.FiveMinutes }
      );
      await expect(entry.getByTestId('mode-overview-section-row')).toHaveText(
        /Provider-managed$/
      );

      await expect
        .poll(
          () =>
            getPmmMySqlServices().some(
              ({ cluster }) => cluster === instanceName
            ),
          { timeout: TIMEOUTS.FiveMinutes, intervals: [5000, 10000] }
        )
        .toBe(true);
    });

    test('Settings show the destination in use', async ({ page }) => {
      await page.goto(DESTINATIONS_URL);
      const usedBy = page.getByTestId(`used-by-${destination}-toggle`);
      await expect(usedBy).toHaveText('1 instance');
      await usedBy.click();
      await expect(
        page
          .getByRole('dialog')
          .filter({ hasText: `Instances using ${destination}` })
      ).toContainText(instanceName);
      await page.getByTestId(`used-by-${destination}-dialog-close`).click();

      await findRowAndClickActions(page, destination, 'Delete');
      const deleteDialog = page.getByTestId(`${destination}-confirm-dialog`);
      await expect(deleteDialog).toContainText(
        `${destination} is used by ${instanceName}. Remove it from those instances first.`
      );
      await deleteDialog.getByTestId('confirm-dialog-close').click();

      await findRowAndClickActions(page, destination, 'Edit');
      const editDialog = page.getByTestId('monitoring-destination-form-dialog');
      await editDialog
        .getByTestId('text-input-spec.parameters.url')
        .fill(`${PMM_SERVER_URL}:443`);
      await expect(
        editDialog.getByTestId('monitoring-destination-restart')
      ).toContainText('the instance that sends to this destination');
      await editDialog.getByTestId('form-dialog-cancel').click();
    });

    test('An entry whose destination is gone can be removed', async ({
      page,
      request,
    }) => {
      await updateInstance(request, namespace, instanceName, (instance) => {
        instance.spec?.monitoring?.destinations?.push({
          name: goneDestination,
          destinationRef: { name: goneDestination },
        });
      });

      await page.goto(overviewUrl);
      const goneEntry = page.getByTestId(
        `monitoring-${goneDestination}-overview-section`
      );
      await expect(
        goneEntry.getByTestId('details-overview-section-row')
      ).toContainText(
        `${goneDestination} no longer exists. Remove it from monitoring.`,
        { timeout: TIMEOUTS.OneMinute }
      );

      await page.getByTestId('monitoring-edit-button').click();
      const modal = page.getByTestId('monitoring-edit-form-dialog');
      const goneCard = modal
        .getByTestId('monitoring-unavailable-entries')
        .getByRole('button', { name: goneDestination });
      await expect(goneCard).toContainText('The destination no longer exists');
      await goneCard.click();
      await expect(goneCard).toContainText('Removed when you save');
      await modal.getByTestId('form-dialog-save').click();
      await expect(page.getByText('Monitoring updated')).toBeVisible();

      await expect(goneEntry).toBeHidden();
      await expect(
        page.getByTestId(`monitoring-${destination}-overview-section`)
      ).toBeVisible();
    });

    test('Remove the destination from the instance', async ({ page }) => {
      await page.goto(overviewUrl);
      await page.getByTestId('monitoring-edit-button').click();
      const modal = page.getByTestId('monitoring-edit-form-dialog');
      const card = modal.getByRole('button', { name: destination });
      await card.click();
      await expect(card).toHaveAttribute('aria-pressed', 'false');
      const disruption = modal.getByTestId('monitoring-disruption');
      await expect(disruption).toContainText(
        `Removing ${PMM_CLASS_TITLE} restarts the instance pods`
      );
      await expect(disruption).toContainText(
        'single-replica components restart'
      );
      await modal.getByTestId('form-dialog-save').click();
      await expect(page.getByText('Monitoring updated')).toBeVisible();

      await expect(page.getByTestId('monitoring-overview-section')).toHaveText(
        /Not configured$/
      );
    });

    test('Delete the unused destination, keeping its Secret', async ({
      page,
      request,
    }) => {
      // Core recounts the destination's instances right after the change.
      await expect
        .poll(
          async () =>
            (await getDestination(request, namespace, destination)).status
              ?.attachedInstances ?? 0,
          { timeout: TIMEOUTS.OneMinute }
        )
        .toBe(0);

      await page.goto(DESTINATIONS_URL);
      await findRowAndClickActions(page, destination, 'Delete');
      await expect(
        page.getByTestId(`${destination}-confirm-dialog`)
      ).toContainText(
        `Its credentials Secret ${credentialsSecret} is kept and can be reused.`
      );
      await confirmDeletion(page, 'confirm-dialog-delete');

      await expect(
        page.getByRole('row').filter({ hasText: destination })
      ).toBeHidden({ timeout: TIMEOUTS.OneMinute });
      expect(
        await getStatusCode(request, secretPath(namespace, credentialsSecret))
      ).toBe(200);
    });

    test('Add a destination on the kept Secret', async ({ page }) => {
      const dialog = await startAddingPmmDestination(page, reusingDestination);
      await dialog.getByTestId('radio-option-existing').check();
      await dialog.getByTestId('select-existing-secret-name-button').click();
      await page
        .getByRole('option', { name: credentialsSecret, exact: true })
        .click();
      await dialog.getByTestId('form-dialog-add').click();
      await expect(dialog).toBeHidden();

      await expectDestinationReady(page, reusingDestination);
    });

    test('Replace the credentials of a destination', async ({
      page,
      request,
    }) => {
      const before = await getSecret(request, namespace, credentialsSecret);

      await page.goto(DESTINATIONS_URL);
      await findRowAndClickActions(page, reusingDestination, 'Edit');
      const dialog = page.getByTestId('monitoring-destination-form-dialog');
      await dialog.getByTestId('radio-option-update').check();
      await dialog
        .getByTestId('text-input-credentials.token')
        .fill(createPmmServiceAccountToken());
      await dialog.getByTestId('form-dialog-save').click();
      await expect(dialog).toBeHidden();

      // The API never returns Secret data; a new version shows the rewrite.
      const after = await getSecret(request, namespace, credentialsSecret);
      expect(after.metadata?.resourceVersion).not.toBe(
        before.metadata?.resourceVersion
      );
    });
  }
);
