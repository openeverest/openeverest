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
import type { StorageClass } from 'shared-types/kubernetes.types';
import { Messages } from '../providers.messages';
import { useStorageClassesOptions } from './useStorageClassesOptions';

// RBAC and the query are mocked here to reach branches the providers
// integration test doesn't: name mapping and permissions still loading.
type StorageClassesQueryMock = {
  data?: StorageClass[];
  isLoading: boolean;
  error: null;
};

const { storageClassesQuery, permissions } = vi.hoisted(() => {
  const query: StorageClassesQueryMock = { isLoading: false, error: null };
  return {
    storageClassesQuery: query,
    permissions: { canRead: true, isPermissionsLoading: false },
  };
});

vi.mock('hooks/api/storage-classes/useStorageClasses', () => ({
  useStorageClassesList: () => storageClassesQuery,
}));

vi.mock('hooks/rbac', () => ({
  useRBACPermissions: () => permissions,
}));

const PARAMS = { namespace: 'ns', cluster: 'main' };

describe('useStorageClassesOptions', () => {
  beforeEach(() => {
    storageClassesQuery.data = undefined;
    storageClassesQuery.isLoading = false;
    permissions.canRead = true;
    permissions.isPermissionsLoading = false;
  });

  it('maps storage classes to options by name, skipping nameless ones', () => {
    storageClassesQuery.data = [
      { metadata: { name: 'standard' }, provisioner: 'p' },
      { metadata: {}, provisioner: 'p' },
      { metadata: { name: 'fast' }, provisioner: 'p' },
    ];

    const { result } = renderHook(() => useStorageClassesOptions(PARAMS));

    expect(result.current.options).toEqual([
      { label: 'standard', value: 'standard' },
      { label: 'fast', value: 'fast' },
    ]);
    expect(result.current.isEmpty).toBe(false);
  });

  it('gives no permission hint while permissions are still loading', () => {
    storageClassesQuery.data = [];
    permissions.canRead = false;
    permissions.isPermissionsLoading = true;

    const { result } = renderHook(() => useStorageClassesOptions(PARAMS));

    expect(result.current.isEmpty).toBe(true);
    expect(result.current.emptyHint).not.toBe(
      Messages.storageClassesNoReadPermission
    );
  });
});
