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

import { render, screen, fireEvent } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { FormProvider, useForm } from 'react-hook-form';
import BackupStoragesInput from '.';

const queryClient = new QueryClient();

vi.mock('hooks/api/backup-storages/useBackupStorages', () => ({
  useBackupStoragesByNamespace: () => ({
    data: [
      { name: 'storage1' },
      { name: 'storage2' },
      { name: 'storage3' },
      { name: 'storage4' },
    ],
    isFetching: false,
  }),
}));

const FormProviderWrapper = ({ children }: { children: React.ReactNode }) => {
  const methods = useForm({
    defaultValues: {
      storageLocation: null,
    },
  });
  return (
    <FormProvider {...methods}>
      <form>{children}</form>
    </FormProvider>
  );
};

// --- Unit tests for getAvailableStorages utility ---

// --- Component integration tests ---

describe('BackupStoragesInput (on-demand context)', () => {
  it(
    'shows all storages and helper text when limit > active',
    { timeout: 30000 },
    () => {
      render(
        <FormProviderWrapper>
          <QueryClientProvider client={queryClient}>
            <BackupStoragesInput
              namespace="test"
              schedules={[]}
              maxStorages={2}
              instanceStorageNames={['storage1']}
            />
          </QueryClientProvider>
        </FormProviderWrapper>
      );

      expect(
        screen.getByText(
          'You are currently using 1 out of 2 available storages.'
        )
      ).toBeInTheDocument();

      fireEvent.click(
        screen.getAllByRole('button').find((el) => el.title === 'Open')!
      );
      expect(screen.getAllByRole('option').length).toBe(4);
    }
  );

  it('shows "(in use)" highlight for instance storages when limit not reached', () => {
    render(
      <FormProviderWrapper>
        <QueryClientProvider client={queryClient}>
          <BackupStoragesInput
            namespace="test"
            schedules={[]}
            maxStorages={3}
            instanceStorageNames={['storage1', 'storage2']}
          />
        </QueryClientProvider>
      </FormProviderWrapper>
    );

    fireEvent.click(
      screen.getAllByRole('button').find((el) => el.title === 'Open')!
    );
    // All 4 storages shown
    expect(screen.getAllByRole('option').length).toBe(4);
    // In-use labels shown for storage1 and storage2
    expect(screen.getAllByText('(in use)').length).toBe(2);
  });

  it('shows only instance storages when limit == active > 1', () => {
    render(
      <FormProviderWrapper>
        <QueryClientProvider client={queryClient}>
          <BackupStoragesInput
            namespace="test"
            schedules={[]}
            maxStorages={2}
            instanceStorageNames={['storage1', 'storage3']}
          />
        </QueryClientProvider>
      </FormProviderWrapper>
    );

    expect(
      screen.getByText(
        'Storage limit reached. You are using 2 out of 2 available storages.'
      )
    ).toBeInTheDocument();

    fireEvent.click(
      screen.getAllByRole('button').find((el) => el.title === 'Open')!
    );
    expect(screen.getAllByRole('option').length).toBe(2);
    // No "(in use)" labels when showing restricted list
    expect(screen.queryByText('(in use)')).not.toBeInTheDocument();
  });

  it('disables field and prefills when limit == active == 1', () => {
    render(
      <FormProviderWrapper>
        <QueryClientProvider client={queryClient}>
          <BackupStoragesInput
            namespace="test"
            schedules={[]}
            maxStorages={1}
            instanceStorageNames={['storage2']}
          />
        </QueryClientProvider>
      </FormProviderWrapper>
    );

    expect(
      screen.getByText(
        'Storage limit reached. You are using 1 out of 1 available storages.'
      )
    ).toBeInTheDocument();

    // Field should be disabled
    const input = screen.getByRole('combobox');
    expect(input).toBeDisabled();
  });

  it('shows all storages when active == 0 and limit == 1', () => {
    render(
      <FormProviderWrapper>
        <QueryClientProvider client={queryClient}>
          <BackupStoragesInput
            namespace="test"
            schedules={[]}
            maxStorages={1}
            instanceStorageNames={[]}
          />
        </QueryClientProvider>
      </FormProviderWrapper>
    );

    expect(
      screen.getByText('You are currently using 0 out of 1 available storages.')
    ).toBeInTheDocument();

    fireEvent.click(
      screen.getAllByRole('button').find((el) => el.title === 'Open')!
    );
    expect(screen.getAllByRole('option').length).toBe(4);
  });

  it('shows all storages with no helper text when maxStorages is undefined', () => {
    render(
      <FormProviderWrapper>
        <QueryClientProvider client={queryClient}>
          <BackupStoragesInput
            namespace="test"
            schedules={[]}
            instanceStorageNames={['storage1']}
          />
        </QueryClientProvider>
      </FormProviderWrapper>
    );

    expect(screen.queryByText(/available storages/)).not.toBeInTheDocument();

    fireEvent.click(
      screen.getAllByRole('button').find((el) => el.title === 'Open')!
    );
    expect(screen.getAllByRole('option').length).toBe(4);
  });
});
