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

import React from 'react';
import { act, renderHook } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { AxiosError, AxiosHeaders } from 'axios';
import { api } from 'api/api';
import { SECRETS_QUERY_KEY } from 'hooks/api/secrets';
import {
  BackupStorageCRD,
  BackupStoragePayload,
} from 'shared-types/backupStorages.types';
import {
  getBackupStoragesQueryKey,
  useDeleteBackupStorage,
  useSaveBackupStorage,
} from './useBackupStorages';

vi.mock('api/api', () => ({
  api: { delete: vi.fn(), get: vi.fn(), post: vi.fn(), put: vi.fn() },
}));

const wrapperFor =
  (queryClient: QueryClient) =>
  ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
vi.mock('hooks/api/useClusterName', () => ({ useClusterName: () => 'main' }));

describe('useDeleteBackupStorage', () => {
  it("refreshes every list of the storage's namespace, the pickers' too", async () => {
    vi.mocked(api.delete).mockResolvedValue({});
    const queryClient = new QueryClient();
    const teamAList = getBackupStoragesQueryKey('main', 'team-a');
    const teamAPickerList = getBackupStoragesQueryKey('main', 'team-a', {
      backupClass: 'psmdb-pbm',
    });
    const teamBList = getBackupStoragesQueryKey('main', 'team-b');
    [teamAList, teamAPickerList, teamBList].forEach((key) =>
      queryClient.setQueryData<BackupStorageCRD[]>(key, [])
    );
    const { result } = renderHook(() => useDeleteBackupStorage(), {
      wrapper: wrapperFor(queryClient),
    });

    await act(() =>
      result.current.mutateAsync({ namespace: 'team-a', name: 's3-prod' })
    );

    expect(api.delete).toHaveBeenCalledWith(
      'clusters/main/namespaces/team-a/backup-storages/s3-prod'
    );
    expect(queryClient.getQueryState(teamAList)?.isInvalidated).toBe(true);
    expect(queryClient.getQueryState(teamAPickerList)?.isInvalidated).toBe(
      true
    );
    expect(queryClient.getQueryState(teamBList)?.isInvalidated).toBe(false);
  });
});

describe('useSaveBackupStorage', () => {
  it("refreshes the namespace's storages and Secrets, also when the storage is refused", async () => {
    vi.mocked(api.post)
      .mockResolvedValueOnce({ data: {} })
      .mockRejectedValueOnce(new Error('refused'));
    vi.mocked(api.delete).mockResolvedValue({});
    const queryClient = new QueryClient();
    const storages = getBackupStoragesQueryKey('main', 'team-a');
    const secrets = [SECRETS_QUERY_KEY, 'main', 'team-a', {}];
    queryClient.setQueryData<BackupStorageCRD[]>(storages, []);
    queryClient.setQueryData(secrets, []);
    const storage: BackupStoragePayload = {
      metadata: { name: 's3-prod' },
      spec: {
        backupStorageClassRef: { name: 's3' },
        authentication: {
          mode: 'AccessKey',
          credentialsSecretRef: { name: 'backup-storage-s3-prod-credentials' },
        },
        location: { objectStore: { bucket: 'backups' } },
      },
    };
    const { result } = renderHook(() => useSaveBackupStorage(), {
      wrapper: wrapperFor(queryClient),
    });

    await act(() =>
      result.current
        .mutateAsync({
          namespace: 'team-a',
          storage,
          credentialsSecret: {
            action: 'create',
            name: 'backup-storage-s3-prod-credentials',
            labels: {},
            stringData: {},
          },
        })
        .catch(() => undefined)
    );

    expect(api.post).toHaveBeenLastCalledWith(
      'clusters/main/namespaces/team-a/backup-storages',
      storage
    );
    expect(queryClient.getQueryState(storages)?.isInvalidated).toBe(true);
    expect(queryClient.getQueryState(secrets)?.isInvalidated).toBe(true);
  });

  describe('an edit refused as made against an old version', () => {
    const conflict = new AxiosError('conflict', '409', undefined, undefined, {
      status: 409,
      statusText: 'Conflict',
      data: {},
      headers: {},
      config: { headers: new AxiosHeaders() },
    });
    const savedSpec: BackupStoragePayload['spec'] = {
      backupStorageClassRef: { name: 's3' },
      authentication: {
        mode: 'AccessKey',
        credentialsSecretRef: { name: 's3-prod-credentials' },
      },
      location: { objectStore: { bucket: 'backups' } },
      parameters: { region: 'us-east-1' },
    };
    const edit: BackupStoragePayload = {
      metadata: { name: 's3-prod', resourceVersion: '5' },
      spec: { ...savedSpec, parameters: { region: 'eu-west-1' } },
    };
    const path = 'clusters/main/namespaces/team-a/backup-storages/s3-prod';
    const renderSave = () => {
      const { result } = renderHook(() => useSaveBackupStorage(), {
        wrapper: wrapperFor(new QueryClient()),
      });
      return () =>
        result.current.mutateAsync({
          namespace: 'team-a',
          storage: edit,
          savedSpec,
        });
    };

    beforeEach(() => {
      vi.mocked(api.get).mockReset();
      vi.mocked(api.put).mockReset().mockRejectedValueOnce(conflict);
    });

    it('goes again at the latest version when only core changed the storage, by writing its status', async () => {
      const latestMetadata = { name: 's3-prod', resourceVersion: '6' };
      vi.mocked(api.get).mockResolvedValue({
        data: { metadata: latestMetadata, spec: savedSpec, status: {} },
      });
      vi.mocked(api.put).mockResolvedValueOnce({ data: {} });
      const saveEdit = renderSave();

      await act(() => saveEdit());

      expect(api.get).toHaveBeenCalledWith(path);
      expect(api.put).toHaveBeenLastCalledWith(
        path,
        { metadata: latestMetadata, spec: edit.spec },
        expect.anything()
      );
    });

    it('is reported when the spec changed since the edit started', async () => {
      vi.mocked(api.get).mockResolvedValue({
        data: {
          metadata: { name: 's3-prod', resourceVersion: '6' },
          spec: { ...savedSpec, tls: { insecureSkipVerify: true } },
        },
      });

      const saveEdit = renderSave();

      await act(async () => {
        await expect(saveEdit()).rejects.toBe(conflict);
      });

      expect(api.put).toHaveBeenCalledTimes(1);
    });
  });
});
