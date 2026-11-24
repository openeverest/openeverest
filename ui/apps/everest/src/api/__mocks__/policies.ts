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

import rbacModel from '/src/static/model.conf?raw';

export const getRBACPolicies = vi.fn();

const applyPolicies = async (next: {
  enabled: boolean;
  m: string;
  p: string[][];
}) => {
  getRBACPolicies.mockResolvedValue(next);
  // Imported lazily: utils/rbac imports this module, so a static import would be circular.
  const { AuthorizerObservable, getPolicies, initializeAuthorizerFetchLoop } =
    await import('utils/rbac');
  const previousPolicies = getPolicies();
  let onRebuilt = () => {};
  // The authorizer is rebuilt without being awaited; wait so checks never see the previous policy.
  const rebuilt = new Promise<void>((resolve) => {
    onRebuilt = resolve;
  });
  AuthorizerObservable.subscribe(onRebuilt);
  await initializeAuthorizerFetchLoop('alice');
  if (getPolicies() !== previousPolicies) {
    await rebuilt;
  }
  AuthorizerObservable.unsubscribe(onRebuilt);
};

// Loads v2 policy rules (resource, action, object) into the real Casbin authorizer.
export const grantRBAC = async (...rules: string[][]) => {
  await applyPolicies({
    enabled: true,
    m: rbacModel,
    p: rules.map((rule) => ['p', 'alice', ...rule]),
  });
};

// Restores the default disabled-RBAC policy so later tests are not stuck as alice.
export const resetRBAC = async () => {
  await applyPolicies({
    enabled: false,
    m: '',
    p: [],
  });
};
