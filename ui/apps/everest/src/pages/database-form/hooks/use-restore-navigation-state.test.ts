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
import { useRestoreNavigationState } from './use-restore-navigation-state';

const mockUseLocation = vi.fn();

vi.mock('react-router-dom', () => ({
  useLocation: () => mockUseLocation(),
}));

describe('useRestoreNavigationState', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('marks restore mode for a valid backup clone payload', () => {
    const restoreDataSource = {
      type: 'Backup',
      backupName: 'daily-1',
      storageName: 's3-main',
    };

    mockUseLocation.mockReturnValue({
      state: {
        selectedDbCluster: 'source-db',
        namespace: 'ns-a',
        restoreDataSource,
      },
    });

    const { result } = renderHook(() => useRestoreNavigationState());

    expect(result.current).toEqual({
      sourceInstanceName: 'source-db',
      sourceNamespace: 'ns-a',
      restoreDataSource,
      isRestore: true,
    });
  });

  it('marks restore mode for a valid PITR-date clone payload', () => {
    const restoreDataSource = {
      type: 'PointInTime',
      storageName: 's3-main',
      recoveryTarget: 'date',
      date: '2026-07-29T10:15:00Z',
      sourceInstanceName: 'source-db',
    };

    mockUseLocation.mockReturnValue({
      state: {
        selectedDbCluster: 'source-db',
        namespace: 'ns-a',
        restoreDataSource,
      },
    });

    const { result } = renderHook(() => useRestoreNavigationState());

    expect(result.current.isRestore).toBe(true);
    expect(result.current.restoreDataSource).toEqual(restoreDataSource);
  });

  it('falls back to non-restore mode when payload shape is invalid', () => {
    mockUseLocation.mockReturnValue({
      state: {
        selectedDbCluster: 'source-db',
        namespace: 'ns-a',
        restoreDataSource: {
          type: 'PointInTime',
          storageName: 's3-main',
          recoveryTarget: 'date',
        },
      },
    });

    const { result } = renderHook(() => useRestoreNavigationState());

    expect(result.current).toEqual({
      sourceInstanceName: 'source-db',
      sourceNamespace: 'ns-a',
      restoreDataSource: undefined,
      isRestore: false,
    });
  });

  it('falls back to non-restore mode when source identity is incomplete', () => {
    mockUseLocation.mockReturnValue({
      state: {
        selectedDbCluster: 'source-db',
        restoreDataSource: {
          type: 'Backup',
          backupName: 'daily-1',
        },
      },
    });

    const { result } = renderHook(() => useRestoreNavigationState());

    expect(result.current.isRestore).toBe(false);
    expect(result.current.sourceNamespace).toBeUndefined();
  });
});
