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

const OLD_RBAC_FILE = 'old_rbac_permissions';

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

export const setRBACPermissionsK8S = async (
  permissions: [string, string, string][] = []
) => {
  const command = `kubectl patch configmap/everest-rbac --namespace everest-system --type merge -p '{"data":{"enabled": "${permissions !== undefined}", "policy.csv":"g,${process.env.RBAC_USER},role:e2e-rbac-user\\n${permissions.map((p) => `p,role:e2e-rbac-user,${p.join(',')}`).join('\\n')}"}}'`;
  execSync(command);

  // We need this to give time for the RBAC to be applied, or headless tests might fail for being too fast
  return new Promise<void>((resolve) =>
    setTimeout(
      () => {
        resolve();
      },
      process.env.CI ? 3000 : 500
    )
  );
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

const patchRBAC = (enabled: boolean, policy: string[]) =>
  execSync(
    `kubectl patch configmap/everest-rbac --namespace everest-system --type merge -p '${JSON.stringify(
      { data: { enabled: String(enabled), 'policy.csv': policy.join('\n') } }
    )}'`
  );

const ADMIN_POLICY = `g,${process.env.CI_USER},role:admin`;
const RBAC_APPLY_DELAY_MS = process.env.CI ? 3000 : 1000;

// Grants the RBAC user exactly the permissions. Unlike
// setRBACPermissionsK8S, the admin stays an admin, so tests running as the
// admin at the same time are not affected.
export const setRBACUserPermissions = async (
  permissions: [string, string, string][]
) => {
  patchRBAC(true, [
    ADMIN_POLICY,
    `g,${process.env.RBAC_USER},role:e2e-rbac-user`,
    ...permissions.map((p) => `p,role:e2e-rbac-user,${p.join(',')}`),
  ]);
  await new Promise((resolve) => setTimeout(resolve, RBAC_APPLY_DELAY_MS));
};

export const disableRBAC = async () => {
  patchRBAC(false, [ADMIN_POLICY]);
  await new Promise((resolve) => setTimeout(resolve, RBAC_APPLY_DELAY_MS));
};
