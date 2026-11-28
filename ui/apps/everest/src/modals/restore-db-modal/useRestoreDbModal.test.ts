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
import { grantRBAC, resetRBAC } from 'api/__mocks__/policies';
import { BackupStatus } from 'shared-types/backups.types';
import { getRestoresListQueryKey } from 'hooks/api/restores/useInstanceRestores';
import { stopAuthorizerFetchLoop } from 'utils/rbac';
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

vi.mock('api/policies');

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

const availablePitr = {
  available: true,
  earliest: '2026-07-29T08:00:00Z',
  latest: '2026-07-29T10:00:00Z',
};

const renderRestoreHook = async (
  params: Partial<Parameters<typeof useRestoreDbModal>[0]> = {}
) => {
  const { result } = renderHook(() =>
    useRestoreDbModal({
      instanceName: 'source-db',
      namespace: 'ns-a',
      isNewClusterMode: false,
      closeModal: mockCloseModal,
      ...params,
    })
  );

  await waitFor(() => expect(result.current.isLoading).toBe(false));
  return result;
};

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

  afterEach(() => {
    stopAuthorizerFetchLoop();
  });

  it('navigates to create-db flow for navigate-new action', async () => {
    mockResolveRestoreAction.mockReturnValue({
      kind: 'navigate-new',
      dataSource: { type: 'Backup', backupName: 'daily-1' },
    });

    const result = await renderRestoreHook({ isNewClusterMode: true });

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

  it('creates restore and performs success side-effects for restore action', async () => {
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

    const result = await renderRestoreHook();

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

  it('does nothing when restore action resolves to none', async () => {
    mockResolveRestoreAction.mockReturnValue({ kind: 'none' });

    const result = await renderRestoreHook({ isNewClusterMode: true });

    result.current.onSubmit({} as never);

    expect(mockMutate).not.toHaveBeenCalled();
    expect(mockNavigate).not.toHaveBeenCalled();
    expect(mockInvalidateQueries).not.toHaveBeenCalled();
    expect(mockEnqueueSnackbar).not.toHaveBeenCalled();
    expect(mockCloseModal).not.toHaveBeenCalled();
  });

  it('preselects backup name in form defaults and sorts succeeded backups by startedAt', async () => {
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

    const result = await renderRestoreHook({
      isNewClusterMode: true,
      preselectedBackupName: 'newer',
    });

    expect(result.current.formDefaultValues[RestoreDbFields.backupName]).toBe(
      'newer'
    );
    expect(result.current.succeededBackups.map((item) => item.name)).toEqual([
      'newer',
      'older',
    ]);
  });
});

// Isolated: grantRBAC enables RBAC for alice and would leak into later tests
// without resetRBAC.
describe('useRestoreDbModal RBAC filtering', () => {
  beforeEach(() => {
    vi.clearAllMocks();

    mockUseClusterName.mockReturnValue('main');
    mockUseBackupsList.mockReturnValue({ data: [], isLoading: false });
    mockUseDbInstance.mockReturnValue({ data: undefined });
    mockUseCreateInstanceRestore.mockReturnValue({
      mutate: mockMutate,
      isPending: false,
    });
  });

  afterEach(async () => {
    await resetRBAC();
    stopAuthorizerFetchLoop();
  });

  it('only offers PITR storages the user can read', async () => {
    mockUseDbInstance.mockReturnValue({
      data: {
        status: {
          backup: {
            storages: [
              {
                name: 'readable-storage',
                pitr: availablePitr,
              },
              {
                name: 'unreadable-storage',
                pitr: availablePitr,
              },
            ],
          },
        },
      },
    });
    await grantRBAC(['backup-storages', 'read', 'main/ns-a/readable-storage']);

    const result = await renderRestoreHook();

    expect(result.current.pitrStorages.map((storage) => storage.name)).toEqual([
      'readable-storage',
    ]);
    expect(result.current.isPitrHiddenByPermissions).toBe(false);
  });

  it('hides PITR when the instance has storages but none are readable', async () => {
    mockUseDbInstance.mockReturnValue({
      data: {
        status: {
          backup: {
            storages: [
              {
                name: 'unreadable-storage',
                pitr: availablePitr,
              },
            ],
          },
        },
      },
    });
    await grantRBAC(['backup-storages', 'read', 'main/ns-a/other-storage']);

    const result = await renderRestoreHook();

    expect(result.current.pitrStorages).toEqual([]);
    expect(result.current.isPitrHiddenByPermissions).toBe(true);
  });
});
