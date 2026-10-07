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
import { can, canAll } from 'utils/rbac';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useRBACPermissions } from './rbac';

vi.mock('utils/rbac', () => ({
  can: vi.fn(),
  canAll: vi.fn(),
  AuthorizerObservable: {
    subscribe: vi.fn(),
    unsubscribe: vi.fn(),
  },
}));

describe('useRBACPermissions', () => {
  beforeEach(() => {
    vi.mocked(can).mockReset();
    vi.mocked(canAll).mockReset();
    vi.mocked(can).mockImplementation(
      async (action) => action === 'read-connection'
    );
    vi.mocked(canAll).mockResolvedValue(false);
  });

  it('checks the read-connection action separately from read', async () => {
    const { result } = renderHook(() =>
      useRBACPermissions('instances', 'team-a/db-a')
    );

    await waitFor(() => {
      expect(result.current.canReadConnection).toBe(true);
    });

    expect(can).toHaveBeenCalledWith(
      'read-connection',
      'instances',
      'team-a/db-a'
    );
    expect(result.current.canRead).toBe(false);
  });
});
