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

import { execFileSync } from 'child_process';
import { APIRequestContext, expect } from '@playwright/test';
import { HttpApi } from '@generated/api-types';
import { EVEREST_CI_CLUSTER, TIMEOUTS } from '@e2e/constants';
import { getCITokenFromLocalStorage } from '@e2e/utils/localStorage';

type Schemas = HttpApi.components['schemas'];
type Instance = Schemas['Instance'];

// The PMM 3 server `make deploy-monitoring` installs. It is only reachable in
// the cluster, so its API is called from inside its pod.
const PMM_POD = ['-n', 'default', 'pmm-0', '-c', 'pmm'];
export const PMM_SERVER_URL =
  'https://monitoring-service.default.svc.cluster.local';

const callPmmApi = <T>(path: string, body?: object): T => {
  const user = `${process.env.MONITORING_USER || 'admin'}:${process.env.MONITORING_PASSWORD || 'admin'}`;
  const post = body
    ? ['-X', 'POST', '-H', 'Content-Type: application/json']
    : [];
  const data = body ? ['-d', JSON.stringify(body)] : [];
  const output = execFileSync('kubectl', [
    'exec',
    ...PMM_POD,
    '--',
    'curl',
    '-sfk',
    '-u',
    user,
    ...post,
    ...data,
    `https://localhost:8443${path}`,
  ]);
  return JSON.parse(output.toString());
};

// A token of a new PMM service account with the Admin role, which a PMM
// destination needs. PMM refuses a service account name that is taken.
export const createPmmServiceAccountToken = () => {
  const name = `openeverest-e2e-${Date.now()}`;
  const { id } = callPmmApi<{ id: number }>('/graph/api/serviceaccounts', {
    name,
    role: 'Admin',
  });
  return callPmmApi<{ key: string }>(
    `/graph/api/serviceaccounts/${id}/tokens`,
    { name }
  ).key;
};

// The MySQL services pmm-client registered with the PMM server.
export const getPmmMySqlServices = () =>
  callPmmApi<{ mysql?: { service_name: string; cluster: string }[] }>(
    '/v1/inventory/services'
  ).mysql ?? [];

const namespacePath = (namespace: string) =>
  `/v1/clusters/${EVEREST_CI_CLUSTER}/namespaces/${namespace}`;

export const destinationPath = (namespace: string, name: string) =>
  `${namespacePath(namespace)}/monitoring-destinations/${name}`;

export const instancePath = (namespace: string, name: string) =>
  `${namespacePath(namespace)}/instances/${name}`;

export const secretPath = (namespace: string, name: string) =>
  `${namespacePath(namespace)}/secrets/${name}`;

// The suites outlive a single token; this returns a cached, fresh one.
const authHeaders = async () => ({
  Authorization: `Bearer ${await getCITokenFromLocalStorage()}`,
});

export const getStatusCode = async (request: APIRequestContext, path: string) =>
  (await request.get(path, { headers: await authHeaders() })).status();

const getResource = async <T>(
  request: APIRequestContext,
  path: string
): Promise<T> => {
  const response = await request.get(path, { headers: await authHeaders() });
  expect(response.status(), await response.text()).toBe(200);
  return response.json();
};

export const getDestination = (
  request: APIRequestContext,
  namespace: string,
  name: string
) =>
  getResource<Schemas['MonitoringDestination']>(
    request,
    destinationPath(namespace, name)
  );

export const getInstance = (
  request: APIRequestContext,
  namespace: string,
  name: string
) => getResource<Instance>(request, instancePath(namespace, name));

export const getSecret = (
  request: APIRequestContext,
  namespace: string,
  name: string
) => getResource<Schemas['Secret']>(request, secretPath(namespace, name));

// A status write between the read and the write makes the API answer 409, so
// the change is retried on the latest version.
export const updateInstance = async (
  request: APIRequestContext,
  namespace: string,
  name: string,
  change: (instance: Instance) => void
) => {
  await expect(async () => {
    const instance = await getInstance(request, namespace, name);
    change(instance);
    const response = await request.put(instancePath(namespace, name), {
      headers: await authHeaders(),
      data: instance,
    });
    expect(response.ok(), await response.text()).toBe(true);
  }).toPass({ intervals: [1000, 2000, 5000], timeout: TIMEOUTS.OneMinute });
};

// Deletes the resources one after the other, each once the previous one is
// gone: an instance before the destinations it sends to, and those before
// their credentials Secrets, since the API refuses to delete what is in use.
// The delete is repeated while waiting: the API may still refuse it for a
// moment after the user is gone. Best effort: it never fails the caller, so
// it fits set-up and teardown alike.
export const deleteInOrder = async (
  request: APIRequestContext,
  paths: string[]
) => {
  for (const path of paths) {
    try {
      await expect
        .poll(
          async () => {
            await request.delete(path, { headers: await authHeaders() });
            return getStatusCode(request, path);
          },
          { timeout: TIMEOUTS.FiveMinutes, intervals: [2000, 5000, 10000] }
        )
        .toBe(404);
    } catch {
      // Left for the next run's set-up to remove.
    }
  }
};
