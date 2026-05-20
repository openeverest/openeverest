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

import { render, screen } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { OnDemandBackupModal } from './on-demand-backup-modal';
import { ScheduleModalContext } from '../backups.context';
import { Instance } from 'shared-types/api.types';
import { FormMode } from 'components/ui-generator/ui-generator.types';
import { ScheduleWizardMode } from 'shared-types/wizard.types';

const mockCreateBackup = vi.fn();
const mockUpdateInstance = vi.fn();
let mockLoadingStorages = false;
let mockLoadingClasses = false;

vi.mock('hooks/api/useClusterName.ts', () => ({
  useClusterName: () => 'main',
}));

vi.mock('hooks/api/backups/useBackups.ts', () => ({
  useBackupsList: () => ({ data: [] }),
  useCreateBackupOnDemand: () => ({
    mutate: mockCreateBackup,
    isPending: false,
  }),
  getBackupListQueryKey: () => ['backups'],
}));

vi.mock('hooks/api/backup-classes/useBackupClasses.ts', () => ({
  useBackupClassesList: () => ({
    data: [
      {
        metadata: { name: 'standard-class' },
        spec: {
          displayName: 'Standard',
          supportedProviders: [],
          providerManaged: { limits: { maxStorages: 3 } },
        },
      },
    ],
    isLoading: mockLoadingClasses,
  }),
  useBackupClassUiSchema: () => ({ sections: undefined, uiSchema: undefined }),
}));

vi.mock('hooks/api/backup-storages/useBackupStorages.ts', () => ({
  useBackupStoragesByNamespace: () => ({
    data: [{ name: 'storage-1' }, { name: 'storage-2' }],
    isFetching: false,
    isLoading: mockLoadingStorages,
  }),
}));

vi.mock('hooks/api/db-instances/useUpdateDbInstance.ts', () => ({
  useUpdateDbInstanceWithConflictRetry: () => ({
    mutate: mockUpdateInstance,
    isPending: false,
  }),
}));

// Mock the fields wrapper to avoid deep dependency chains
vi.mock('./on-demand-backup-fields-wrapper.tsx', () => ({
  OnDemandBackupFieldsWrapper: () => (
    <div data-testid="fields-wrapper">Fields</div>
  ),
}));

const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: false } },
});

const mockInstance: Instance = {
  apiVersion: 'core.openeverest.io/v1alpha1',
  kind: 'Instance',
  metadata: { name: 'my-instance' } as never,
  spec: {
    provider: 'test-provider',
    backup: {
      enabled: true,
      classRef: { name: 'standard-class' },
      storages: [
        {
          name: 'storage-1',
          storageRef: { name: 'storage-1' },
          main: true,
          schedules: [],
        },
      ],
    },
  },
} as unknown as Instance;

const renderModal = (contextOverrides = {}) => {
  const contextValue = {
    instance: mockInstance,
    openOnDemandModal: true,
    setOpenOnDemandModal: vi.fn(),
    openScheduleModal: false,
    setOpenScheduleModal: vi.fn(),
    mode: FormMode.New as ScheduleWizardMode,
    setMode: vi.fn(),
    selectedScheduleName: '',
    setSelectedScheduleName: vi.fn(),
    ...contextOverrides,
  };

  return {
    ...render(
      <MemoryRouter initialEntries={['/databases/test-ns/my-instance/backups']}>
        <Routes>
          <Route
            path="/databases/:namespace/:instanceName/backups"
            element={
              <QueryClientProvider client={queryClient}>
                <ScheduleModalContext.Provider value={contextValue}>
                  <OnDemandBackupModal />
                </ScheduleModalContext.Provider>
              </QueryClientProvider>
            }
          />
        </Routes>
      </MemoryRouter>
    ),
    contextValue,
  };
};

describe('OnDemandBackupModal', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockLoadingStorages = false;
    mockLoadingClasses = false;
    queryClient.clear();
  });

  it(
    'renders form with header and submit button when data is ready',
    { timeout: 15000 },
    async () => {
      renderModal();

      expect(screen.getByText('Create on-demand backup')).toBeInTheDocument();
      expect(screen.getByText('Fields')).toBeInTheDocument();
      expect(
        screen.getByText(
          'Select a backup class and storage to create an on-demand backup.'
        )
      ).toBeInTheDocument();

      expect(
        screen.getByRole('button', { name: /create/i })
      ).toBeInTheDocument();
    }
  );

  it('shows loading spinner when backup storages are loading', async () => {
    mockLoadingStorages = true;

    renderModal();

    expect(screen.getByRole('progressbar')).toBeInTheDocument();
    expect(screen.queryByText('Fields')).not.toBeInTheDocument();
  });

  it('shows loading spinner when backup classes are loading', async () => {
    mockLoadingClasses = true;

    renderModal();

    expect(screen.getByRole('progressbar')).toBeInTheDocument();
    expect(screen.queryByText('Fields')).not.toBeInTheDocument();
  });
});
