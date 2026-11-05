// everest
// Copyright (C) 2023 Percona LLC
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

import {
  request as apiRequest,
  APIRequestContext,
  Page,
} from '@playwright/test';
import { expect, test } from '@e2e/fixtures/auth';
import {
  gotoDbClusterBackups,
  gotoDbClusterRestores,
  findDbAndClickActions,
} from '@e2e/utils/db-clusters-list';
import { getCITokenFromLocalStorage } from '@e2e/utils/localStorage';
import { getClusterDetailedInfo } from '@e2e/utils/storage-class';
import {
  moveForward,
  submitWizard,
  populateBasicInformation,
  populateResources,
  populateEngineResources,
  populateAdvancedConfig,
  openDbCreationForm,
} from '@e2e/utils/db-wizard';
import {
  EVEREST_CI_NAMESPACES,
  getBucketNamespacesMap,
  TIMEOUTS,
} from '@e2e/constants';
import {
  waitForStatus,
  waitForDelete,
  findRowAndClickActions,
  waitForDbListLoad,
} from '@e2e/utils/table';
import { clickOnDemandBackup } from '@e2e/pr/db-cluster-details/utils';
import {
  prepareTestDB,
  dropTestDB,
  queryTestDB,
  insertTestDB,
} from '@e2e/utils/db-cmd-line';
import { addFirstScheduleInDBWizard } from '@e2e/pr/db-cluster/db-wizard/db-wizard-utils';
import { getDbClusterAPI } from '@e2e/utils/db-cluster';
import { shouldExecuteDBCombination } from '@e2e/utils/generic';
import { dismissOnboarding, loginCIUser } from '@e2e/utils/user';

type PitrTime = {
  day: string;
  month: string;
  year: string;
  hour: string;
  minute: string;
  second: string;
};

let token: string;
let backupStorage: string;
let firstRestoreName: string;
let pitrRestoreTime: PitrTime = {
  day: '',
  month: '',
  year: '',
  hour: '',
  minute: '',
  second: '',
};

function getCurrentPITRTime(): PitrTime {
  const now: Date = new Date();
  return {
    day: now.getDate().toString(),
    month: (now.getMonth() + 1).toString(), // Months are zero-indexed.
    year: now.getFullYear().toString(),
    hour: now.getHours().toString(),
    minute: now.getMinutes().toString(),
    second: now.getSeconds().toString(),
  };
}

function getFormattedPITRTime(time: PitrTime): string {
  return `${time.day.padStart(2, '0')}/${time.month.padStart(2, '0')}/${time.year} at ${time.hour.padStart(2, '0')}:${time.minute.padStart(2, '0')}:${time.second.padStart(2, '0')}`;
}

function getBackupStorage(): string {
  const bucketNamespacesMap = getBucketNamespacesMap();
  const hasEverestTesting = bucketNamespacesMap.some(
    ([bucket, ns]) => bucket === 'everest-testing' && ns === 'everest'
  );

  if (
    process.env.EVEREST_S3_ACCESS_KEY &&
    process.env.EVEREST_S3_SECRET_KEY &&
    hasEverestTesting
  ) {
    return 'everest-testing';
  }
  return 'bucket-1';
}

// The dialog sends the restore request asynchronously and navigating away
// before it settles aborts it, so wait for the response before leaving the page.
async function submitRestoreDialog(page: Page, namespace: string) {
  const restoreResponse = page.waitForResponse(
    (resp) =>
      resp.request().method() === 'POST' &&
      resp.url().endsWith(`/namespaces/${namespace}/restores`)
  );
  await page.getByTestId('form-dialog-restore').click({ timeout: 10000 });
  const response = await restoreResponse;
  expect(response.ok(), await response.text()).toBeTruthy();
}

test.describe.configure({ retries: 0 });

// Reference provider for the UI golden path. Flag/engine variations are covered
// by unit tests + curated fixtures (see the roadmap), not by a live matrix.
const db = 'pxc';
const size = 3;
// v2 empty-state tiles and the toolbar drawer are keyed by the Provider CR
// metadata.name (not the engine short-code), so DB creation is entered by
// provider name.
const PXC_PROVIDER_NAME = 'percona-xtradb-cluster';

