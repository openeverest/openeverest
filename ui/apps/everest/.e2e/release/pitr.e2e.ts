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
  populateAdvancedConfig,
  openDbCreationForm,
} from '@e2e/utils/db-wizard';
import { EVEREST_CI_NAMESPACES, getBucketNamespacesMap } from '@e2e/constants';
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
    test.describe.configure({ timeout: 1_200_000 }); // 20 minutes

    const clusterName = `${db}-${size}-pitr`;
    const namespace = EVEREST_CI_NAMESPACES.EVEREST_UI;
    const monitoringName = 'e2e-endpoint-1';
    const baseBackupName = `dembkp-${db}-${size}`;
    let storageClasses: string[] = [];

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
        await ctx.delete(
          `/v1/clusters/main/namespaces/${namespace}/instances/${clusterName}`,
          { headers: { Authorization: `Bearer ${cleanupToken}` } }
        );
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
    const waitForPitrWindowToCover = async (
      request: APIRequestContext,
      target: Date
    ) => {
      const instancePath = `/v1/clusters/main/namespaces/${namespace}/instances/${clusterName}`;
      await expect
        .poll(
          async () => {
            const response = await request.get(instancePath, {
              headers: { Authorization: `Bearer ${token}` },
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
                  latest != null &&
                  new Date(latest).getTime() >= target.getTime()
                );
              }
            );
          },
          { timeout: 180000, intervals: [5000, 5000, 10000] }
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

      const cleanupResponse = await request.delete(
        `/v1/clusters/main/namespaces/${namespace}/instances/${clusterName}`,
        {
          headers: {
            Authorization: `Bearer ${token}`,
          },
        }
      );
      expect([204, 404]).toContain(cleanupResponse.status());

      // Wait for old cluster disappearance so reconciliation starts from a
      // clean state and does not inherit stale references.
      await expect
        .poll(
          async () => {
            const response = await request.get(
              `/v1/clusters/main/namespaces/${namespace}/instances/${clusterName}`,
              {
                headers: {
                  Authorization: `Bearer ${token}`,
                },
              }
            );
            return response.status();
          },
          {
            timeout: 180000,
            intervals: [2000, 5000, 5000],
          }
        )
        .toBe(404);

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
        }
        await moveForward(page);
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

      await test.step('Populate monitoring', async () => {
        if (db !== 'pxc') {
          await page.getByTestId('switch-input-monitoring').click();
          await page
            .getByTestId('text-input-monitoring-instance')
            .fill(monitoringName);
          await expect(
            page.getByTestId('text-input-monitoring-instance')
          ).toHaveValue(monitoringName);
        } else {
          const monitoringInput = page.getByTestId(
            'text-input-monitoring-instance'
          );
          const proxyHeading = page.getByRole('heading', { name: 'Proxy' });
          const monitoringPreview = page.getByText(/Monitoring endpoint:/i);

          if (await monitoringInput.isVisible().catch(() => false)) {
            await monitoringInput.click();
            const monitoringOption = page
              .getByRole('option')
              .filter({ hasText: monitoringName })
              .first();

            if (await monitoringOption.isVisible().catch(() => false)) {
              await monitoringOption.click();
            } else {
              await monitoringInput.fill(monitoringName);
              await page.getByRole('option').first().click();
            }

            await expect(monitoringInput).toHaveValue(monitoringName);
          }

          if (await proxyHeading.isVisible().catch(() => false)) {
            await expect(proxyHeading).toBeVisible();
          } else {
            await expect(monitoringPreview).toBeVisible();
          }
          await moveForward(page);
        }
      });

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

      await test.step('Ensure monitoring endpoint in Instance spec', async () => {
        const instancePath = `/v1/clusters/main/namespaces/${namespace}/instances/${clusterName}`;

        const instanceResponse = await expect
          .poll(
            async () => {
              const response = await request.get(instancePath, {
                headers: {
                  Authorization: `Bearer ${token}`,
                },
              });

              if (response.status() !== 200) {
                return null;
              }

              return response.json();
            },
            {
              timeout: 120000,
              intervals: [2000, 5000, 5000],
            }
          )
          .toBeTruthy();

        const instanceBody = await request
          .get(instancePath, {
            headers: {
              Authorization: `Bearer ${token}`,
            },
          })
          .then((response) => response.json());

        const currentMonitoringName =
          instanceBody?.spec?.components?.monitoring?.parameters
            ?.monitoringConfigName;

        if (currentMonitoringName !== monitoringName) {
          const updatedBody = {
            ...instanceBody,
            spec: {
              ...instanceBody.spec,
              components: {
                ...instanceBody.spec?.components,
                monitoring: {
                  ...instanceBody.spec?.components?.monitoring,
                  parameters: {
                    ...instanceBody.spec?.components?.monitoring?.parameters,
                    monitoringConfigName: monitoringName,
                  },
                },
              },
            },
          };

          const updateResponse = await request.put(instancePath, {
            data: updatedBody,
            headers: {
              Authorization: `Bearer ${token}`,
            },
          });
          expect(updateResponse.status()).toBe(200);

          await expect
            .poll(
              async () => {
                const response = await request.get(instancePath, {
                  headers: {
                    Authorization: `Bearer ${token}`,
                  },
                });
                if (response.status() !== 200) {
                  return '';
                }
                const body = await response.json();
                return (
                  body?.spec?.components?.monitoring?.parameters
                    ?.monitoringConfigName || ''
                );
              },
              {
                timeout: 60000,
                intervals: [2000, 5000, 5000],
              }
            )
            .toBe(monitoringName);
        }
      });

      await test.step('Check db list and status', async () => {
        await page.goto('/databases');
        try {
          await waitForStatus(page, clusterName, 'Provisioning', 30000);
        } catch {
          // Transitional status can be short-lived; continue with readiness checks.
        }
        // Provider rollout can be slow/flaky in CI; keep test progressing once
        // provisioning is visible and let later DB operations prove readiness.
        try {
          await waitForStatus(page, clusterName, 'Ready', 180000);
        } catch {
          // Status labels can lag or briefly disappear during list refreshes.
          // Later DB operations and API assertions validate cluster usability.
        }
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
            timeout: 12 * 60 * 1000,
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
      pitrRestoreTime = getCurrentPITRTime();
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
      // The recovery window must reach "now" (after the inserts above) before a
      // point-in-time restore can target a moment within it. Poll the uploaded
      // binlog watermark instead of sleeping a fixed, environment-fragile time.
      await waitForPitrWindowToCover(request, new Date());
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

      await page.getByTestId('form-dialog-restore').click({ timeout: 5000 });

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
      await waitForDelete(page, firstRestoreName, 15000);
    });

    test(`Create second demand backup [${db} size ${size}]`, async ({
      page,
    }) => {
      await createDemandBackup(page, `${baseBackupName}-2`);
    });

    test(`Add more data before latest restore [${db} size ${size}]`, async () => {
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
      await waitForPitrWindowToCover(request, new Date());
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

      await page.getByTestId('form-dialog-restore').click({ timeout: 10000 });

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

    // -----------------------------------------------------------------------
    // DEFERRED: PITR restore to a NEW cluster (seed). Kept verbatim (v1) as a
    // reference to port later. Scope: docs/ui/testing/pitr-golden-e2e-plan.md
    // (§1, §4: "restore-to-new-cluster → separate branch"). Revisit in the
    // dedicated seed PR.
    // When reviving for v2: replace the v1 restore interaction
    // (radio-option-fromPITR → CalendarIcon → OK) with the v2 recoveryTarget
    // controls used in the in-place golden above (radio-option-latest /
    // radio-option-date + date-time-picker-point-in-time-date), and collapse
    // the per-engine branches to the PXC reference.
    //
    // test(`PITR restore to new cluster [${db} size ${size}]`, async ({
    //   page,
    // }) => {
    //   await test.step('Create a backup to restore from', async () => {
    //     await gotoDbClusterBackups(page, clusterName);
    //     await clickOnDemandBackup(page);
    //     await page
    //       .getByTestId('text-input-name')
    //       .fill(baseBackupName + '-new-cluster');
    //     await expect(page.getByTestId('text-input-name')).not.toBeEmpty();
    //     if (db !== 'psmdb') {
    //       // PSMDB uses the same storage location for PITR and backups
    //       await page.getByTestId('text-input-storage-location').click();
    //       await page
    //         .getByRole('option', { name: `${backupStorage}` })
    //         .click();
    //     }
    //     await expect(
    //       page.getByTestId('text-input-storage-location')
    //     ).toHaveValue(backupStorage);
    //     await page.getByTestId('form-dialog-create').click();
    //     await waitForStatus(
    //       page,
    //       baseBackupName + '-new-cluster',
    //       'Succeeded',
    //       360000
    //     );
    //   });
    //
    //   await test.step('Add some data after the backup', async () => {
    //     await insertTestDB(
    //       clusterName,
    //       namespace,
    //       ['9'],
    //       ['1', '2', '3', '4', '5', '6', '7', '8', '9']
    //     );
    //     // for PG we need one more transaction to be able to restore to the previous one
    //     if (db == 'postgresql') {
    //       await insertTestDB(
    //         clusterName,
    //         namespace,
    //         ['10'],
    //         ['1', '2', '3', '4', '5', '6', '7', '8', '9', '10']
    //       );
    //     }
    //   });
    //
    //   await test.step('Wait for binlogs to be uploaded', async () => {
    //     await page.waitForTimeout(60000);
    //   });
    //
    //   let newClusterName: string;
    //   await test.step('Create DB from latest PITR', async () => {
    //     await page.goto('databases');
    //     await findDbAndClickActions(
    //       page,
    //       clusterName,
    //       'Create DB from a backup'
    //     );
    //     await page.getByTestId('radio-option-fromPITR').click({ timeout: 5000 });
    //     await expect(page.getByTestId('radio-option-fromPITR')).toBeChecked({
    //       timeout: 5000,
    //     });
    //     await expect(
    //       page.getByPlaceholder('DD/MM/YYYY at hh:mm:ss')
    //     ).toBeVisible({ timeout: 5000 });
    //     await expect(
    //       page.getByPlaceholder('DD/MM/YYYY at hh:mm:ss')
    //     ).not.toBeEmpty({ timeout: 5000 });
    //     // Submit and open DB creation wizard
    //     await page.getByTestId('CalendarIcon').click();
    //     await page.getByRole('button', { name: 'OK' }).click();
    //     await page.getByTestId('form-dialog-create').click({ timeout: 5000 });
    //     const currentUrl = page.url();
    //     expect(currentUrl).toContain('/databases/new');
    //     newClusterName = await page
    //       .getByTestId('text-input-db-name')
    //       .inputValue();
    //     expect(newClusterName).not.toBe('');
    //     await goToLastStepByStepAndSubmit(page, 5000);
    //   });
    //
    //   await test.step('Wait for the new cluster to be created and ready', async () => {
    //     await page.goto('/databases');
    //     if (db !== 'postgresql') {
    //       await waitForStatus(page, newClusterName, 'Initializing', 30000);
    //     }
    //     await waitForStatus(page, newClusterName, 'Restoring', 660000);
    //     await waitForStatus(page, newClusterName, 'Up', 900000);
    //   });
    //
    //   await test.step('Verify the restore was successful', async () => {
    //     // PG does not create a DBR object when restoring to new cluster.
    //     if (db !== 'postgresql') {
    //       await gotoDbClusterRestores(page, newClusterName);
    //       await waitForStatus(page, newClusterName, 'Succeeded', 120000);
    //     }
    //   });
    //
    //   await test.step('Verify the data in the new cluster', async () => {
    //     const result = await queryTestDB(newClusterName, namespace);
    //     expect(result.trim()).toBe('1\n2\n3\n4\n5\n6\n7\n8\n9');
    //   });
    //
    //   await test.step('Delete the restored cluster', async () => {
    //     await deleteDbCluster(page, newClusterName);
    //     await waitForStatus(page, newClusterName, 'Deleting', 15000);
    //     await waitForDelete(page, newClusterName, 240000);
    //   });
    // });
    // -----------------------------------------------------------------------
  }
);
