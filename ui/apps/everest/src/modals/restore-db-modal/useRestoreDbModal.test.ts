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
import { BackupStatus } from 'shared-types/backups.types';
import { getRestoresListQueryKey } from 'hooks/api/restores/useInstanceRestores';
import { Messages } from './restore-db-modal.messages';
import { RestoreDbFields } from './restore-db-modal-schema';
import { useRestoreDbModal } from './useRestoreDbModal';

const mockNavigate = vi.fn();
const mockInvalidateQueries = vi.fn();
const mockMutate = vi.fn();
const mockCloseModal = vi.fn();

const mockUseBackupsList = vi.fn();
const mockUseDbInstance = vi.fn();
const mockUseCreateInstanceRestore = vi.fn();
const mockUseClusterName = vi.fn();
const mockResolveRestoreAction = vi.fn();
const mockEnqueueSnackbar = vi.fn();

vi.mock('react-router-dom', () => ({
  useNavigate: () => mockNavigate,
}));

vi.mock('@tanstack/react-query', () => ({
  useQueryClient: () => ({
    invalidateQueries: mockInvalidateQueries,
  }),
}));

vi.mock('notistack', () => ({
  enqueueSnackbar: (...args: unknown[]) => mockEnqueueSnackbar(...args),
}));

vi.mock('hooks/api/backups/useBackups', () => ({
  useBackupsList: (...args: unknown[]) => mockUseBackupsList(...args),
}));

vi.mock('hooks/api/db-instances', () => ({
  useDbInstance: (...args: unknown[]) => mockUseDbInstance(...args),
}));

vi.mock('hooks/api/restores/useInstanceRestores', async () => {
  const actual = await vi.importActual<
    typeof import('hooks/api/restores/useInstanceRestores')
  >('hooks/api/restores/useInstanceRestores');

  return {
    ...actual,
    useCreateInstanceRestore: (...args: unknown[]) =>
      mockUseCreateInstanceRestore(...args),
  };
});

vi.mock('hooks/api/useClusterName', () => ({
  useClusterName: () => mockUseClusterName(),
}));

vi.mock('./resolve-restore-action', () => ({
  resolveRestoreAction: (...args: unknown[]) =>
    mockResolveRestoreAction(...args),
}));

describe('useRestoreDbModal', () => {
  beforeEach(() => {
    vi.clearAllMocks();

    mockUseClusterName.mockReturnValue('main-cluster');
    mockUseBackupsList.mockReturnValue({ data: [], isLoading: false });
    mockUseDbInstance.mockReturnValue({ data: undefined });
    mockUseCreateInstanceRestore.mockReturnValue({
      mutate: mockMutate,
      isPending: false,
    });
  });

  it('navigates to create-db flow for navigate-new action', () => {
    mockResolveRestoreAction.mockReturnValue({
      kind: 'navigate-new',
      dataSource: { type: 'Backup', backupName: 'daily-1' },
    });

    const { result } = renderHook(() =>
      useRestoreDbModal({
        instanceName: 'source-db',
        namespace: 'ns-a',
        isNewClusterMode: true,
        closeModal: mockCloseModal,
      })
    );

    result.current.onSubmit({} as never);

    expect(mockCloseModal).toHaveBeenCalledTimes(1);
    expect(mockNavigate).toHaveBeenCalledWith('/databases/new', {
      state: {
        selectedDbCluster: 'source-db',
        namespace: 'ns-a',
        restoreDataSource: { type: 'Backup', backupName: 'daily-1' },
      },
    });
    expect(mockMutate).not.toHaveBeenCalled();
  });

  it('creates restore and performs success side-effects for restore action', () => {
    mockResolveRestoreAction.mockReturnValue({
      kind: 'restore',
      variables: {
        instanceName: 'source-db',
        type: 'Backup',
        backupName: 'daily-1',
      },
    });

    mockMutate.mockImplementation(
      (
        _variables: unknown,
        options?: {
          onSuccess?: () => void;
        }
      ) => {
        options?.onSuccess?.();
      }
    );

    const { result } = renderHook(() =>
      useRestoreDbModal({
        instanceName: 'source-db',
        namespace: 'ns-a',
        isNewClusterMode: false,
        closeModal: mockCloseModal,
      })
    );

    result.current.onSubmit({} as never);

    expect(mockMutate).toHaveBeenCalledWith(
      {
        instanceName: 'source-db',
        type: 'Backup',
        backupName: 'daily-1',
      },
      expect.any(Object)
    );
    expect(mockInvalidateQueries).toHaveBeenCalledWith({
      queryKey: getRestoresListQueryKey('main-cluster', 'ns-a', 'source-db'),
    });
    expect(mockEnqueueSnackbar).toHaveBeenCalledWith(Messages.restoreStarted, {
      variant: 'success',
    });
    expect(mockCloseModal).toHaveBeenCalledTimes(1);
    expect(mockNavigate).not.toHaveBeenCalled();
  });

  it('does nothing when restore action resolves to none', () => {
    mockResolveRestoreAction.mockReturnValue({ kind: 'none' });

    const { result } = renderHook(() =>
      useRestoreDbModal({
        instanceName: 'source-db',
        namespace: 'ns-a',
        isNewClusterMode: true,
        closeModal: mockCloseModal,
      })
    );

    result.current.onSubmit({} as never);

    expect(mockMutate).not.toHaveBeenCalled();
    expect(mockNavigate).not.toHaveBeenCalled();
    expect(mockInvalidateQueries).not.toHaveBeenCalled();
    expect(mockEnqueueSnackbar).not.toHaveBeenCalled();
    expect(mockCloseModal).not.toHaveBeenCalled();
  });

  it('preselects backup name in form defaults and sorts succeeded backups by startedAt', () => {
    mockUseBackupsList.mockReturnValue({
      isLoading: false,
      data: [
        {
          metadata: { name: 'older' },
          spec: { storageRef: { name: 's3-2' } },
          status: {
            state: BackupStatus.SUCCEEDED,
            startedAt: '2026-07-29T08:00:00Z',
          },
        },
        {
          metadata: { name: 'newer' },
          spec: { storageRef: { name: 's3-1' } },
          status: {
            state: BackupStatus.SUCCEEDED,
            startedAt: '2026-07-29T10:00:00Z',
          },
        },
      ],
    });

    const { result } = renderHook(() =>
      useRestoreDbModal({
        instanceName: 'source-db',
        namespace: 'ns-a',
        isNewClusterMode: true,
        preselectedBackupName: 'newer',
        closeModal: mockCloseModal,
      })
    );

    expect(result.current.formDefaultValues[RestoreDbFields.backupName]).toBe(
      'newer'
    );
    expect(result.current.succeededBackups.map((item) => item.name)).toEqual([
      'newer',
      'older',
    ]);
  });
});
