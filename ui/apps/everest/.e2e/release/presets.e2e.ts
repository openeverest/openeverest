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

import { APIRequestContext, Locator, Page } from '@playwright/test';
import { expect, test } from '@e2e/fixtures/auth';
import {
  EVEREST_CI_CLUSTER,
  EVEREST_CI_NAMESPACES,
  getBucketNamespacesMap,
  TIMEOUTS,
} from '@e2e/constants';
import { getCITokenFromLocalStorage } from '@e2e/utils/localStorage';
import { openDbCreationForm } from '@e2e/utils/db-wizard';
import { waitForInstanceStatus } from '@e2e/utils/table';
import {
  applyClusterInstancePreset,
  createInstancePreset,
  deleteClusterInstancePreset,
  deleteInstancePreset,
  PresetSpec,
} from '@e2e/utils/presets';

// Only the PXC provider is deployed in the release lane.
const PROVIDER = 'percona-xtradb-cluster';
const BACKUP_CLASS = 'pxc';

const NAMESPACE = EVEREST_CI_NAMESPACES.EVEREST_UI;
// Any other namespace the wizard offers: it only has to hold a preset.
const OTHER_NAMESPACE = 'default';

// One name in all three places — cluster-wide, NAMESPACE and OTHER_NAMESPACE —
// because telling same-named presets apart is what the feature is about.
const PRESET = 'e2e-preset';
const CLUSTER_PRESET_ANNOTATION = 'openeverest.io/cluster-instance-preset';
const NAMESPACE_PRESET_ANNOTATION = 'openeverest.io/instance-preset';

const INSTANCE_FROM_NAMESPACE_PRESET = 'e2e-from-ns-preset';
const INSTANCE_FROM_CLUSTER_PRESET = 'e2e-from-cl-preset';

const clusterUrl = `/v1/clusters/${EVEREST_CI_CLUSTER}`;
const instanceUrl = (name: string) =>
  `${clusterUrl}/namespaces/${NAMESPACE}/instances/${name}`;
const authHeaders = (token: string) => ({ Authorization: `Bearer ${token}` });

// The smallest instance the provider takes: the two create tests bring it up
// next to the PITR and restore goldens, which share the node.
const presetSpec = (version: string, extra: PresetSpec = {}): PresetSpec => ({
  providerRef: { name: PROVIDER },
  version,
  topology: { type: 'cluster' },
  components: {
    engine: {
      replicas: 1,
      resources: { limits: { cpu: '1', memory: '1Gi' } },
      storage: { size: '1Gi' },
    },
    proxy: { type: 'haproxy', replicas: 1 },
  },
  ...extra,
});

const getBackupStorage = () =>
  getBucketNamespacesMap().find(
    ([, namespace]) => namespace === NAMESPACE
  )?.[0];

const scratchCard = (page: Page) =>
  page.getByRole('button', { name: /^Start from scratch/ });

type Scope = 'Cluster-wide' | 'Namespace';
const presetCard = (list: Locator, scope: Scope) =>
  list.getByRole('button', { name: new RegExp(`^${PRESET}\\s*${scope}`) });

// Cards that do not fit the row move behind "Browse all", so the cards are
// looked up in its dialog when it is offered (a stand may hold other presets).
const openPresetList = async (page: Page): Promise<Locator> => {
  await expect(scratchCard(page)).toBeVisible();
  const browse = page.getByRole('button', { name: /^Browse all \d+ presets$/ });
  if (await browse.isVisible()) {
    await browse.click();
    return page.getByRole('dialog');
  }
  return page.locator('form');
};

const pickPreset = async (page: Page, scope: Scope) => {
  const list = await openPresetList(page);
  await presetCard(list, scope).click();
  // A picked card is kept in the row even when it was taken from the dialog.
  await expect(presetCard(page.locator('form'), scope)).toHaveAttribute(
    'aria-pressed',
    'true'
  );
};

const selectNamespace = async (page: Page, namespace: string) => {
  await page.getByTestId('k8s-namespace-autocomplete').click();
  await page.getByRole('option', { name: namespace, exact: true }).click();
  await expect(page.getByTestId('text-input-k8s-namespace')).toHaveValue(
    namespace
  );
};

const openWizard = async (page: Page, namespace = NAMESPACE) => {
  await page.goto('/databases');
  await openDbCreationForm(page, PROVIDER);
  await expect(page.getByTestId('step-header')).toBeVisible();
  await selectNamespace(page, namespace);
};

const droppedPresetNotice = (page: Page) =>
  page.getByRole('status').filter({ hasText: `Preset “${PRESET}”` });

const createFromPickedPreset = async (page: Page, name: string) => {
  await page.getByTestId('text-input-db-name').fill(name);
  await page.getByTestId('db-wizard-submit-button').click();
  // The wizard goes to the list once the create request succeeds.
  await page.waitForURL('**/databases', { timeout: TIMEOUTS.ThirtySeconds });
  // A preset is only worth offering if what it describes comes up.
  await waitForInstanceStatus(page, name, 'Ready', TIMEOUTS.FifteenMinutes);
};

