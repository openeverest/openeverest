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

import { execSync } from 'child_process';
import { APIRequestContext, Page } from '@playwright/test';
import { expect, test } from '@e2e/fixtures/auth';
import { EVEREST_CI_NAMESPACES, TIMEOUTS } from '@e2e/constants';
import { deleteDbCluster } from '@e2e/utils/db-clusters-list';
import { getCITokenFromLocalStorage } from '@e2e/utils/localStorage';
import { findRowAndClickActions, waitForDelete } from '@e2e/utils/table';

// Percona Server for MongoDB is the first provider that reports the facts the
// status is derived from: availability, named operations and problems.
const PROVIDER_NAME = 'percona-server-mongodb';
const FROM_VERSION = '8.0.8';
const TO_VERSION = '8.0.12';

const namespace = EVEREST_CI_NAMESPACES.EVEREST_UI;
const dbName = 'status-golden';
// Asks for more memory than any node has, so its pods are never scheduled.
const stuckDbName = 'status-stuck';

const instancesPath = `/v1/clusters/main/namespaces/${namespace}/instances`;

const authHeaders = async () => ({
  Authorization: `Bearer ${await getCITokenFromLocalStorage()}`,
});

const instanceSpec = (replicas: number, memory: string) => ({
  providerRef: { name: PROVIDER_NAME },
  version: FROM_VERSION,
  components: {
    engine: {
      type: 'mongod',
      replicas,
      resources: {
        requests: { cpu: '200m', memory },
        limits: { cpu: '1', memory },
      },
      storage: { size: '2Gi' },
    },
  },
});

const createInstance = async (
  request: APIRequestContext,
  name: string,
  spec: ReturnType<typeof instanceSpec>
) => {
  const response = await request.post(instancesPath, {
    headers: await authHeaders(),
    data: {
      apiVersion: 'core.openeverest.io/v1alpha1',
      kind: 'Instance',
      metadata: { name },
      spec,
    },
  });
  expect(response.ok(), await response.text()).toBeTruthy();
};

const deleteInstance = async (request: APIRequestContext, name: string) => {
  const response = await request.delete(`${instancesPath}/${name}`, {
    headers: await authHeaders(),
  });
  expect([200, 202, 204, 404]).toContain(response.status());
};

// The API replaces the Instance, so the change is made on what is stored.
const changeVersion = async (
  request: APIRequestContext,
  name: string,
  version: string
) => {
  await expect(async () => {
    const headers = await authHeaders();
    const current = await request.get(`${instancesPath}/${name}`, { headers });
    expect(current.ok()).toBeTruthy();
    const instance = await current.json();

    const updated = await request.put(`${instancesPath}/${name}`, {
      headers,
      data: { ...instance, spec: { ...instance.spec, version } },
    });
    // A write in between makes the stored version newer: read it again.
    expect(updated.ok(), await updated.text()).toBeTruthy();
  }).toPass({ timeout: TIMEOUTS.ThirtySeconds });
};

// Pauses or resumes the Instance on the operator's own object, which is how
// it gets paused behind OpenEverest's back.
const setPausedOutsideOpenEverest = (name: string, paused: boolean) => {
  execSync(
    `kubectl patch psmdb ${name} --namespace ${namespace} --type merge -p '{"spec":{"pause":${paused}}}'`
  );
};

// An Instance is otherwise gone within seconds of the request. The finalizer
// keeps it in deletion for as long as the test looks at it.
const DELETION_HOLD = 'e2e.openeverest.io/hold-deletion';

const setDeletionHold = (name: string, hold: boolean) => {
  const stored = execSync(
    `kubectl get instance ${name} --namespace ${namespace} -o jsonpath='{.metadata.finalizers}'`
  ).toString();
  const others: string[] = (stored ? JSON.parse(stored) : []).filter(
    (finalizer: string) => finalizer !== DELETION_HOLD
  );
  const finalizers = hold ? [...others, DELETION_HOLD] : others;

  execSync(
    `kubectl patch instance ${name} --namespace ${namespace} --type merge -p '${JSON.stringify(
      { metadata: { finalizers } }
    )}'`
  );
};

