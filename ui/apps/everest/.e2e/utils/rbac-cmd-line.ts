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
import { request } from '@playwright/test';
import { getTokenFromLocalStorage } from './localStorage';

const OLD_RBAC_FILE = 'old_rbac_permissions';

const BASE_URL = process.env.EVEREST_URL || 'http://localhost:8080';
const RBAC_APPLY_TIMEOUT_MS = process.env.CI ? 15000 : 5000;
const RBAC_APPLY_POLL_INTERVAL_MS = 200;

export const saveOldRBACPermissions = async () => {
  const command = `kubectl get configmap everest-rbac --namespace everest-system -o jsonpath="{.data}" > ${OLD_RBAC_FILE}`;
  execSync(command);
};

export const restoreOldRBACPermissions = async () => {
  const oldRbacFileContent = execSync(`cat ${OLD_RBAC_FILE}`).toString();
  const command = `kubectl patch configmap/everest-rbac --namespace everest-system --type merge -p '{"data":${oldRbacFileContent}}'`;
  execSync(command);

  return new Promise<void>((resolve) =>
    setTimeout(() => {
      resolve();
    }, 1000)
  );
};

// rbac.setup switches to the RBAC user, so its token (the same for the whole
// run) lives in the saved storage state — read it once.
let rbacToken: string | undefined;

// A permission from GET /v1/permissions is [subject, resource, action, object].
// Match on the trailing [resource, action, object]; the count must match too,
// so a shrinking policy isn't satisfied by the previous (larger) one while the
// server is still reloading.
const policyIsApplied = (
  returned: string[][],
  expected: [string, string, string][]
): boolean =>
  returned.length === expected.length &&
  expected.every(([resource, action, object]) =>
    returned.some((perm) => {
      const [r, a, o] = perm.slice(-3);
      return r === resource && a === action && o === object;
    })
  );

// Poll GET /v1/permissions until the freshly-patched policy is reflected by the
// server, instead of sleeping a fixed amount. The fixed sleep raced the
// server's ConfigMap reload and made RBAC tests flaky (stale permissions ->
// 404 on deep-linked pages).
const waitForRBACPolicyApplied = async (
  permissions: [string, string, string][]
) => {
  if (permissions.length === 0) {
    return;
  }
  rbacToken ??= await getTokenFromLocalStorage();
  const ctx = await request.newContext({ baseURL: BASE_URL });
  const deadline = Date.now() + RBAC_APPLY_TIMEOUT_MS;
  try {
    while (Date.now() < deadline) {
      const resp = await ctx.get('/v1/permissions', {
        headers: { Authorization: `Bearer ${rbacToken}` },
      });
      if (resp.ok()) {
        const body = await resp.json();
        if (
          body?.enabled &&
          policyIsApplied(body.permissions ?? [], permissions)
        ) {
          return;
        }
      }
      await new Promise((r) => setTimeout(r, RBAC_APPLY_POLL_INTERVAL_MS));
    }
    throw new Error(
      `RBAC policy was not reflected in /v1/permissions within ${RBAC_APPLY_TIMEOUT_MS}ms`
    );
  } finally {
    await ctx.dispose();
  }
};

export const setRBACPermissionsK8S = async (
  permissions: [string, string, string][] = []
) => {
  const command = `kubectl patch configmap/everest-rbac --namespace everest-system --type merge -p '{"data":{"enabled": "${permissions !== undefined}", "policy.csv":"g,${process.env.RBAC_USER},role:e2e-rbac-user\\n${permissions.map((p) => `p,role:e2e-rbac-user,${p.join(',')}`).join('\\n')}"}}'`;
  execSync(command);

  await waitForRBACPolicyApplied(permissions);
};

export const giveUserAdminPermissions = async () => {
  execSync(
    `kubectl patch configmap/everest-rbac --namespace everest-system --type merge -p '{"data":{"enabled": "true", "policy.csv":"g,${process.env.RBAC_USER},role:admin"}}'`
  );

  return new Promise<void>((resolve) =>
    setTimeout(() => {
      resolve();
    }, 5000)
  );
};
