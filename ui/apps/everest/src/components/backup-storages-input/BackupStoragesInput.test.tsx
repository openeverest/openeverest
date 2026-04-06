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

import { DbType } from '@percona/types';
import { render, screen, fireEvent } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { FormProvider, useForm } from 'react-hook-form';
import BackupStoragesInput from '.';

const queryClient = new QueryClient();

vi.mock('hooks/api/backup-storages/useBackupStorages', () => ({
  useBackupStoragesByNamespace: () => ({
    data: [
      {
        name: 'storage1',
      },
      {
        name: 'storage2',
      },
      {
        name: 'storage3',
      },
      {
        name: 'storage4',
      },
    ],
    isFetching: false,
  }),
}));

const backupMocks = vi.hoisted(() => ({
  useDbBackups: vi.fn().mockReturnValue({ data: [], isFetching: false }),
}));

vi.mock('hooks/api/backups/useBackups', () => ({
  useDbBackups: backupMocks.useDbBackups,
}));

const FormProviderWrapper = ({ children }: { children: React.ReactNode }) => {
  const methods = useForm({
    defaultValues: {
      storageLoaction: '',
    },
  });
  return (
    <FormProvider {...methods}>
      <form>{children}</form>
    </FormProvider>
  );
};

describe.skip('BackupStoragesInput', () => {
  it('should show all available storages if not PG', async () => {
    render(
      <FormProviderWrapper>
        <QueryClientProvider client={queryClient}>
          <BackupStoragesInput
            namespace="test"
            dbType={DbType.Mysql}
            schedules={[
              {
                name: 'schedule1',
                schedule: '0 0 * * *',
                backupStorageName: 'storage1',
                enabled: true,
              },
              {
                name: 'schedule2',
                schedule: '0 0 * * *',
                backupStorageName: 'storage2',
                enabled: true,
              },
              {
                name: 'schedule3',
                schedule: '0 0 * * *',
                backupStorageName: 'storage3',
                enabled: true,
              },
            ]}
          />
        </QueryClientProvider>
      </FormProviderWrapper>
    );

    fireEvent.click(
      screen.getAllByRole('button').find((el) => el.title === 'Open')!
    );
    expect(screen.getAllByRole('option').length).toBe(4);
    expect(screen.queryByText('<SLOTS ACHIEVED>')).not.toBeInTheDocument();
  });

  it('should show only occupied storages when PG took 3 slots across backups/schedules', async () => {
    backupMocks.useDbBackups.mockReturnValue({
      data: [
        {
          backupStorageName: 'storage1',
        },
        {
          backupStorageName: 'storage2',
        },
      ],
      isFetching: false,
    });
    render(
      <FormProviderWrapper>
        <QueryClientProvider client={queryClient}>
          <BackupStoragesInput
            namespace="test"
            dbType={DbType.Postresql}
            schedules={[
              {
                name: 'schedule3',
                schedule: '0 0 * * *',
                backupStorageName: 'storage3',
                enabled: true,
              },
            ]}
          />
        </QueryClientProvider>
      </FormProviderWrapper>
    );

    fireEvent.click(
      screen.getAllByRole('button').find((el) => el.title === 'Open')!
    );
    expect(screen.getAllByRole('option').length).toBe(3);

    backupMocks.useDbBackups.mockClear();
  });

  // This is a specific case of the previous test
  it('should show only occupied storages when PG took 3 slots from schedules', async () => {
    render(
      <FormProviderWrapper>
        <QueryClientProvider client={queryClient}>
          <BackupStoragesInput
            namespace="test"
            dbType={DbType.Postresql}
            schedules={[
              {
                name: 'schedule1',
                schedule: '0 0 * * *',
                backupStorageName: 'storage1',
                enabled: true,
              },
              {
                name: 'schedule2',
                schedule: '0 0 * * *',
                backupStorageName: 'storage2',
                enabled: true,
              },
              {
                name: 'schedule3',
                schedule: '0 0 * * *',
                backupStorageName: 'storage3',
                enabled: true,
              },
            ]}
          />
        </QueryClientProvider>
      </FormProviderWrapper>
    );

    fireEvent.click(
      screen.getAllByRole('button').find((el) => el.title === 'Open')!
    );
    expect(screen.getAllByRole('option').length).toBe(3);

    backupMocks.useDbBackups.mockClear();
  });

  // This is a specific case of the previous test
  it('should show only occupied storages when PG took 3 slots from backups', async () => {
    backupMocks.useDbBackups.mockReturnValue({
      data: [
        {
          backupStorageName: 'storage1',
        },
        {
          backupStorageName: 'storage2',
        },
        {
          backupStorageName: 'storage3',
        },
      ],
      isFetching: false,
    });
    render(
      <FormProviderWrapper>
        <QueryClientProvider client={queryClient}>
          <BackupStoragesInput
            namespace="test"
            dbType={DbType.Postresql}
            schedules={[]}
          />
        </QueryClientProvider>
      </FormProviderWrapper>
    );

    fireEvent.click(
      screen.getAllByRole('button').find((el) => el.title === 'Open')!
    );
    expect(screen.getAllByRole('option').length).toBe(3);

    backupMocks.useDbBackups.mockClear();
  });

  it('should hide used storages in schedules when required', async () => {
    backupMocks.useDbBackups.mockReturnValue({
      data: [
        {
          backupStorageName: 'storage1',
        },
        {
          backupStorageName: 'storage2',
        },
        {
          backupStorageName: 'storage3',
        },
        {
          backupStorageName: 'storage4',
        },
      ],
      isFetching: false,
    });
    render(
      <FormProviderWrapper>
        <QueryClientProvider client={queryClient}>
          <BackupStoragesInput
            namespace="test"
            hideUsedStoragesInSchedules
            dbType={DbType.Postresql}
            schedules={[
              {
                name: 'schedule1',
                schedule: '0 0 * * *',
                backupStorageName: 'storage1',
                enabled: true,
              },
              {
                name: 'schedule2',
                schedule: '0 0 * * *',
                backupStorageName: 'storage2',
                enabled: true,
              },
            ]}
          />
        </QueryClientProvider>
      </FormProviderWrapper>
    );

    fireEvent.click(
      screen.getAllByRole('button').find((el) => el.title === 'Open')!
    );
    expect(screen.getAllByRole('option').length).toBe(2);

    backupMocks.useDbBackups.mockClear();
  });
});
