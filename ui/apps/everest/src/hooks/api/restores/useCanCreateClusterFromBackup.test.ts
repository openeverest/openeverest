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
  const setPermissions = (permissions: {
    canCreateClusters?: boolean;
    canCreateBackups?: boolean;
    canReadMonitoring?: boolean;
  }) => {
    mockUseRBACPermissions.mockImplementation((resource: string) => {
      if (resource === 'database-clusters') {
        return { canCreate: permissions.canCreateClusters ?? false };
      }
      if (resource === 'database-cluster-backups') {
        return { canCreate: permissions.canCreateBackups ?? false };
      }
      if (resource === 'monitoring-configs') {
        return { canRead: permissions.canReadMonitoring ?? false };
      }
      return { canCreate: false, canRead: false };
    });
  };

  beforeEach(() => {
    vi.clearAllMocks();
    setPermissions({
      canCreateClusters: false,
      canCreateBackups: false,
      canReadMonitoring: false,
    });
  });

  it('returns true only when restore and create permissions are both granted', () => {
    mockUseCanRestore.mockReturnValue(true);
    setPermissions({ canCreateClusters: true });

    const { result } = renderHook(() =>
      useCanCreateClusterFromBackup('ns-a', 'inst-a')
    );

    expect(result.current).toBe(true);
  });

  it('returns false when restore permission is missing', () => {
    mockUseCanRestore.mockReturnValue(false);
    setPermissions({ canCreateClusters: true });

    const { result } = renderHook(() =>
      useCanCreateClusterFromBackup('ns-a', 'inst-a')
    );

    expect(result.current).toBe(false);
  });

  it('returns false when cluster create permission is missing', () => {
    mockUseCanRestore.mockReturnValue(true);
    setPermissions({ canCreateClusters: false });

    const { result } = renderHook(() =>
      useCanCreateClusterFromBackup('ns-a', 'inst-a')
    );

    expect(result.current).toBe(false);
  });

  it('checks cluster creation permission in the source namespace wildcard scope', () => {
    mockUseCanRestore.mockReturnValue(true);
    setPermissions({ canCreateClusters: true });

    renderHook(() => useCanCreateClusterFromBackup('team-ns', 'inst-a'));

    expect(mockUseRBACPermissions).toHaveBeenCalledWith(
      'database-clusters',
      'team-ns/*'
    );
  });

  it('requires backup create permission when schedules exist', () => {
    mockUseCanRestore.mockReturnValue(true);
    setPermissions({
      canCreateClusters: true,
      canCreateBackups: false,
    });

    const { result } = renderHook(() =>
      useCanCreateClusterFromBackup('ns-a', 'inst-a', { hasSchedules: true })
    );

    expect(result.current).toBe(false);
  });

  it('requires monitoring read permission when monitoring is enabled', () => {
    mockUseCanRestore.mockReturnValue(true);
    setPermissions({
      canCreateClusters: true,
      canReadMonitoring: false,
    });

    const { result } = renderHook(() =>
      useCanCreateClusterFromBackup('ns-a', 'inst-a', {
        monitoringEnabled: true,
      })
    );

    expect(result.current).toBe(false);
  });

  it('passes when all required permissions for schedules and monitoring are present', () => {
    mockUseCanRestore.mockReturnValue(true);
    setPermissions({
      canCreateClusters: true,
      canCreateBackups: true,
      canReadMonitoring: true,
    });

    const { result } = renderHook(() =>
      useCanCreateClusterFromBackup('ns-a', 'inst-a', {
        hasSchedules: true,
        monitoringEnabled: true,
      })
    );

    expect(result.current).toBe(true);
  });
});
