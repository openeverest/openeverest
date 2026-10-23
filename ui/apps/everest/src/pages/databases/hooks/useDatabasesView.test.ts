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

import { renderHook } from '@testing-library/react';
import { useDatabasesView } from './useDatabasesView';

const mockUseNamespacePermissionsForResource = vi.fn();

vi.mock('hooks/api/namespaces/useNamespaces', () => ({
  useNamespaces: () => ({ data: ['ns-a'], isLoading: false }),
}));

vi.mock('hooks/api/providers/useProviders', () => ({
  useProviders: () => ({
    data: [{ metadata: { name: 'psmdb' } }],
    isLoading: false,
  }),
}));

vi.mock('hooks/api/db-instances/useDbInstanceList', () => ({
  useInstancesForNamespaces: () => [],
}));

vi.mock('hooks/api/extension-catalog', () => ({
  useExtensionCatalog: () => ({ isLoading: false }),
}));

vi.mock('hooks/rbac', () => ({
  useNamespacePermissionsForResource: (...args: unknown[]) =>
    mockUseNamespacePermissionsForResource(...args),
}));

// Grants create only on the given v2 resources, mimicking a policy that has
// no legacy v1 (database-clusters) permissions.
const grantOnly = (resources: string[]) =>
  mockUseNamespacePermissionsForResource.mockImplementation(
    (resource: string) => ({
      canCreate: resources.includes(resource) ? ['ns-a'] : [],
      isPermissionsLoading: false,
    })
  );

describe('useDatabasesView RBAC', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('allows creating with a v2-only policy (instances)', () => {
    grantOnly(['instances']);

    const { result } = renderHook(() => useDatabasesView());

    expect(mockUseNamespacePermissionsForResource).toHaveBeenCalledWith(
      'instances'
    );
    expect(result.current.hasCreatePermission).toBe(true);
    expect(result.current.canAddCluster).toBe(true);
  });

  it('does not allow creating with only the legacy v1 resource', () => {
    grantOnly(['database-clusters']);

    const { result } = renderHook(() => useDatabasesView());

    expect(result.current.hasCreatePermission).toBe(false);
    expect(result.current.canAddCluster).toBe(false);
  });
});
