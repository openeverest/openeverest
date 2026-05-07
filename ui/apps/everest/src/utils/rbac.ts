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

import { getRBACPolicies, RBACPolicies } from 'api/policies';
import { Enforcer, newEnforcer, newModelFromString } from 'casbin';

let enforcer: Enforcer | null = null;
let username: string = '';
let policies: RBACPolicies = {
  enabled: false,
  m: '',
  p: [],
};
let timeoutId: NodeJS.Timeout;

// We use the observer pattern to notify the authorizer and policies to components/hooks/etc that might need to react on changes
const observers: Array<(enforcer: Enforcer, policies: RBACPolicies) => void> =
  [];
export const AuthorizerObservable = Object.freeze({
  notify: (enforcer: Enforcer, policies: RBACPolicies) =>
    observers.forEach((observer) => observer(enforcer, policies)),
  subscribe: (func: () => void) => observers.push(func),
  unsubscribe: (func: () => void) => {
    [...observers].forEach((observer, index) => {
      if (observer === func) {
        observers.splice(index, 1);
      }
    });
  },
});

export type RBACAction = 'read' | 'update' | 'delete' | 'create';
export type RBACResource =
  | 'namespaces'
  | 'database-engines'
  | 'database-clusters'
  | 'database-cluster-backups'
  | 'database-cluster-restores'
  | 'database-cluster-credentials'
  | 'backup-storages'
  | 'monitoring-instances'
  | 'pod-scheduling-policies'
  | 'data-importers'
  | 'load-balancer-configs'
  | 'data-import-jobs'
  | 'enginefeatures/split-horizon-dns-configs';

const constructEnforcer = async () => {
  const model = newModelFromString(policies.m);
  const e = await newEnforcer(model);

  for (const policyEntry of policies.p) {
    const arr = policyEntry.map((v: string) => v.trim());
    const pType = arr.shift();
    if (pType === 'p') {
      await e.addPolicy(...arr);
    } else if (pType === 'g') {
      await e.addGroupingPolicy(...arr);
    }
  }

  enforcer = e;
  AuthorizerObservable.notify(enforcer, policies);
  return enforcer;
};

const assignPolicies = async () => {
  const newPolicies = await getRBACPolicies();

  if (
    newPolicies.enabled !== policies.enabled ||
    newPolicies.p.length !== policies.p.length ||
    JSON.stringify(newPolicies.p) !== JSON.stringify(policies.p)
  ) {
    policies = newPolicies;
    constructEnforcer();
  }
};

export const getEnforcer = async () => {
  if (!enforcer) {
    return constructEnforcer();
  }

  return enforcer;
};

export const getPolicies = () => policies;

export const initializeAuthorizerFetchLoop = async (user: string) => {
  username = user;
  clearInterval(timeoutId);
  await assignPolicies();

  timeoutId = setInterval(async () => {
    await assignPolicies();
  }, 5000);
};

export const stopAuthorizerFetchLoop = () => {
  clearInterval(timeoutId);
};

export const can = async (
  action: RBACAction,
  resource: RBACResource,
  specificResource: string
  // When policies are disabled, we allow all actions
  // Params are inverted because of the way our policies are defined: "sub, res, act, obj" instead of "sub, obj, act"
) =>
  policies.enabled
    ? (await getEnforcer()).enforce(
        username,
        specificResource,
        action,
        resource
      )
    : true;

export const cannot = async (
  action: RBACAction,
  resource: RBACResource,
  specificResource: string
) => !(await can(action, resource, specificResource));

export const canAll = async (
  action: RBACAction,
  resource: RBACResource,
  specificResource: string[]
) => {
  for (let i = 0; i < specificResource.length; ++i) {
    if (await cannot(action, resource, specificResource[i])) {
      return false;
    }
  }
  return true;
};
