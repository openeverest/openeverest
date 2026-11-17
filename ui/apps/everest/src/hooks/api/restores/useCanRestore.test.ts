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
import { useRBACPermissions } from 'hooks/rbac';
import { stopAuthorizerFetchLoop } from 'utils/rbac';
import { useCanRestore } from './useCanRestore';

vi.mock('api/policies');

// Mirrors the server's CreateRestore rule (#3310), keyed by the target instance.
describe('useCanRestore with v2 policies', () => {
  afterEach(() => {
    stopAuthorizerFetchLoop();
  });

  it('allows restore with restores:create and backups:read', async () => {
    await grantRBAC(
      ['restores', 'create', 'main/ns1/db1'],
      ['backups', 'read', 'main/ns1/db1']
    );

    const { result } = renderHook(() => useCanRestore('ns1', 'db1'));

    await waitFor(() => expect(result.current).toBe(true));
  });

  it.each([
    ['backups:read is missing', [['restores', 'create', 'main/ns1/db1']]],
    ['restores:create is missing', [['backups', 'read', 'main/ns1/db1']]],
    [
      'grants cover another instance',
      [
        ['restores', 'create', 'main/ns1/db2'],
        ['backups', 'read', 'main/ns1/db2'],
      ],
    ],
  ])('denies restore when %s', async (_, rules) => {
    await grantRBAC(...rules);

    const { result } = renderHook(() => ({
      canRestore: useCanRestore('ns1', 'db1'),
      // Settles together with the checks inside useCanRestore.
      backups: useRBACPermissions('backups', 'ns1/db1'),
    }));

    await waitFor(() =>
      expect(result.current.backups.isPermissionsLoading).toBe(false)
    );
    expect(result.current.canRestore).toBe(false);
  });
});
