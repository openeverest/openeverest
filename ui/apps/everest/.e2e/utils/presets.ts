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
import { APIRequestContext, expect } from '@playwright/test';
import { EVEREST_CI_CLUSTER } from '@e2e/constants';

// The spec of an Instance: a preset carries it as is.
export type PresetSpec = Record<string, unknown>;

const API_VERSION = 'core.openeverest.io/v1alpha1';

const instancePresetsUrl = (namespace: string) =>
  `/v1/clusters/${EVEREST_CI_CLUSTER}/namespaces/${namespace}/instance-presets`;

const authHeaders = (token: string) => ({
  Authorization: `Bearer ${token}`,
});

export const deleteInstancePreset = async (
  request: APIRequestContext,
  token: string,
  namespace: string,
  name: string
) => {
  const response = await request.delete(
    `${instancePresetsUrl(namespace)}/${name}`,
    { headers: authHeaders(token) }
  );
  expect([204, 404], await response.text()).toContain(response.status());
};

export const createInstancePreset = async (
  request: APIRequestContext,
  token: string,
  namespace: string,
  name: string,
  spec: PresetSpec
) => {
  // A leftover of an interrupted run would answer 409.
  await deleteInstancePreset(request, token, namespace, name);
  const response = await request.post(instancePresetsUrl(namespace), {
    headers: authHeaders(token),
    data: { metadata: { name }, spec },
  });
  expect(response.status(), await response.text()).toBe(201);
};

// The API only lists cluster-wide presets: cluster administrators manage them
// with kubectl, so the tests do the same.
export const applyClusterInstancePreset = (name: string, spec: PresetSpec) => {
  const manifest = {
    apiVersion: API_VERSION,
    kind: 'ClusterInstancePreset',
    metadata: { name },
    spec,
  };
  execSync('kubectl apply -f -', {
    input: JSON.stringify(manifest),
    stdio: ['pipe', 'pipe', 'pipe'],
  });
};

export const deleteClusterInstancePreset = (name: string) => {
  execSync(
    `kubectl delete clusterinstancepreset ${name} --ignore-not-found --wait=false`,
    { stdio: 'pipe' }
  );
};