const leftoverVolumes = () =>
  execSync(
    `kubectl get pvc --namespace ${namespace} -o name | grep -E '${dbName}|${stuckDbName}' || true`
  )
    .toString()
    .trim();

const statusOf = (page: Page, name: string) =>
  page.getByTestId(`${name}-status`);

const openDetails = async (page: Page, name: string) => {
  await page.goto(`/databases/${namespace}/${name}/overview`);
  await expect(page.getByTestId('cluster-overview')).toBeVisible({
    timeout: TIMEOUTS.ThirtySeconds,
  });
};

test.describe.configure({ retries: 0 });

test.describe.serial(
  'Instance status golden (PSMDB)',
  {
    tag: '@release',
  },
  () => {
    test.describe.configure({ timeout: TIMEOUTS.ThirtyMinutes });

    test.beforeAll(async ({ request }) => {
      const providers = await request.get('/v1/clusters/main/providers', {
        headers: await authHeaders(),
      });
      const { items = [] } = await providers.json();
      const provider = items.find(
        (item: { metadata?: { name?: string } }) =>
          item.metadata?.name === PROVIDER_NAME
      );
      const versions: string[] = (provider?.spec?.versions ?? []).map(
        (version: { name: string }) => version.name
      );
      test.skip(
        ![FROM_VERSION, TO_VERSION].every((version) =>
          versions.includes(version)
        ),
        `${PROVIDER_NAME} with versions ${FROM_VERSION} and ${TO_VERSION} is not installed`
      );

      // Leftovers of a run that was interrupted.
      await deleteInstance(request, dbName);
      await deleteInstance(request, stuckDbName);
      await expect
        .poll(
          async () => {
            const list = await request.get(instancesPath, {
              headers: await authHeaders(),
            });
            const { items: instances = [] } = await list.json();
            return instances.filter(
              (item: { metadata?: { name?: string } }) =>
                item.metadata?.name === dbName ||
                item.metadata?.name === stuckDbName
            ).length;
          },
          { timeout: TIMEOUTS.FiveMinutes, intervals: [5000] }
        )
        .toBe(0);
      // The volumes outlive the Instance by a moment, and an Instance created
      // under the same name would start from their data.
      await expect
        .poll(() => leftoverVolumes(), {
          timeout: TIMEOUTS.FiveMinutes,
          intervals: [5000],
        })
        .toBe('');
    });

    // Runs even when a step failed, which serial mode would otherwise skip.
    test.afterAll(async ({ request }) => {
      try {
        await deleteInstance(request, dbName);
        await deleteInstance(request, stuckDbName);
        setDeletionHold(dbName, false);
      } catch {
        // Teardown must never fail the suite; the next run deletes leftovers.
      }
    });

    test('A new Instance reads Creating, then Ready', async ({
      page,
      request,
    }) => {
      await createInstance(request, dbName, instanceSpec(3, '1G'));
      await createInstance(request, stuckDbName, instanceSpec(1, '512G'));

      await page.goto('/databases');

      await test.step('It is being created, and is not called unavailable', async () => {
        await expect(statusOf(page, dbName)).toHaveText('Creating', {
          timeout: TIMEOUTS.ThirtySeconds,
        });
      });

      await test.step('It becomes ready', async () => {
        await expect(statusOf(page, dbName)).toHaveText('Ready', {
          timeout: TIMEOUTS.FifteenMinutes,
        });
      });

      await test.step('Ready: no alert, connection details, editing offered', async () => {
        await openDetails(page, dbName);
        await expect(statusOf(page, dbName)).toHaveText('Ready');
        await expect(page.getByTestId('instance-status-alert')).toHaveCount(0);
        await expect(
          page.getByTestId('connection-details-overview-section')
        ).toContainText('Username');
        await expect(page.getByTestId('resources-edit-button')).toBeVisible();
      });
    });

    test('Pods that cannot be scheduled are a warning that may still clear', async ({
      page,
    }) => {
      await page.goto('/databases');
      await expect(statusOf(page, stuckDbName)).toHaveText(
        "Pods can't be scheduled",
        { timeout: TIMEOUTS.FiveMinutes }
      );

      await openDetails(page, stuckDbName);
      const alert = page.getByTestId('instance-status-alert');

      await test.step('The alert names the problem, gives advice and quotes the cause', async () => {
        await expect(alert).toContainText("Pods can't be scheduled");
        await expect(alert).toContainText('Add nodes, free up CPU or memory');
        await expect(alert).toContainText('this may clear by itself');
        await expect(alert).toContainText('Insufficient memory');
      });

      await test.step('The Instance stays editable, which is how it gets fixed', async () => {
        await expect(page.getByTestId('resources-edit-button')).toBeVisible();
      });
    });

    test('An Instance paused outside OpenEverest is an error until it is resumed', async ({
      page,
    }) => {
      setPausedOutsideOpenEverest(dbName, true);
      await openDetails(page, dbName);
      const alert = page.getByTestId('instance-status-alert');

      await test.step('A reason the UI has no label for is still shown, as an error', async () => {
        await expect(statusOf(page, dbName)).toHaveText(
          /paused outside open ?everest/i,
          { timeout: TIMEOUTS.ThreeMinutes }
        );
        await expect(alert).toContainText('This will not clear by itself');
        await expect(alert).toContainText('spec.pause is set');
      });

      await test.step('It recovers once the cause is gone', async () => {
        setPausedOutsideOpenEverest(dbName, false);
        await expect(statusOf(page, dbName)).toHaveText('Ready', {
          timeout: TIMEOUTS.FiveMinutes,
        });
        await expect(alert).toHaveCount(0);
      });
    });

    test('An upgrade keeps the Instance available and holds edits back', async ({
      page,
      request,
    }) => {
      await changeVersion(request, dbName, TO_VERSION);
      await page.goto('/databases');

      await test.step('The list says both: upgrading, and available', async () => {
        await expect(statusOf(page, dbName)).toHaveText(
          'Upgrading \u00b7 available',
          { timeout: TIMEOUTS.ThreeMinutes }
        );
      });

      await test.step('So does the header, and nothing is reported as wrong', async () => {
        await openDetails(page, dbName);
        await expect(statusOf(page, dbName)).toHaveText(
          'Upgrading \u00b7 available',
          { timeout: TIMEOUTS.ThirtySeconds }
        );
        await expect(page.getByTestId('instance-status-alert')).toHaveCount(0);
      });

      await test.step('Connection details stay, editing waits for the upgrade', async () => {
        await expect(
          page.getByTestId('connection-details-overview-section')
        ).toContainText('Username');
        await expect(page.getByTestId('resources-edit-button')).toHaveCount(0);
      });

      await test.step('The upgrade finishes and editing is offered again', async () => {
        await expect(statusOf(page, dbName)).toHaveText('Ready', {
          timeout: TIMEOUTS.FifteenMinutes,
        });
        await expect(page.getByTestId('resources-edit-button')).toBeVisible();
      });
    });

    test('An Instance being deleted reads Deleting until it is gone', async ({
      page,
      request,
    }) => {
      await deleteInstance(request, stuckDbName);
      setDeletionHold(dbName, true);
      await deleteDbCluster(page, dbName);

      await test.step('It reads Deleting, and can no longer be acted on', async () => {
        await expect(statusOf(page, dbName)).toHaveText('Deleting', {
          timeout: TIMEOUTS.ThirtySeconds,
        });
        // The dialog closes on its own once the request is accepted.
        await expect(page.getByTestId('form-dialog-delete')).toHaveCount(0);
        await findRowAndClickActions(page, dbName);
        await expect(page.getByTestId(`${dbName}-delete`)).toBeDisabled();
        await page.keyboard.press('Escape');
      });

      await test.step('It leaves the list once the deletion goes through', async () => {
        setDeletionHold(dbName, false);
        await waitForDelete(page, dbName, TIMEOUTS.FiveMinutes);
      });
    });
  }
);
