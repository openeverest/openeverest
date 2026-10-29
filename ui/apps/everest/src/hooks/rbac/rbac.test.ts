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
import { grantRBAC } from 'api/__mocks__/policies';
import { stopAuthorizerFetchLoop } from 'utils/rbac';
import {
  useNamespacePermissionsForResource,
  useRBACAction,
  useRBACPermissions,
} from './rbac';

vi.mock('api/policies');

vi.mock('hooks/api/namespaces', () => {
  const namespaces = { data: ['ns1', 'ns2'] };
  return { useNamespaces: () => namespaces };
});

describe('RBAC hooks with v2 policies', () => {
  afterEach(() => {
    stopAuthorizerFetchLoop();
  });

  it('lists only the namespaces a namespaced grant covers', async () => {
    await grantRBAC(['instances', 'create', 'main/ns1/*']);

    const { result } = renderHook(() =>
      useNamespacePermissionsForResource('instances')
    );

    await waitFor(() => expect(result.current.canCreate).toEqual(['ns1']));
  });

  it('matches a cluster-scoped grant', async () => {
    await grantRBAC(['backup-classes', 'read', 'main/*']);

    const { result } = renderHook(() => useRBACPermissions('backup-classes'));

    await waitFor(() => expect(result.current.canRead).toBe(true));
  });

  it('matches a cluster-wide grant when checking every namespace', async () => {
    await grantRBAC(['instances', 'read', 'main/*/*']);

    const { result } = renderHook(() => useRBACPermissions('instances'));

    await waitFor(() => expect(result.current.canRead).toBe(true));
  });

  it('does not grant read-connection with read', async () => {
    await grantRBAC(['instances', 'read', 'main/ns1/db1']);

    const { result } = renderHook(() =>
      useRBACAction('read-connection', 'instances', 'ns1/db1')
    );

    await waitFor(() =>
      expect(result.current.isPermissionsLoading).toBe(false)
    );
    expect(result.current.isAllowed).toBe(false);
  });

  it('grants read-connection with the * action', async () => {
    await grantRBAC(['instances', '*', 'main/ns1/db1']);

    const { result } = renderHook(() =>
      useRBACAction('read-connection', 'instances', 'ns1/db1')
    );

    await waitFor(() => expect(result.current.isAllowed).toBe(true));
  });
});
