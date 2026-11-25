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
import { prepareTestDB } from '@e2e/utils/db-cmd-line';
import { clickOnDemandBackup } from '@e2e/pr/db-cluster-details/utils';
import { waitForInstanceStatus, waitForStatus } from '@e2e/utils/table';
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
const INSTANCES = [
  INSTANCE_FROM_NAMESPACE_PRESET,
  INSTANCE_FROM_CLUSTER_PRESET,
];

// What every preset of this spec describes: the smallest instance the provider
// takes, because both goldens bring it up next to the PITR and restore ones.
const ENGINE = { replicas: 1, cpu: '1', memoryGi: '1', diskGi: '1' };
const PROXY = { type: 'haproxy', replicas: 1 };
// Monthly, so the overview can describe it and it does not run during the test.
const SCHEDULE = { name: 'monthly', enabled: true, cron: '0 0 1 * *' };
const SCHEDULE_TEXT = 'Monthly on day 1 at 12:00 AM';
const DEMAND_BACKUP = 'e2e-preset-backup';

const clusterUrl = `/v1/clusters/${EVEREST_CI_CLUSTER}`;
const instanceUrl = (name: string) =>
  `${clusterUrl}/namespaces/${NAMESPACE}/instances/${name}`;
const authHeaders = (token: string) => ({ Authorization: `Bearer ${token}` });

