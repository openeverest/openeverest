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
import { Authorizer } from 'casbin.js';
import type { HttpApi } from '@generated/api-types';

let authorizer: Authorizer | null = null;
let username: string = '';
let policies: RBACPolicies = {
  enabled: false,
  m: '',
  p: [],
};
let timeoutId: ReturnType<typeof setTimeout>;

// We use the observer pattern to notify the authorizer and policies to components/hooks/etc that might need to react on changes
const observers: Array<
  (authorizer: Authorizer, policies: RBACPolicies) => void
> = [];
export const AuthorizerObservable = Object.freeze({
  notify: (authorizer: Authorizer, policies: RBACPolicies) =>
    observers.forEach((observer) => observer(authorizer, policies)),
  subscribe: (func: () => void) => observers.push(func),
  unsubscribe: (func: () => void) => {
    [...observers].forEach((observer, index) => {
      if (observer === func) {
        observers.splice(index, 1);
      }
    });
  },
});

export type RBACStandardAction = 'read' | 'update' | 'delete' | 'create';
export type RBACAction = RBACStandardAction | 'read-connection';
export type RBACResource =
  | 'namespaces'
  | 'database-engines'
  | 'database-clusters'
  | 'backups'
  | 'backup-classes'
  | 'instances'
  | 'backup-storages'
  | 'monitoring-configs'
  | 'monitoring-classes'
  | 'monitoring-destinations'
  | 'monitoring-bindings'
  | 'secrets'
  | 'pod-scheduling-policies'
  | 'load-balancer-configs'
  | 'enginefeatures/split-horizon-dns-configs'
  | 'providers'
  | 'plugins'
  | 'restores'
  | 'backup-imports'
  | 'config-maps';

type FirstSegment<P extends string> = P extends `${infer Head}/${string}`
  ? Head
  : P;
type ApiResourceUnder<
  Prefix extends string,
  Path = keyof HttpApi.paths,
> = Path extends `${Prefix}/${infer Rest}` ? FirstSegment<Rest> : never;

type RBACObjectFormat<R extends RBACResource> =
  R extends ApiResourceUnder<'/clusters/{cluster}/namespaces/{namespace}'>
    ? 'cluster-namespaced'
    : R extends ApiResourceUnder<'/clusters/{cluster}'>
      ? 'cluster-scoped'
      : 'as-is';

// The scope follows the resource's API path, so a misclassified or missing
// resource is a type error.
const RBAC_OBJECT_FORMATS: { [R in RBACResource]: RBACObjectFormat<R> } = {
  namespaces: 'cluster-scoped',
  providers: 'cluster-scoped',
  plugins: 'cluster-scoped',
  'backup-classes': 'cluster-scoped',
  'monitoring-classes': 'cluster-scoped',
  instances: 'cluster-namespaced',
  backups: 'cluster-namespaced',
  'backup-imports': 'cluster-namespaced',
  restores: 'cluster-namespaced',
  'backup-storages': 'cluster-namespaced',
  'monitoring-configs': 'cluster-namespaced',
  'monitoring-destinations': 'cluster-namespaced',
  'monitoring-bindings': 'cluster-namespaced',
  'config-maps': 'cluster-namespaced',
  secrets: 'cluster-namespaced',
  'database-engines': 'as-is',
  'database-clusters': 'as-is',
  'pod-scheduling-policies': 'as-is',
  'load-balancer-configs': 'as-is',
  'enginefeatures/split-horizon-dns-configs': 'as-is',
};

// Matches ClusterNamespacedObjectName / ClusterObjectName in pkg/rbac/rbac.go
export const toRBACObject = (
  resource: RBACResource,
  object: string,
  cluster: string
): string => {
  switch (RBAC_OBJECT_FORMATS[resource]) {
    case 'cluster-namespaced':
      return object === '*' ? `${cluster}/*/*` : `${cluster}/${object}`;
    case 'cluster-scoped':
      return `${cluster}/${object}`;
    default:
      return object;
  }
};

const constructAuthorizer = async () => {
  const newAuthorizer = new Authorizer('auto', { endpoint: '/' });
  newAuthorizer.user = username;
  await newAuthorizer.initEnforcer(JSON.stringify(policies));

  authorizer = newAuthorizer;
  AuthorizerObservable.notify(authorizer, policies);
  return authorizer;
};

const assignPolicies = async () => {
  const newPolicies = await getRBACPolicies();

  if (
    newPolicies.enabled !== policies.enabled ||
    newPolicies.p.length !== policies.p.length ||
    JSON.stringify(newPolicies.p) !== JSON.stringify(policies.p)
  ) {
    policies = newPolicies;
    constructAuthorizer();
  }
};

export const getAuthorizer = async () => {
  if (!authorizer) {
    return constructAuthorizer();
  }

  return authorizer;
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
    ? (await getAuthorizer()).can(specificResource, action, resource)
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
