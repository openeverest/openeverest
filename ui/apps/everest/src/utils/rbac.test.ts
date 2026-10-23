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

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import rbacModel from '/src/static/model.conf?raw';
import {
  toRBACObject,
  can,
  initializeAuthorizerFetchLoop,
  stopAuthorizerFetchLoop,
} from './rbac';

const mockGetRBACPolicies = vi.fn();

vi.mock('api/policies', () => ({
  getRBACPolicies: () => mockGetRBACPolicies(),
}));

describe('toRBACObject', () => {
  it('namespaced v2 -> cluster/namespace/name', () => {
    expect(toRBACObject('instances', 'ns1/*', 'main')).toBe('main/ns1/*');
    expect(toRBACObject('backups', 'ns1/db1', 'main')).toBe('main/ns1/db1');
    expect(toRBACObject('instances', '*', 'main')).toBe('*/*/*');
  });

  it('cluster-scoped v2 -> cluster/name', () => {
    expect(toRBACObject('backup-classes', '*', 'main')).toBe('main/*');
    expect(toRBACObject('namespaces', 'ns1', 'main')).toBe('main/ns1');
  });

  it('v1 unchanged', () => {
    expect(toRBACObject('database-clusters', 'ns1/*', 'main')).toBe('ns1/*');
  });
});

describe('Casbin Authorizer real policies', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    stopAuthorizerFetchLoop();
  });

  it("admin's */*/* allows instances create on main/ns1/*", async () => {
    mockGetRBACPolicies.mockResolvedValue({
      enabled: true,
      m: rbacModel,
      p: [['p', 'admin', '*', '*', '*/*/*']],
    });

    await initializeAuthorizerFetchLoop('admin');

    const allowed = await can('create', 'instances', 'main/ns1/*');
    expect(allowed).toBe(true);
  });

  it('p, role:dev, instances, *, main/ns1/* allows ns1 but denies ns2', async () => {
    mockGetRBACPolicies.mockResolvedValue({
      enabled: true,
      m: rbacModel,
      p: [['p', 'role:dev', 'instances', '*', 'main/ns1/*']],
    });

    await initializeAuthorizerFetchLoop('role:dev');

    const allowedNs1 = await can('create', 'instances', 'main/ns1/*');
    const allowedNs2 = await can('create', 'instances', 'main/ns2/*');

    expect(allowedNs1).toBe(true);
    expect(allowedNs2).toBe(false);
  });
});
