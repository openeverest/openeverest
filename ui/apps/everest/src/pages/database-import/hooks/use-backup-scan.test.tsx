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

import { ReactNode } from 'react';
import { act, renderHook, waitFor } from '@testing-library/react';
import {
  focusManager,
  QueryClient,
  QueryClientProvider,
} from '@tanstack/react-query';
import { createBackupImportFn, getBackupImportsFn } from 'api/backup-imports';
import { getBackupImportsQueryKey } from 'hooks/api/backup-imports';
import { getNamespaceBackupsKeyPrefix } from 'hooks/api/backups/useBackups';
import { BackupImport, BackupImportList } from 'shared-types/backups.types';
import { useBackupScan } from './use-backup-scan';

vi.mock('api/backup-imports', () => ({
  getBackupImportsFn: vi.fn(),
  createBackupImportFn: vi.fn(),
}));
vi.mock('hooks/api/useClusterName', () => ({ useClusterName: () => 'main' }));

const target = {
  namespace: 'ns-a',
  storageName: 's3-old',
  backupClassName: 'psmdb-pbm',
};
const spec: BackupImport['spec'] = {
  classRef: { name: 'psmdb-pbm' },
  storageRef: { name: 's3-old' },
};

const backupImport = (
  name: string,
  status: BackupImport['status'],
  minutesAgo = 0,
  importSpec = spec
): BackupImport => ({
  metadata: {
    name,
    creationTimestamp: new Date(
      Date.now() - minutesAgo * 60 * 1000
    ).toISOString(),
  },
  spec: importSpec,
  status,
});

const listOf = (...items: BackupImport[]): BackupImportList => ({ items });

const renderScan = () => {
  const queryClient = new QueryClient({
    // Same focus default as the app's client.
    defaultOptions: { queries: { retry: false, refetchOnWindowFocus: false } },
  });
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
  return {
    queryClient,
    ...renderHook(useBackupScan, { initialProps: target, wrapper }),
  };
};

describe('useBackupScan', () => {
  beforeEach(() => {
    vi.mocked(getBackupImportsFn).mockReset();
    vi.mocked(createBackupImportFn).mockReset();
  });

  afterEach(() => {
    vi.useRealTimers();
    focusManager.setFocused(undefined);
  });

  it('reloads the scan state when the tab regains focus', async () => {
    vi.mocked(getBackupImportsFn).mockResolvedValue(
      listOf(backupImport('import-a', { state: 'Succeeded' }))
    );
    const { result } = renderScan();
    await waitFor(() => expect(result.current.state).toBe('Succeeded'));
    expect(getBackupImportsFn).toHaveBeenCalledTimes(1);

    act(() => {
      focusManager.setFocused(false);
      focusManager.setFocused(true);
    });

    await waitFor(() => expect(getBackupImportsFn).toHaveBeenCalledTimes(2));
  });

  it("shows the target's newest scan, whoever started it", async () => {
    vi.mocked(getBackupImportsFn).mockResolvedValue(
      listOf(
        backupImport('old', { state: 'Succeeded' }, 60),
        backupImport('new', { state: 'Pending' }, 1),
        backupImport('other', { state: 'Failed' }, 0, {
          ...spec,
          storageRef: { name: 's3-other' },
        })
      )
    );

    const { result } = renderScan();

    await waitFor(() => expect(result.current.state).toBe('Pending'));
    expect(result.current).toMatchObject({
      isScanning: true,
      canScan: false,
      hasScanned: true,
    });
  });

  it('starts a scan of the storage with the class', async () => {
    vi.mocked(getBackupImportsFn).mockResolvedValue(listOf());
    vi.mocked(createBackupImportFn).mockResolvedValue(
      backupImport('import-x1', undefined)
    );
    const { result } = renderScan();
    await waitFor(() => expect(result.current.canScan).toBe(true));

    act(() => result.current.scan());

    await waitFor(() =>
      expect(createBackupImportFn).toHaveBeenCalledWith('main', 'ns-a', {
        metadata: { generateName: 'import-' },
        spec,
      })
    );
    await waitFor(() => expect(getBackupImportsFn).toHaveBeenCalledTimes(2));
  });

  it.each([
    ['an erroring scan', backupImport('x', { state: 'Error' }, 1)],
    ['a stalled scan', backupImport('x', { state: 'Pending' }, 3)],
  ])('lets %s be started again', async (_, current) => {
    vi.mocked(getBackupImportsFn).mockResolvedValue(listOf(current));

    const { result } = renderScan();

    await waitFor(() => expect(result.current.canScan).toBe(true));
    expect(result.current.isScanning).toBe(true);
  });

  it('refreshes the backups when a scan it follows succeeds', async () => {
    vi.mocked(getBackupImportsFn).mockResolvedValue(
      listOf(backupImport('x', { state: 'Pending' }))
    );
    const { result, queryClient } = renderScan();
    await waitFor(() => expect(result.current.state).toBe('Pending'));
    const invalidate = vi.spyOn(queryClient, 'invalidateQueries');

    vi.mocked(getBackupImportsFn).mockResolvedValue(
      listOf(backupImport('x', { state: 'Succeeded', discoveredCount: 2 }))
    );
    await act(() =>
      queryClient.refetchQueries({
        queryKey: getBackupImportsQueryKey('main', 'ns-a'),
      })
    );

    await waitFor(() =>
      expect(invalidate).toHaveBeenCalledWith({
        queryKey: getNamespaceBackupsKeyPrefix('main', 'ns-a'),
      })
    );
  });

  it('stops polling once the list fails', async () => {
    vi.useFakeTimers();
    vi.mocked(getBackupImportsFn)
      .mockResolvedValueOnce(listOf(backupImport('x', { state: 'Pending' })))
      .mockRejectedValue(new Error('unavailable'));
    renderScan();

    await act(() => vi.advanceTimersByTimeAsync(2100));
    expect(getBackupImportsFn).toHaveBeenCalledTimes(2);

    await act(() => vi.advanceTimersByTimeAsync(10000));
    expect(getBackupImportsFn).toHaveBeenCalledTimes(2);
  });
});
