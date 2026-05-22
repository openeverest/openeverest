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

import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { FormProvider, useForm } from 'react-hook-form';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { OnDemandBackupFieldsWrapper } from './on-demand-backup-fields-wrapper';
import { ScheduleModalContext } from '../backups.context';
import { BackupFields } from './on-demand-backup-modal.types';
import { Instance } from 'shared-types/api.types';
import { FormMode } from 'components/ui-generator/ui-generator.types';
import { ScheduleWizardMode } from 'shared-types/wizard.types';

const mockBackupClasses = [
  {
    metadata: { name: 'class-limited' },
    spec: {
      displayName: 'Limited Class',
      supportedProviders: [],
      providerManaged: {
        limits: { maxStorages: 1, maxSchedulesPerStorage: 2 },
      },
    },
  },
  {
    metadata: { name: 'class-generous' },
    spec: {
      displayName: 'Generous Class',
      supportedProviders: [],
      providerManaged: {
        limits: { maxStorages: 5, maxSchedulesPerStorage: 10 },
      },
    },
  },
];

vi.mock('hooks/api/backup-classes/useBackupClasses.ts', () => ({
  useBackupClassesList: () => ({
    data: mockBackupClasses,
    isLoading: false,
  }),
  useBackupClassUiSchema: () => ({ sections: undefined, uiSchema: undefined }),
}));

vi.mock('hooks/api/backup-storages/useBackupStorages', () => ({
  useBackupStoragesByNamespace: () => ({
    data: [{ name: 'storage-a' }, { name: 'storage-b' }, { name: 'storage-c' }],
    isFetching: false,
    isLoading: false,
  }),
}));

vi.mock('hooks/api/useClusterName.ts', () => ({
  useClusterName: () => 'main',
}));

const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: false } },
});

const mockInstance: Instance = {
  apiVersion: 'core.openeverest.io/v1alpha1',
  kind: 'Instance',
  metadata: { name: 'test-instance' } as never,
  spec: {
    provider: 'test-provider',
    backup: {
      enabled: true,
      classRef: { name: 'class-limited' },
      storages: [
        {
          name: 'storage-a',
          storageRef: { name: 'storage-a' },
          main: true,
          schedules: [],
        },
      ],
    },
  },
} as unknown as Instance;

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
};

const FormWrapper = ({
  children,
  initialClass = '',
}: {
  children: React.ReactNode;
  initialClass?: string;
}) => {
  const methods = useForm({
    defaultValues: {
      [BackupFields.name]: 'backup-test',
      [BackupFields.backupClassName]: initialClass,
      [BackupFields.storageName]: undefined,
    },
    mode: 'onChange',
  });
  return (
    <FormProvider {...methods}>
      <form>{children}</form>
    </FormProvider>
  );
};

const renderFieldsWrapper = (initialClass?: string) =>
  render(
    <MemoryRouter initialEntries={['/databases/test-ns/test-instance/backups']}>
      <Routes>
        <Route
          path="/databases/:namespace/:instanceName/backups"
          element={
            <QueryClientProvider client={queryClient}>
              <ScheduleModalContext.Provider value={contextValue}>
                <FormWrapper initialClass={initialClass}>
                  <OnDemandBackupFieldsWrapper />
                </FormWrapper>
              </ScheduleModalContext.Provider>
            </QueryClientProvider>
          }
        />
      </Routes>
    </MemoryRouter>
  );

describe('OnDemandBackupFieldsWrapper', () => {
  beforeEach(() => {
    queryClient.clear();
  });

  it('auto-fills backupClassName with the first available class', async () => {
    renderFieldsWrapper();

    await waitFor(() => {
      const select = screen.getByLabelText('Backup class');
      expect(select).toHaveTextContent('Limited Class');
    });
  });

  it(
    'changes storage limits when backup class is changed',
    { timeout: 30000 },
    async () => {
      // Start with class-limited pre-set (maxStorages=1, 1 instance storage)
      renderFieldsWrapper('class-limited');

      // With limit==active==1 → field should be disabled
      await waitFor(() => {
        const storageInput = screen.getByTestId('text-input-storage-name');
        expect(storageInput).toBeDisabled();
      });

      // Change to "class-generous" with maxStorages=5
      const selectButton = screen.getByLabelText('Backup class');
      fireEvent.mouseDown(selectButton);

      const generousOption = await screen.findByText('Generous Class');
      fireEvent.click(generousOption);

      // Now with maxStorages=5 and 1 active storage → field enabled, all options shown
      await waitFor(() => {
        const storageInput = screen.getByTestId('text-input-storage-name');
        expect(storageInput).not.toBeDisabled();
      });

      // Open autocomplete to verify multiple options are available
      const storageInput = screen.getByTestId('text-input-storage-name');
      fireEvent.click(storageInput);
      fireEvent.keyDown(storageInput, { key: 'ArrowDown' });

      await waitFor(() => {
        expect(screen.getAllByRole('option').length).toBe(3);
      });
    }
  );
});
