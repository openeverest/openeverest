import { getRBACPolicies, RBACPolicies } from 'api/policies';
import { Authorizer } from 'casbin.js';

let authorizer: Authorizer | null = null;
let username: string = '';
let policies: RBACPolicies = {
  enabled: false,
  m: '',
  p: [],
};
let timeoutId: NodeJS.Timeout;

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
