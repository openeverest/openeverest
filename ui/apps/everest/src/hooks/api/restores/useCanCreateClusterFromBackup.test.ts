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
import { useCanCreateClusterFromBackup } from './useCanCreateClusterFromBackup';

const mockUseCanRestore = vi.fn();
const mockUseRBACPermissions = vi.fn();

vi.mock('./useCanRestore', () => ({
  useCanRestore: (...args: unknown[]) => mockUseCanRestore(...args),
}));

vi.mock('hooks/rbac', () => ({
  useRBACPermissions: (...args: unknown[]) => mockUseRBACPermissions(...args),
}));

describe('useCanCreateClusterFromBackup', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockUseRBACPermissions.mockReturnValue({ canCreate: false });
  });

  it('returns true only when restore and create permissions are both granted', () => {
    mockUseCanRestore.mockReturnValue(true);
    mockUseRBACPermissions.mockReturnValue({ canCreate: true });

    const { result } = renderHook(() =>
      useCanCreateClusterFromBackup('ns-a', 'inst-a')
    );

    expect(result.current).toBe(true);
  });

  it('returns false when restore permission is missing', () => {
    mockUseCanRestore.mockReturnValue(false);
    mockUseRBACPermissions.mockReturnValue({ canCreate: true });

    const { result } = renderHook(() =>
      useCanCreateClusterFromBackup('ns-a', 'inst-a')
    );

    expect(result.current).toBe(false);
  });

  it('returns false when cluster create permission is missing', () => {
    mockUseCanRestore.mockReturnValue(true);
    mockUseRBACPermissions.mockReturnValue({ canCreate: false });

    const { result } = renderHook(() =>
      useCanCreateClusterFromBackup('ns-a', 'inst-a')
    );

    expect(result.current).toBe(false);
  });

  it('checks cluster creation permission in the source namespace wildcard scope', () => {
    mockUseCanRestore.mockReturnValue(true);
    mockUseRBACPermissions.mockReturnValue({ canCreate: true });

    renderHook(() => useCanCreateClusterFromBackup('team-ns', 'inst-a'));

    expect(mockUseRBACPermissions).toHaveBeenCalledWith(
      'database-clusters',
      'team-ns/*'
    );
  });
});
