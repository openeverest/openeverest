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

import {test, expect} from '@fixtures'
import {EVEREST_CI_NAMESPACE, TIMEOUTS} from '@root/constants';
import {checkError} from '@tests/utils/api';

const PRESET_NAME = 'test-preset';
const INSTANCE_NAME = 'test-instance-patch';
const CLUSTER_NAME = 'main';
const MERGE_PATCH = 'application/merge-patch+json';

const instanceURL = `/v1/clusters/${CLUSTER_NAME}/namespaces/${EVEREST_CI_NAMESPACE}/instances/${INSTANCE_NAME}`;

test.describe('Instance PATCH', () => {
  // Serial, not parallel: the suite runs fullyParallel with 5 workers, so a
  // parallel describe would run beforeAll once per worker and race five
  // creates of the same instance. These tests also share one instance's
  // replicas, so they have to run in order.
  test.describe.configure({mode: 'serial', timeout: TIMEOUTS.OneMinute});

  test.beforeAll(async ({request}) => {
    const resolveResponse = await request.get(
      `/v1/clusters/${CLUSTER_NAME}/instance-presets/${PRESET_NAME}/resolve?namespace=${EVEREST_CI_NAMESPACE}`
    );
    await checkError(resolveResponse);
    const preset = await resolveResponse.json();

    const response = await request.post(
      `/v1/clusters/${CLUSTER_NAME}/namespaces/${EVEREST_CI_NAMESPACE}/instances`,
      {data: {metadata: {name: INSTANCE_NAME}, spec: preset.spec}}
    );
    await checkError(response);
  });

  test.afterAll(async ({request}) => {
    await request.delete(instanceURL);
  });

  test('applies a merge patch and leaves the rest alone', async ({request}) => {
    const before = await request.get(instanceURL);
    await checkError(before);
    const original = await before.json();

    const response = await request.patch(instanceURL, {
      headers: {'Content-Type': MERGE_PATCH},
      data: {spec: {components: {engine: {replicas: 3}}}},
    });

    await checkError(response);
    const patched = await response.json();
    expect(patched.spec.components.engine.replicas).toBe(3);
    // Members the patch did not name keep their stored value.
    expect(patched.spec.version).toBe(original.spec.version);
    expect(patched.spec.providerRef.name).toBe(original.spec.providerRef.name);
  });

  test('a misspelt path is rejected rather than silently ignored', async ({request}) => {
    const response = await request.patch(instanceURL, {
      headers: {'Content-Type': MERGE_PATCH},
      data: {spec: {replicaz: 3}},
    });

    // Without fieldValidation=Strict the API server prunes the unknown member
    // and answers 200 having changed nothing.
    expect(response.status()).toBe(422);
    expect(await response.text()).toContain('spec.replicaz');

    const after = await request.get(instanceURL);
    await checkError(after);
    expect((await after.json()).spec.replicaz).toBeUndefined();
  });

  test('an undeclared Content-Type is rejected with 415', async ({request}) => {
    const response = await request.patch(instanceURL, {
      headers: {'Content-Type': 'application/json'},
      data: {spec: {components: {engine: {replicas: 4}}}},
    });

    // The 415 mapping matches on a kin-openapi message that library keeps
    // unexported, so nothing fails at compile time if they reword it.
    expect(response.status()).toBe(415);
  });

  test('a stale resourceVersion fails the precondition with 409', async ({request}) => {
    const before = await request.get(instanceURL);
    await checkError(before);
    const staleVersion = (await before.json()).metadata.resourceVersion;

    const first = await request.patch(instanceURL, {
      headers: {'Content-Type': MERGE_PATCH},
      data: {metadata: {resourceVersion: staleVersion}, spec: {components: {engine: {replicas: 5}}}},
    });
    await checkError(first);

    const second = await request.patch(instanceURL, {
      headers: {'Content-Type': MERGE_PATCH},
      data: {metadata: {resourceVersion: staleVersion}, spec: {components: {engine: {replicas: 6}}}},
    });

    expect(second.status()).toBe(409);

    const after = await request.get(instanceURL);
    await checkError(after);
    expect((await after.json()).spec.components.engine.replicas).toBe(5);
  });

  test('a member the caller may not set is rejected', async ({request}) => {
    for (const patch of [
      {status: {}},
      {metadata: {finalizers: []}},
      {metadata: {annotations: null}},
      {metadata: {annotations: {'openeverest.io/last-actor-id': 'someone-else'}}},
    ]) {
      const response = await request.patch(instanceURL, {
        headers: {'Content-Type': MERGE_PATCH},
        data: patch,
      });
      expect(response.status(), `patch ${JSON.stringify(patch)}`).toBe(400);
    }
  });
});
