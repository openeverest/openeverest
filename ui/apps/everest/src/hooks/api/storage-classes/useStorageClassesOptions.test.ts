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
import { Messages } from 'components/ui-generator/api-providers/providers.messages';
import { useStorageClassesOptions } from './useStorageClassesOptions';

const { storageClassesQuery, permissions } = vi.hoisted(() => ({
  storageClassesQuery: {
    data: undefined as
      | { metadata?: { name?: string }; provisioner: string }[]
      | undefined,
    isLoading: false,
    error: null,
  },
  permissions: { canRead: true, isPermissionsLoading: false },
}));

vi.mock('./useStorageClasses', () => ({
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

  it('maps storage classes to options by name', () => {
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

  it('explains an empty list by the missing read permission', () => {
    storageClassesQuery.data = [];
    permissions.canRead = false;

    const { result } = renderHook(() => useStorageClassesOptions(PARAMS));

    expect(result.current.isEmpty).toBe(true);
    expect(result.current.emptyHint).toBe(
      Messages.storageClassesNoReadPermission
    );
  });

  it.each([
    ['the user can read storage classes', { canRead: true, loading: false }],
    ['permissions are still loading', { canRead: false, loading: true }],
  ])(
    'gives no permission hint for an empty list when %s',
    (_case, { canRead, loading }) => {
      storageClassesQuery.data = [];
      permissions.canRead = canRead;
      permissions.isPermissionsLoading = loading;

      const { result } = renderHook(() => useStorageClassesOptions(PARAMS));

      expect(result.current.isEmpty).toBe(true);
      expect(result.current.emptyHint).toBeUndefined();
    }
  );

  it('gives no permission hint when the API returned classes the user may read', () => {
    storageClassesQuery.data = [
      { metadata: { name: 'standard' }, provisioner: 'p' },
    ];
    permissions.canRead = false;

    const { result } = renderHook(() => useStorageClassesOptions(PARAMS));

    expect(result.current.emptyHint).toBeUndefined();
  });
});