test.describe.serial(
  'PITR golden (PXC)',
  {
    tag: '@release',
  },
  () => {
    test.skip(!shouldExecuteDBCombination(db, size));
    test.describe.configure({ timeout: 1_800_000 }); // 30 minutes

    const clusterName = `${db}-${size}-pitr`;
    const namespace = EVEREST_CI_NAMESPACES.EVEREST_UI;
    const baseBackupName = `dembkp-${db}-${size}`;
    // Keep PXC's generated restore Job labels within 63 bytes.
    const restoredClusterNames = [
      `${clusterName}-date`,
      `${clusterName}-latest`,
    ];
    let seedRestoreTime: PitrTime;
    let storageClasses: string[] = [];
    // Start of the latest insert batch; the PITR waits below poll until the
    // recovery window covers it.
    let lastInsertStartedAt: Date;

    test.beforeAll(async ({ request }) => {
      token = await getCITokenFromLocalStorage();
      const { storageClassNames = [] } = await getClusterDetailedInfo(
        token,
        request
      );
      storageClasses = storageClassNames;
      backupStorage = getBackupStorage();
    });

    // Best-effort teardown: delete the cluster (which cascades its backups and
    // restores) via the API so the suite leaves nothing behind. Runs even when
    // a step failed — unlike a trailing "delete" test, which serial mode would
    // skip after a failure — keeping the shared namespace clean so re-runs and
    // future parallelization of the release lane start from a clean slate.
    test.afterAll(async () => {
      try {
        const cleanupToken = await getCITokenFromLocalStorage();
        const ctx = await apiRequest.newContext({
          baseURL: process.env.EVEREST_URL || 'http://localhost:8080',
        });
        for (const name of [...restoredClusterNames, clusterName]) {
          await ctx.delete(
            `/v1/clusters/main/namespaces/${namespace}/instances/${name}`,
            { headers: { Authorization: `Bearer ${cleanupToken}` } }
          );
        }
        await ctx.dispose();
      } catch {
        // Teardown must never fail the suite; any leftover cluster is removed by
        // the next run's pre-create delete.
      }
    });

    // Poll the instance's PITR recovery window until it covers `target` — i.e.
    // the binlogs up to that moment have been uploaded to the PITR storage. This
    // is the exact precondition a point-in-time restore needs, so the wait is
    // both correct and self-documenting (vs. a blind fixed sleep).
    //
    // `target` must be a moment at or before the last write we need restored,
    // never "now": the PXC binlog collector reports the timestamp of the last
    // uploaded transaction truncated to whole seconds, and it only advances when
    // a new transaction is written. Waiting for a moment after our last write
    // would depend on some unrelated later write that may never happen.
    const waitForPitrWindowToCover = async (
      request: APIRequestContext,
      target: Date
    ) => {
      const instancePath = `/v1/clusters/main/namespaces/${namespace}/instances/${clusterName}`;
      const targetSecond = Math.floor(target.getTime() / 1000) * 1000;
      await expect
        .poll(
          async () => {
            // The suite outlives the token taken in beforeAll; this returns a cached, fresh one.
            const response = await request.get(instancePath, {
              headers: {
                Authorization: `Bearer ${await getCITokenFromLocalStorage()}`,
              },
            });
            if (response.status() !== 200) {
              return false;
            }
            const instance = await response.json();
            const storages = instance?.status?.backup?.storages ?? [];
            return storages.some(
              (storage: { pitr?: { latestRestorableTime?: string } }) => {
                const latest = storage.pitr?.latestRestorableTime;
                return (
                  latest != null && new Date(latest).getTime() >= targetSecond
                );
              }
            );
          },
          { timeout: 480000, intervals: [5000, 5000, 10000] }
        )
        .toBe(true);
    };

    // Create an on-demand backup with the given name against the suite's backup
    // storage and wait for it to succeed. Shared by both demand backups so they
    // use the same resilient combobox-driven flow.
    const createDemandBackup = async (page: Page, backupName: string) => {
      await gotoDbClusterBackups(page, clusterName);
      await clickOnDemandBackup(page);
      await page.getByTestId('text-input-name').fill(backupName);
      await expect(page.getByTestId('text-input-name')).not.toBeEmpty();

      const storageCombobox = page.getByRole('combobox', {
        name: /backup storage/i,
      });
      await expect(storageCombobox).toBeVisible();

      const currentStorage = (await storageCombobox.inputValue()).trim();
      if (currentStorage !== backupStorage) {
        await storageCombobox.click();
        await page.getByRole('option', { name: `${backupStorage}` }).click();
      }

      await expect(storageCombobox).toHaveValue(backupStorage);

      const createButton = page
        .getByRole('dialog')
        .getByRole('button', { name: /^create$/i });
      await expect(createButton).toBeVisible();
      await createButton.click();

      await waitForStatus(page, backupName, 'Succeeded', 360000);
    };

    test(`Cluster creation with PITR [${db} size ${size}]`, async ({
      page,
      request,
    }) => {
      expect(storageClasses.length).toBeGreaterThan(0);

      for (const name of [...restoredClusterNames, clusterName]) {
        const instancePath = `/v1/clusters/main/namespaces/${namespace}/instances/${name}`;
        const headers = {
          Authorization: `Bearer ${await getCITokenFromLocalStorage()}`,
        };
        const cleanupResponse = await request.delete(instancePath, { headers });
        expect([204, 404]).toContain(cleanupResponse.status());

        await expect
          .poll(
            async () =>
              (
                await request.get(instancePath, {
                  headers: {
                    Authorization: `Bearer ${await getCITokenFromLocalStorage()}`,
                  },
                })
              ).status(),
            { timeout: 180000, intervals: [2000, 5000, 5000] }
          )
          .toBe(404);
      }

      await expect
        .poll(
          async () => {
            const storageResponse = await request.get(
              `/v1/clusters/main/namespaces/${namespace}/backup-storages/${backupStorage}`,
              {
                headers: {
                  Authorization: `Bearer ${token}`,
                },
              }
            );
            return storageResponse.status();
          },
          {
            timeout: 120000,
            intervals: [2000, 5000, 5000],
          }
        )
        .toBe(200);

      await page.goto('/databases');
      await openDbCreationForm(page, PXC_PROVIDER_NAME);

      await test.step('Populate basic information', async () => {
        await populateBasicInformation(
          page,
          namespace,
          clusterName,
          db,
          storageClasses[0],
          false,
          ''
        );
        await moveForward(page);
      });

      await test.step('Populate resources', async () => {
        if (db !== 'pxc') {
          await page
            .getByRole('button')
            .getByText(size + ' node')
            .click();
          await expect(page.getByText('Nodes (' + size + ')')).toBeVisible();
          await populateResources(page, 0.6, 1, 1, size);
          await moveForward(page);
        } else {
          // Resources comes after Database Version in the PXC wizard.
          await moveForward(page);
          // The topology defaults (4 CPU / 8Gi / 100Gi per node) do not fit a CI runner.
          // Whole CPUs only: the wizard sends 0.6 as a JSON float, which the API rejects.
          await populateEngineResources(page, 1, 1, 1);
        }
      });

      await test.step('Add a schedule and enable per-storage PITR', async () => {
        await addFirstScheduleInDBWizard(page, backupStorage);

        const pitrToggle = page
          .getByTestId(`pitr-toggle-${backupStorage}`)
          .getByRole('switch');
        await expect(pitrToggle).toBeVisible();
        await expect(pitrToggle).not.toBeChecked();
        await pitrToggle.check();
        await expect(pitrToggle).toBeChecked();

        await moveForward(page);
      });

      // TODO(monitoring): the PXC provider dropped the wizard Monitoring step and
      // spec.components.monitoring in favour of MonitoringDestination
      // (openeverest/provider-percona-xtradb-cluster#46). Restore monitoring
      // coverage here once core and the UI support MonitoringDestination.

      await test.step('Populate advanced db config', async () => {
        if (db === 'pxc') {
          return;
        }

        await populateAdvancedConfig(page, db, false, '', true, '');
        await moveForward(page);
      });

      await test.step('Submit wizard', async () => {
        await submitWizard(page);
      });

      await test.step('Check db list and status', async () => {
        // The wizard navigates to the list once the create request succeeds;
        // navigating ourselves would abort the in-flight request.
        await page.waitForURL('**/databases', {
          timeout: TIMEOUTS.ThirtySeconds,
        });
        try {
          await waitForStatus(page, clusterName, 'Provisioning', 30000);
        } catch {
          // Transitional status can be short-lived; continue with readiness checks.
        }
        await waitForStatus(page, clusterName, 'Ready', 900000);
      });

      await test.step('Verify PITR is enabled on the created cluster', async () => {
        // Fail fast: confirm the wizard actually persisted PITR before the long
        // backup/data/wait cycle. getDbClusterAPI hits the translated
        // /v1/database-clusters endpoint, which exposes the flat spec.backup
        // shape (same as the schedule assertions in create-db-cluster.e2e.ts).
        // TODO(live): confirm the exact PITR field path this endpoint returns in
        // v2 (flat spec.backup.pitr vs per-storage spec.backup.storages[].pitr).
        const cluster = await getDbClusterAPI(
          clusterName,
          namespace,
          request,
          token
        );
        expect(cluster?.spec?.engine?.type).toBe(db);
        expect(cluster?.spec?.engine?.replicas).toBe(size);

        await expect
          .poll(
            async () => {
              const latestCluster = await getDbClusterAPI(
                clusterName,
                namespace,
                request,
                token
              );
              return latestCluster?.spec?.backup?.pitr?.enabled === true;
            },
            {
              timeout: 60000,
              intervals: [2000, 5000, 5000],
            }
          )
          .toBe(true);
      });
    });

    test(`Add data [${db} size ${size}]`, async ({ request }) => {
      let addDataToken = token;

      await expect
        .poll(
          async () => {
            const connectionResponse = await request.get(
              `/v1/clusters/main/namespaces/${namespace}/instances/${clusterName}/connection`,
              {
                headers: {
                  Authorization: `Bearer ${addDataToken}`,
                },
              }
            );

            if (connectionResponse.status() === 401) {
              addDataToken = await getCITokenFromLocalStorage();
              token = addDataToken;
            }

            return connectionResponse.status();
          },
          {
            timeout: 20 * 60 * 1000,
            intervals: [5000, 10000, 15000],
          }
        )
        .toBe(200);

      await prepareTestDB(clusterName, namespace);
    });

    test(`Create base demand backup [${db} size ${size}]`, async ({ page }) => {
      // The base full backup is the PITR recovery anchor. A demand ("now")
      // backup is deterministic and faster than waiting for the schedule cron;
      // it is functionally identical to a scheduled backup as the anchor.
      await createDemandBackup(page, `${baseBackupName}-1`);
    });

    test(`Add more data and record restore point [${db} size ${size}]`, async () => {
      // State at T = rows 1..6; the rows added after T (7..9) must NOT survive
      // a restore to T.
      await insertTestDB(
        clusterName,
        namespace,
        ['4', '5', '6'],
        ['1', '2', '3', '4', '5', '6']
      );
      // The restore replays binlogs with --stop-datetime, which is second-granular and exclusive.
      await new Promise((resolve) => setTimeout(resolve, 2000));
      pitrRestoreTime = getCurrentPITRTime();
      await new Promise((resolve) => setTimeout(resolve, 2000));
      lastInsertStartedAt = new Date();
      await insertTestDB(
        clusterName,
        namespace,
        ['7', '8', '9'],
        ['1', '2', '3', '4', '5', '6', '7', '8', '9']
      );
    });

    test(`Wait for binlogs to be uploaded [${db} size ${size}]`, async ({
      request,
    }) => {
      // The recovery window must cover the inserts above before a
      // point-in-time restore can target a moment within it. Poll the uploaded
      // binlog watermark instead of sleeping a fixed, environment-fragile time.
      await waitForPitrWindowToCover(request, lastInsertStartedAt);
    });

    test(`Delete data before date restore [${db} size ${size}]`, async () => {
      await dropTestDB(clusterName, namespace);
    });

    test(`Restore to a specific date [${db} size ${size}]`, async ({
      page,
    }) => {
      let clusterRowFound = false;
      for (let attempt = 0; attempt < 6; attempt += 1) {
        await page.goto('/databases');

        const onLoginPage =
          page.url().includes('/login') ||
          (await page
            .getByRole('heading', { name: 'Log in' })
            .isVisible({ timeout: 3000 })
            .catch(() => false));

        if (onLoginPage) {
          await loginCIUser(page);
          await dismissOnboarding(page);
          continue;
        }

        await waitForDbListLoad(page);
        if (
          (await page
            .locator('.MuiTableRow-root')
            .filter({ hasText: clusterName })
            .count()) > 0
        ) {
          clusterRowFound = true;
          break;
        }

        await page.reload({ waitUntil: 'networkidle' });
      }

      expect(clusterRowFound).toBeTruthy();

      await findDbAndClickActions(page, clusterName, 'Restore from a backup');

      await page.getByTestId('radio-option-fromPITR').click({ timeout: 5000 });
      await expect(page.getByTestId('radio-option-fromPITR')).toBeChecked({
        timeout: 5000,
      });

      // v2: choose the date target; the picker appears only for 'date'.
      await page.getByTestId('radio-option-date').click({ timeout: 5000 });
      await expect(page.getByPlaceholder('DD/MM/YYYY at hh:mm:ss')).toBeVisible(
        { timeout: 5000 }
      );

      // v2 (MUI x-date-pickers v7): the open-picker button is exposed by its
      // accessible name ("Choose date…"), not a `CalendarIcon` test id.
      await page
        .getByRole('button', { name: /choose date/i })
        .click({ timeout: 5000 });
      await page
        .getByLabel(pitrRestoreTime.hour + ' hours', { exact: true })
        .click({ timeout: 5000 });
      await page
        .getByLabel(pitrRestoreTime.minute + ' minutes', { exact: true })
        .click({ timeout: 5000 });
      await page
        .getByLabel(pitrRestoreTime.second + ' seconds', { exact: true })
        .click({ timeout: 5000 });
      await expect(page.getByPlaceholder('DD/MM/YYYY at hh:mm:ss')).toHaveValue(
        getFormattedPITRTime(pitrRestoreTime)
      );

      await submitRestoreDialog(page, namespace);

      await page.goto('/databases');
      await waitForStatus(page, clusterName, 'Restoring', 30000);
      await waitForStatus(page, clusterName, 'Ready', 900000);

      await gotoDbClusterRestores(page, clusterName);
      await waitForStatus(page, 'restore-', 'Succeeded', 120000);

      // Restores get auto-generated random names and a PITR restore row shows
      // the storage (not the source backup), so capture this one's name now to
      // delete it below and keep the post-latest-restore assertion unambiguous.
      firstRestoreName =
        (
          await page
            .getByRole('row')
            .filter({ hasText: 'restore-' })
            .getByText(/^restore-/)
            .first()
            .textContent()
        )?.trim() ?? '';
      expect(firstRestoreName).toMatch(/^restore-/);
    });

    test(`Check data after date restore [${db} size ${size}]`, async () => {
      const result = await queryTestDB(clusterName, namespace);
      expect(result.trim()).toBe('1\n2\n3\n4\n5\n6');
    });

    test(`Delete first restore [${db} size ${size}]`, async ({ page }) => {
      // Remove the first (date) restore, targeted by the random name captured
      // above, so the single-restore assertion after the latest restore below
      // does not collide with two 'restore-' rows.
      await gotoDbClusterRestores(page, clusterName);
      await findRowAndClickActions(page, firstRestoreName, 'Delete');
      await expect(page.getByLabel('Delete restore')).toBeVisible();
      await page.getByTestId('confirm-dialog-delete').click();
      // Restore cleanup is a provider finalizer that can wait a 10s requeue.
      await waitForDelete(page, firstRestoreName, TIMEOUTS.OneMinute);
    });

    test(`Create second demand backup [${db} size ${size}]`, async ({
      page,
    }) => {
      await createDemandBackup(page, `${baseBackupName}-2`);
    });

    test(`Add more data before latest restore [${db} size ${size}]`, async () => {
      lastInsertStartedAt = new Date();
      await insertTestDB(
        clusterName,
        namespace,
        ['7', '8'],
        ['1', '2', '3', '4', '5', '6', '7', '8']
      );
    });

    test(`Wait for binlogs before latest restore [${db} size ${size}]`, async ({
      request,
    }) => {
      // Same precondition as before the date restore: wait until the recovery
      // window covers the just-inserted rows so "restore to latest" includes
      // them, rather than relying on a fixed sleep.
      await waitForPitrWindowToCover(request, lastInsertStartedAt);
    });

    test(`Delete data before latest restore [${db} size ${size}]`, async () => {
      await dropTestDB(clusterName, namespace);
    });

    test(`Restore to the latest moment [${db} size ${size}]`, async ({
      page,
    }) => {
      await page.goto('databases');
      await findDbAndClickActions(page, clusterName, 'Restore from a backup');

      await page.getByTestId('radio-option-fromPITR').click({ timeout: 10000 });
      await expect(page.getByTestId('radio-option-fromPITR')).toBeChecked({
        timeout: 10000,
      });

      // 'latest' is the default recovery target and needs no date input.
      await expect(page.getByTestId('radio-option-latest')).toBeChecked({
        timeout: 10000,
      });

      await submitRestoreDialog(page, namespace);

      await page.goto('/databases');
      await waitForStatus(page, clusterName, 'Restoring', 30000);
      await waitForStatus(page, clusterName, 'Ready', 900000);

      await gotoDbClusterRestores(page, clusterName);
      await waitForStatus(page, 'restore-', 'Succeeded', 120000);
    });

    test(`Check data after latest restore [${db} size ${size}]`, async () => {
      const result = await queryTestDB(clusterName, namespace);
      expect(result.trim()).toBe('1\n2\n3\n4\n5\n6\n7\n8');
    });

    test('Prepare source data for PITR restores to new databases', async ({
      page,
      request,
    }) => {
      await createDemandBackup(page, `${baseBackupName}-seed`);
      await insertTestDB(
        clusterName,
        namespace,
        ['9'],
        ['1', '2', '3', '4', '5', '6', '7', '8', '9']
      );
      // PXC's --stop-datetime is second-granular and exclusive, so keep the
      // recorded point strictly between the two committed inserts.
      await new Promise((resolve) => setTimeout(resolve, 2000));
      seedRestoreTime = getCurrentPITRTime();
      await new Promise((resolve) => setTimeout(resolve, 2000));
      await insertTestDB(
        clusterName,
        namespace,
        ['10'],
        ['1', '2', '3', '4', '5', '6', '7', '8', '9', '10']
      );
      await waitForPitrWindowToCover(request, new Date());
    });

    for (const { recoveryTarget, expectedData } of [
      { recoveryTarget: 'date', expectedData: '1\n2\n3\n4\n5\n6\n7\n8\n9' },
      {
        recoveryTarget: 'latest',
        expectedData: '1\n2\n3\n4\n5\n6\n7\n8\n9\n10',
      },
    ]) {
      test(`PITR restore to a new database (${recoveryTarget}) [${db} size ${size}]`, async ({
        page,
        request,
      }) => {
        const restoredClusterName = `${clusterName}-${recoveryTarget}`;
        const instancePath = `/v1/clusters/main/namespaces/${namespace}/instances`;
        const sourceResponse = await request.get(
          `${instancePath}/${clusterName}`,
          {
            headers: {
              Authorization: `Bearer ${await getCITokenFromLocalStorage()}`,
            },
          }
        );
        expect(sourceResponse.ok()).toBeTruthy();
        const sourceBefore = await sourceResponse.json();
        expect(sourceBefore.metadata.uid).toBeTruthy();

        try {
          await test.step('Carry the PITR target through the restore wizard', async () => {
            await page.goto('/databases');
            await findDbAndClickActions(
              page,
              clusterName,
              'Create DB from a backup'
            );
            await page.getByTestId('radio-option-fromPITR').click();
            await expect(
              page.getByTestId('radio-option-fromPITR')
            ).toBeChecked();
            await expect(page.getByTestId('radio-option-latest')).toBeChecked();

            let date: string | undefined;
            if (recoveryTarget === 'date') {
              await page.getByTestId('radio-option-date').click();
              const dateInput = page.getByPlaceholder('DD/MM/YYYY at hh:mm:ss');
              await dateInput.fill(getFormattedPITRTime(seedRestoreTime));
              await dateInput.press('Tab');
              await expect(dateInput).toHaveValue(
                getFormattedPITRTime(seedRestoreTime)
              );
              date = new Date(
                Number(seedRestoreTime.year),
                Number(seedRestoreTime.month) - 1,
                Number(seedRestoreTime.day),
                Number(seedRestoreTime.hour),
                Number(seedRestoreTime.minute),
                Number(seedRestoreTime.second)
              )
                .toISOString()
                .replace('.000Z', 'Z');
            }

            await page.getByTestId('form-dialog-create').click();
            await page.waitForURL('**/databases/new');
            await page
              .getByTestId('text-input-db-name')
              .fill(restoredClusterName);

            const createResponse = page.waitForResponse(
              (response) =>
                response.request().method() === 'POST' &&
                response.url().endsWith(instancePath)
            );
            await submitWizard(page);
            const response = await createResponse;
            expect(response.ok(), await response.text()).toBeTruthy();
            expect(response.request().postDataJSON()).toHaveProperty(
              'spec.dataSource',
              {
                type: 'PointInTime',
                pointInTime: {
                  recoveryTarget,
                  source: {
                    instanceRef: { name: clusterName },
                    storageRef: { name: backupStorage },
                  },
                  ...(date ? { date } : {}),
                },
              }
            );
          });

          await test.step('Verify the recovered data and unchanged source', async () => {
            await page.goto('/databases');
            await waitForStatus(page, restoredClusterName, 'Ready', 900000);
            expect(
              (await queryTestDB(restoredClusterName, namespace)).trim()
            ).toBe(expectedData);
            expect((await queryTestDB(clusterName, namespace)).trim()).toBe(
              '1\n2\n3\n4\n5\n6\n7\n8\n9\n10'
            );

            const sourceAfter = await request.get(
              `${instancePath}/${clusterName}`,
              {
                headers: {
                  Authorization: `Bearer ${await getCITokenFromLocalStorage()}`,
                },
              }
            );
            expect(sourceAfter.ok()).toBeTruthy();
            const source = await sourceAfter.json();
            expect(source.metadata.uid).toBe(sourceBefore.metadata.uid);
            expect(source.spec).toEqual(sourceBefore.spec);
          });
        } finally {
          await test.step('Delete the restored database before the next case', async () => {
            const restoredPath = `${instancePath}/${restoredClusterName}`;
            const headers = {
              Authorization: `Bearer ${await getCITokenFromLocalStorage()}`,
            };
            const response = await request.delete(restoredPath, { headers });
            expect([204, 404]).toContain(response.status());
            await expect
              .poll(
                async () =>
                  (
                    await request.get(restoredPath, {
                      headers: {
                        Authorization: `Bearer ${await getCITokenFromLocalStorage()}`,
                      },
                    })
                  ).status(),
                { timeout: 180000, intervals: [2000, 5000, 5000] }
              )
              .toBe(404);
          });
        }
      });
    }
  }
);