const deleteInstance = async (
  request: APIRequestContext,
  token: string,
  name: string
) => {
  const response = await request.delete(instanceUrl(name), {
    headers: authHeaders(token),
  });
  expect([204, 404], await response.text()).toContain(response.status());
  // A leftover instance would take resources from the goldens of this lane.
  await expect
    .poll(
      async () =>
        (
          await request.get(instanceUrl(name), { headers: authHeaders(token) })
        ).status(),
      { timeout: TIMEOUTS.TenMinutes, intervals: [5000] }
    )
    .toBe(404);
};

test.describe('Instance presets', { tag: '@release' }, () => {
  // One worker, in order: the tests share the presets created below. Unlike
  // serial mode, a failed test does not skip the ones after it.
  test.describe.configure({ mode: 'default' });
  // Without it a click on a card that is not there waits out the whole test
  // timeout, which the two create tests set to half an hour.
  test.use({ actionTimeout: TIMEOUTS.ThirtySeconds });

  let token: string;
  let clusterPresetSpec: PresetSpec;
  let namespacePresetSpec: PresetSpec;
  let otherNamespaceVersion: string;
  let namespaceVersion: string;
  const backupStorage = getBackupStorage();

  test.beforeAll(async ({ request }) => {
    token = await getCITokenFromLocalStorage();
    expect(backupStorage, 'a backup storage in the CI namespace').toBeTruthy();

    const providers = await request.get(`${clusterUrl}/providers`, {
      headers: authHeaders(token),
    });
    const provider = (await providers.json()).items.find(
      (item: { metadata: { name: string } }) => item.metadata.name === PROVIDER
    );
    // Three versions tell the three same-named presets apart on their cards.
    const [clusterVersion, ...namespaceVersions]: string[] =
      provider.spec.versions.map((version: { name: string }) => version.name);
    [namespaceVersion, otherNamespaceVersion] = namespaceVersions;
    expect(otherNamespaceVersion, 'three provider versions').toBeTruthy();

    for (const name of [
      INSTANCE_FROM_NAMESPACE_PRESET,
      INSTANCE_FROM_CLUSTER_PRESET,
    ]) {
      await deleteInstance(request, token, name);
    }

    clusterPresetSpec = presetSpec(clusterVersion);
    // A namespace preset can pin what lives in its namespace: here, a storage.
    namespacePresetSpec = presetSpec(namespaceVersion, {
      backup: {
        enabled: true,
        classRef: { name: BACKUP_CLASS },
        storages: [
          {
            storageRef: { name: backupStorage },
            // Once a year: the schedule must exist, not run during the test.
            schedules: [{ name: 'yearly', enabled: true, cron: '0 0 1 1 *' }],
          },
        ],
      },
    });
    applyClusterInstancePreset(PRESET, clusterPresetSpec);
    await createInstancePreset(
      request,
      token,
      NAMESPACE,
      PRESET,
      namespacePresetSpec
    );
    await createInstancePreset(
      request,
      token,
      OTHER_NAMESPACE,
      PRESET,
      presetSpec(otherNamespaceVersion)
    );
  });

  test.afterAll(async ({ request }) => {
    for (const name of [
      INSTANCE_FROM_NAMESPACE_PRESET,
      INSTANCE_FROM_CLUSTER_PRESET,
    ]) {
      await deleteInstance(request, token, name);
    }
    deleteClusterInstancePreset(PRESET);
    await deleteInstancePreset(request, token, NAMESPACE, PRESET);
    await deleteInstancePreset(request, token, OTHER_NAMESPACE, PRESET);
  });

  test('offers the cluster-wide presets and the presets of the chosen namespace', async ({
    page,
  }) => {
    await openWizard(page);

    // The namespace decides which presets are offered, so it comes first.
    const namespaceField = await page
      .getByTestId('k8s-namespace-autocomplete')
      .boundingBox();
    const firstCard = await scratchCard(page).boundingBox();
    expect(namespaceField!.y).toBeLessThan(firstCard!.y);

    const list = await openPresetList(page);
    await expect(presetCard(list, 'Cluster-wide')).toBeVisible();
    // Exactly one: the same-named preset of the other namespace is not offered.
    await expect(presetCard(list, 'Namespace')).toHaveCount(1);
    await expect(presetCard(list, 'Namespace')).toContainText(namespaceVersion);
  });

  test('shows in the summary what a namespace preset pins', async ({
    page,
  }) => {
    await openWizard(page);

    await pickPreset(page, 'Namespace');

    await expect(page.getByTestId('section-backups')).toContainText(
      backupStorage!
    );
    await expect(page.getByTestId('section-database-version')).toContainText(
      namespaceVersion
    );
  });

  test('drops a namespace preset when the namespace changes', async ({
    page,
  }) => {
    await openWizard(page);
    await pickPreset(page, 'Namespace');

    await selectNamespace(page, OTHER_NAMESPACE);

    await expect(droppedPresetNotice(page)).toContainText(
      `belongs to namespace “${NAMESPACE}” and can't be used in “${OTHER_NAMESPACE}”`
    );
    await expect(scratchCard(page)).toHaveAttribute('aria-pressed', 'true');
    // The new namespace has a preset of that name: it must not be taken instead.
    const list = await openPresetList(page);
    await expect(presetCard(list, 'Namespace')).toContainText(
      otherNamespaceVersion
    );
    await expect(presetCard(list, 'Namespace')).toHaveAttribute(
      'aria-pressed',
      'false'
    );
    await page.keyboard.press('Escape');
    await expect(page.getByTestId('section-backups')).not.toContainText(
      backupStorage!
    );

    await droppedPresetNotice(page)
      .getByRole('button', { name: 'Close' })
      .click();
    await expect(droppedPresetNotice(page)).toBeHidden();
  });

  test('keeps a cluster-wide preset when the namespace changes', async ({
    page,
  }) => {
    await openWizard(page);
    await pickPreset(page, 'Cluster-wide');

    await selectNamespace(page, OTHER_NAMESPACE);

    const list = await openPresetList(page);
    await expect(presetCard(list, 'Cluster-wide')).toHaveAttribute(
      'aria-pressed',
      'true'
    );
    await expect(droppedPresetNotice(page)).toHaveCount(0);
  });

  test.describe('from a preset to a running instance', () => {
    // Waiting for Ready a second time would not turn a stuck instance green.
    test.describe.configure({ retries: 0 });

    test('brings up an instance created from a namespace preset', async ({
      page,
      request,
    }) => {
      test.setTimeout(TIMEOUTS.ThirtyMinutes);
      await openWizard(page);
      await pickPreset(page, 'Namespace');

      await createFromPickedPreset(page, INSTANCE_FROM_NAMESPACE_PRESET);

      const instance = await (
        await request.get(instanceUrl(INSTANCE_FROM_NAMESPACE_PRESET), {
          headers: authHeaders(token),
        })
      ).json();
      expect(instance.metadata.annotations).toMatchObject({
        [NAMESPACE_PRESET_ANNOTATION]: PRESET,
      });
      expect(instance.metadata.annotations).not.toHaveProperty(
        CLUSTER_PRESET_ANNOTATION
      );
      expect(instance.spec).toMatchObject(namespacePresetSpec);

      await deleteInstance(request, token, INSTANCE_FROM_NAMESPACE_PRESET);
    });

    test('brings up an instance created from a cluster-wide preset', async ({
      page,
      request,
    }) => {
      test.setTimeout(TIMEOUTS.ThirtyMinutes);
      await openWizard(page);
      await pickPreset(page, 'Cluster-wide');

      await createFromPickedPreset(page, INSTANCE_FROM_CLUSTER_PRESET);

      const instance = await (
        await request.get(instanceUrl(INSTANCE_FROM_CLUSTER_PRESET), {
          headers: authHeaders(token),
        })
      ).json();
      expect(instance.metadata.annotations).toMatchObject({
        [CLUSTER_PRESET_ANNOTATION]: PRESET,
      });
      expect(instance.metadata.annotations).not.toHaveProperty(
        NAMESPACE_PRESET_ANNOTATION
      );
      expect(instance.spec).toMatchObject(clusterPresetSpec);

      await deleteInstance(request, token, INSTANCE_FROM_CLUSTER_PRESET);
    });
  });

  test('reports a list that failed to load and loads it on retry', async ({
    page,
  }) => {
    // The real API does not fail on demand, so the namespace list is answered
    // with an error until the retry.
    let listFails = true;
    await page.route(`**/namespaces/${NAMESPACE}/instance-presets*`, (route) =>
      listFails
        ? route.fulfill({
            status: 500,
            contentType: 'application/json',
            body: JSON.stringify({
              message: 'failed to list instance presets',
            }),
          })
        : route.continue()
    );
    await openWizard(page);

    const loadError = page
      .getByRole('alert')
      .filter({ hasText: "Some presets didn't load." });
    await expect(loadError).toBeVisible();
    // The list that did load is still offered.
    const list = await openPresetList(page);
    await expect(presetCard(list, 'Cluster-wide')).toBeVisible();
    await expect(presetCard(list, 'Namespace')).toHaveCount(0);
    await page.keyboard.press('Escape');

    listFails = false;
    await loadError.getByRole('button', { name: 'Retry' }).click();

    await expect(loadError).toBeHidden();
    await expect(
      presetCard(await openPresetList(page), 'Namespace')
    ).toBeVisible();
  });
});
