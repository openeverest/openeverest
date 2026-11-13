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
  APIRequestContext,
  expect,
  request as apiRequest,
} from '@playwright/test';
import { HttpApi } from '@generated/api-types';
import { EVEREST_CI_CLUSTER, TIMEOUTS } from '@e2e/constants';
import { getCITokenFromLocalStorage } from '@e2e/utils/localStorage';

export type Instance = HttpApi.components['schemas']['Instance'];

export const instancePath = (namespace: string, name: string) =>
  `/v1/clusters/${EVEREST_CI_CLUSTER}/namespaces/${namespace}/instances/${name}`;

const authHeaders = async () => ({
  Authorization: `Bearer ${await getCITokenFromLocalStorage()}`,
});

export const getInstance = async (
  request: APIRequestContext,
  namespace: string,
  name: string
): Promise<Instance> => {
  const response = await request.get(instancePath(namespace, name), {
    headers: await authHeaders(),
  });
  expect(response.ok()).toBeTruthy();
  return response.json();
};

export const deleteInstanceAndWait = async (
  request: APIRequestContext,
  namespace: string,
  name: string
) => {
  const response = await request.delete(instancePath(namespace, name), {
    headers: await authHeaders(),
  });
  expect([204, 404]).toContain(response.status());
  await expect
    .poll(
      async () =>
        (
          await request.get(instancePath(namespace, name), {
            headers: await authHeaders(),
          })
        ).status(),
      { timeout: TIMEOUTS.FiveMinutes, intervals: [2000, 5000] }
    )
    .toBe(404);
};

// Best-effort cleanup for afterAll hooks, which have no `request` fixture.
export const deleteInstanceQuietly = async (
  baseURL: string | undefined,
  namespace: string,
  name: string
) => {
  try {
    const context = await apiRequest.newContext({ baseURL });
    await context.delete(instancePath(namespace, name), {
      headers: await authHeaders(),
    });
    await context.dispose();
  } catch {
    // Leftovers are removed by the next run's pre-create delete.
  }
};
