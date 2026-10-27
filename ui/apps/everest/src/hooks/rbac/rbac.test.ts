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

import { renderHook, waitFor } from '@testing-library/react';
import rbacModel from '/src/static/model.conf?raw';
import {
  initializeAuthorizerFetchLoop,
  stopAuthorizerFetchLoop,
} from 'utils/rbac';
import { useNamespacePermissionsForResource, useRBACPermissions } from './rbac';

const mockGetRBACPolicies = vi.fn();

vi.mock('api/policies', () => ({
  getRBACPolicies: () => mockGetRBACPolicies(),
}));

vi.mock('hooks/api/namespaces', () => {
  const namespaces = { data: ['ns1', 'ns2'] };
  return { useNamespaces: () => namespaces };
});

const grant = async (...rules: string[][]) => {
  mockGetRBACPolicies.mockResolvedValue({
    enabled: true,
    m: rbacModel,
    p: rules.map((rule) => ['p', 'alice', ...rule]),
  });
  await initializeAuthorizerFetchLoop('alice');
};

describe('RBAC hooks with v2 policies', () => {
  afterEach(() => {
    stopAuthorizerFetchLoop();
  });

  it('lists only the namespaces a namespaced grant covers', async () => {
    await grant(['instances', 'create', 'main/ns1/*']);

    const { result } = renderHook(() =>
      useNamespacePermissionsForResource('instances')
    );

    await waitFor(() => expect(result.current.canCreate).toEqual(['ns1']));
  });

  it('matches a cluster-scoped grant', async () => {
    await grant(['backup-classes', 'read', 'main/*']);

    const { result } = renderHook(() => useRBACPermissions('backup-classes'));

    await waitFor(() => expect(result.current.canRead).toBe(true));
  });

  it('matches a cluster-wide grant when checking every namespace', async () => {
    await grant(['instances', 'read', 'main/*/*']);

    const { result } = renderHook(() => useRBACPermissions('instances'));

    await waitFor(() => expect(result.current.canRead).toBe(true));
  });
});