const presetSpec = (version: string, extra: PresetSpec = {}): PresetSpec => ({
  providerRef: { name: PROVIDER },
  version,
  topology: { type: 'cluster' },
  components: {
    engine: {
      replicas: ENGINE.replicas,
      resources: {
        limits: { cpu: ENGINE.cpu, memory: `${ENGINE.memoryGi}Gi` },
      },
      storage: { size: `${ENGINE.diskGi}Gi` },
    },
    proxy: PROXY,
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

// A preset is only worth offering if what it describes comes up, so creating
// from one ends when the instance is Ready.
const createFromPreset = async (page: Page, scope: Scope, name: string) => {
  await openWizard(page);
  await pickPreset(page, scope);
  await page.getByTestId('text-input-db-name').fill(name);
  await page.getByTestId('db-wizard-submit-button').click();
  // The wizard goes to the list once the create request succeeds.
  await page.waitForURL('**/databases', { timeout: TIMEOUTS.ThirtySeconds });
  await waitForInstanceStatus(page, name, 'Ready', TIMEOUTS.FifteenMinutes);
};

const getInstance = async (
  request: APIRequestContext,
  token: string,
  name: string
) => {
  const response = await request.get(instanceUrl(name), {
    headers: authHeaders(token),
  });
  expect(response.status(), await response.text()).toBe(200);
  return response.json();
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

const openOverview = async (page: Page, name: string) => {
  await page.goto(`/databases/${NAMESPACE}/${name}/overview`);
  await expect(page.getByTestId(`${name}-status`)).toHaveText('Ready');
};

const escapeRegExp = (text: string) =>
  text.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');

// A row of the overview reads "<label><value>". Some labels are kept on one
// line with a non-breaking space, so any white space matches a space.
const expectOverviewRow = (
  page: Page,
  row: string,
  label: string,
  value: string | number
) =>
  expect(page.getByTestId(`${row}-overview-section-row`)).toHaveText(
    new RegExp(
      `^${escapeRegExp(label).replace(/ /g, '\\s')}\\s*${escapeRegExp(String(value))}$`
    )
  );

// The overview and the edit dialogs read the instance back from the server, so
// they show what was really applied, not what the wizard sent.
const expectPresetOnOverview = async (page: Page, version: string) => {
  await expectOverviewRow(
    page,
    'database-version',
    'Database Version',
    version
  );
  await expectOverviewRow(
    page,
    'number-of-nodes',
    'Number of nodes',
    ENGINE.replicas
  );
  await expectOverviewRow(page, 'cpu', 'CPU', ENGINE.cpu);
  await expectOverviewRow(page, 'memory', 'Memory', `${ENGINE.memoryGi} Gi`);
  await expectOverviewRow(page, 'disk', 'Disk', `${ENGINE.diskGi} Gi`);
  await expectOverviewRow(page, 'proxy-type', 'Proxy type', PROXY.type);
  await expectOverviewRow(
    page,
    'number-of-proxy-nodes',
    'Number of proxy nodes',
    PROXY.replicas
  );
};

const expectPresetInEditDialogs = async (page: Page) => {
  const dialog = page.getByRole('dialog');
  const field = (path: string) => dialog.getByTestId(`text-input-${path}`);

  await page.getByTestId('resources-edit-button').click();
  await expect(field('spec.components.engine.replicas')).toHaveValue(
    String(ENGINE.replicas)
  );
  await expect(
    field('spec.components.engine.resources.limits.cpu')
  ).toHaveValue(ENGINE.cpu);
  await expect(
    field('spec.components.engine.resources.limits.memory')
  ).toHaveValue(ENGINE.memoryGi);
  await expect(field('spec.components.engine.storage.size')).toHaveValue(
    ENGINE.diskGi
  );
  await dialog.getByTestId('form-dialog-cancel').click();
  await expect(dialog).toBeHidden();

  await page.getByTestId('proxy-edit-button').click();
  await expect(
    dialog.getByTestId('select-input-spec.components.proxy.type')
  ).toHaveValue(PROXY.type);
  await expect(field('spec.components.proxy.replicas')).toHaveValue(
    String(PROXY.replicas)
  );
  await dialog.getByTestId('form-dialog-cancel').click();
  await expect(dialog).toBeHidden();
};

test.describe('Instance presets', { tag: '@release' }, () => {
  // One worker, in order: the tests share the presets created below. Unlike
  // serial mode, a failed test does not skip the ones after it.
  test.describe.configure({ mode: 'default' });
  // Without it a click on something that is not there waits out the whole test
  // timeout, which the goldens set to half an hour.
  test.use({ actionTimeout: TIMEOUTS.ThirtySeconds });

  let token: string;
  let clusterPresetSpec: PresetSpec;
  let namespacePresetSpec: PresetSpec;
  let clusterVersion: string;
  let namespaceVersion: string;
  let otherNamespaceVersion: string;
  const backupStorage = getBackupStorage()!;

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
    [clusterVersion, namespaceVersion, otherNamespaceVersion] =
      provider.spec.versions.map((version: { name: string }) => version.name);
    expect(otherNamespaceVersion, 'three provider versions').toBeTruthy();

    for (const name of INSTANCES) {
      await deleteInstance(request, token, name);
    }

    clusterPresetSpec = presetSpec(clusterVersion);
    // A namespace preset can pin what lives in its namespace: here, the storage
    // the backups go to.
    namespacePresetSpec = presetSpec(namespaceVersion, {
      backup: {
        enabled: true,
        classRef: { name: BACKUP_CLASS },
        storages: [
          {
            storageRef: { name: backupStorage },
            schedules: [SCHEDULE],
            pitr: { enabled: true },
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
    // The goldens outlive the token taken in beforeAll.
    const cleanupToken = await getCITokenFromLocalStorage();
    for (const name of INSTANCES) {
      await deleteInstance(request, cleanupToken, name);
    }
    deleteClusterInstancePreset(PRESET);
    await deleteInstancePreset(request, cleanupToken, NAMESPACE, PRESET);
    await deleteInstancePreset(request, cleanupToken, OTHER_NAMESPACE, PRESET);
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
      `${SCHEDULE_TEXT} → ${backupStorage}`
    );
    await expect(page.getByTestId('section-backups')).toContainText(
      `PITR: enabled on ${backupStorage}`
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
      backupStorage
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

  // The golden path of a namespace preset: from the card to a running instance
  // that is what the preset says, backups included.
  test.describe('Namespace preset golden', () => {
    // Each step works on the instance the first one creates. Waiting for an
    // instance a second time would not turn a stuck one green.
    test.describe.configure({ mode: 'serial', retries: 0 });
    test.setTimeout(TIMEOUTS.ThirtyMinutes);
    const name = INSTANCE_FROM_NAMESPACE_PRESET;

    test('brings up an instance that is a copy of the preset', async ({
      page,
      request,
    }) => {
      await createFromPreset(page, 'Namespace', name);

      const instance = await getInstance(request, token, name);
      expect(instance.metadata.annotations).toMatchObject({
        [NAMESPACE_PRESET_ANNOTATION]: PRESET,
      });
      expect(instance.metadata.annotations).not.toHaveProperty(
        CLUSTER_PRESET_ANNOTATION
      );
      expect(instance.spec).toMatchObject(namespacePresetSpec);
    });

    test('shows what the preset set on the overview and in the edit dialogs', async ({
      page,
    }) => {
      await openOverview(page, name);

      await expectPresetOnOverview(page, namespaceVersion);
      await expect(
        page.getByTestId('schedules-overview-section')
      ).toContainText(SCHEDULE_TEXT);
      await expectOverviewRow(page, 'pitr-status', 'Status', 'Enabled');
      await expectOverviewRow(
        page,
        'backup-storage',
        'Backup storage',
        backupStorage
      );
      await expectPresetInEditDialogs(page);
    });

    test('has the schedule and the PITR of the preset on the Backups tab', async ({
      page,
    }) => {
      await openOverview(page, name);
      await page.getByTestId('backups').click();

      await page.getByTestId('scheduled-backups').click();
      const schedule = page.getByTestId(`schedule-${SCHEDULE.name}`);
      await expect(schedule).toContainText(SCHEDULE_TEXT);
      await expect(schedule).toContainText(`Storage: ${backupStorage}`);

      await page.getByTestId('storages-toggle').click();
      await expect(
        page.getByTestId(`pitr-toggle-${backupStorage}`).getByRole('switch')
      ).toBeChecked();
    });

    test('takes a backup to the storage the preset pins', async ({ page }) => {
      await openOverview(page, name);
      await page.getByTestId('backups').click();

      await clickOnDemandBackup(page);
      const dialog = page.getByRole('dialog');
      await expect(dialog.getByTestId('text-input-storage-name')).toHaveValue(
        backupStorage
      );
      await dialog.getByTestId('text-input-name').fill(DEMAND_BACKUP);
      await dialog.getByRole('button', { name: /^create$/i }).click();

      await waitForStatus(
        page,
        DEMAND_BACKUP,
        'Succeeded',
        TIMEOUTS.TenMinutes
      );
    });

    test('takes data, and PITR covers it', async ({ request }) => {
      // Writes three rows through the proxy and reads them back.
      await prepareTestDB(name, NAMESPACE);

      // PITR is recoverable once the binlogs of a write that follows a full
      // backup reach the storage.
      await expect
        .poll(
          async () => {
            const instance = await getInstance(
              request,
              await getCITokenFromLocalStorage(),
              name
            );
            return instance.status?.backup?.storages?.find(
              (storage: { name: string }) => storage.name === backupStorage
            )?.pitr?.state;
          },
          { timeout: TIMEOUTS.TenMinutes, intervals: [5000, 5000, 10000] }
        )
        .toBe('Available');
    });
  });

  // A cluster-wide preset pins nothing of a namespace, so its golden ends at a
  // running instance that shows the preset.
  test.describe('Cluster-wide preset golden', () => {
    test.describe.configure({ mode: 'serial', retries: 0 });
    test.setTimeout(TIMEOUTS.ThirtyMinutes);
    const name = INSTANCE_FROM_CLUSTER_PRESET;

    test('brings up an instance that is a copy of the preset', async ({
      page,
      request,
    }) => {
      await createFromPreset(page, 'Cluster-wide', name);

      const instance = await getInstance(request, token, name);
      expect(instance.metadata.annotations).toMatchObject({
        [CLUSTER_PRESET_ANNOTATION]: PRESET,
      });
      expect(instance.metadata.annotations).not.toHaveProperty(
        NAMESPACE_PRESET_ANNOTATION
      );
      expect(instance.spec).toMatchObject(clusterPresetSpec);
    });

    test('shows what the preset set on the overview and in the edit dialogs', async ({
      page,
    }) => {
      await openOverview(page, name);

      await expectPresetOnOverview(page, clusterVersion);
      // The preset has no backups, so neither has the instance.
      await expect(
        page.getByTestId('schedules-overview-section')
      ).toContainText('0 active schedules');
      await expectOverviewRow(page, 'pitr-status', 'Status', 'Disabled');
      await expectPresetInEditDialogs(page);
    });
  });
});
