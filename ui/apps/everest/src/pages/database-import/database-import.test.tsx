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

import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { TestWrapper } from 'utils/test';
import { buildImportNavigationRouteState } from 'pages/database-form/import-navigation-state';
import { CreateEditModalStorageProps } from 'pages/settings/storage-locations/createEditModal/create-edit-modal.types';
import { Provider } from 'shared-types/api.types';
import { Backup, BackupClass, BackupImport } from 'shared-types/backups.types';
import {
  BackupStorageCRD,
  BackupStorageFormValues,
  StorageType,
} from 'shared-types/backupStorages.types';
import { DatabaseImportPage } from './database-import';
import { Messages } from './database-import.messages';

const mocks = vi.hoisted(() => ({
  navigate: vi.fn(),
  location: { state: undefined as unknown },
}));

const PROVIDERS: Provider[] = [{ metadata: { name: 'psmdb' }, spec: {} }];
const CLASSES: BackupClass[] = [
  {
    metadata: { name: 'psmdb-pbm' },
    spec: {
      executionMode: 'ProviderManaged',
      supportedProviders: ['psmdb'],
      supportsImport: true,
    },
  },
];
const storage = (name: string): BackupStorageCRD => ({
  metadata: { name },
  spec: {
    type: 's3',
    s3: {
      bucket: name,
      region: 'us-east-1',
      endpointURL: 'https://s3.example.com',
      credentialsSecretRef: { name: 'creds' },
    },
  },
});
const STORAGES_BY_NAMESPACE: Record<string, BackupStorageCRD[]> = {
  'ns-a': [storage('s3-old')],
  'ns-b': [storage('s3-new')],
  'ns-empty': [],
};
const ADDED_STORAGE: BackupStorageFormValues = {
  name: 's3-added',
  namespace: 'ns-a',
  type: StorageType.S3,
  bucketName: 's3-added',
  url: 'https://s3.example.com',
  region: 'us-east-1',
  accessKey: 'key',
  secretKey: 'secret',
  verifyTLS: true,
  forcePathStyle: false,
};
const externalBackup = (name: string, path: string): Backup => ({
  metadata: { name },
  spec: {
    classRef: { name: 'psmdb-pbm' },
    storageRef: { name: 's3-old' },
    origin: {
      type: 'External',
      external: {
        path,
        startedAt: '2026-10-01T10:00:00Z',
        completedAt: '2026-10-01T10:05:00Z',
      },
    },
  },
});
const BACKUPS = [externalBackup('import-abc12-1', 'prod/2026-10-01')];
const OTHER_BACKUPS = [externalBackup('import-abc12-2', 'prod/2026-10-02')];
const NO_IMPORTS: BackupImport[] = [];
let importNamespaces = ['ns-a'];
let namespaceBackups = BACKUPS;

vi.mock('react-router-dom', async (importOriginal) => ({
  ...(await importOriginal<typeof import('react-router-dom')>()),
  useNavigate: () => mocks.navigate,
  useLocation: () => mocks.location,
}));
vi.mock('hooks/api/useClusterName', () => ({ useClusterName: () => 'main' }));
vi.mock('hooks/rbac', () => ({
  useRBACPermissions: () => ({ canCreate: true }),
}));
vi.mock('hooks/api/providers/useProviders', () => ({
  useProviders: () => ({ data: PROVIDERS }),
}));
vi.mock('hooks/api/extension-catalog', () => ({
  useExtensionCatalog: () => ({ getProviderMeta: () => undefined }),
}));
vi.mock('hooks/api/backup-classes/useBackupClasses', () => ({
  useImportBackupClasses: () => ({ data: CLASSES, isLoading: false }),
}));
vi.mock('hooks/api/backup-storages/useBackupStorages', () => ({
  BACKUP_STORAGES_QUERY_KEY: 'backupStorages',
  useBackupStoragesByNamespace: (namespace: string) => ({
    data: STORAGES_BY_NAMESPACE[namespace],
    isLoading: false,
    isFetching: false,
  }),
  useCreateBackupStorage: () => ({
    mutate: (
      data: BackupStorageFormValues,
      { onSuccess }: { onSuccess: () => void }
    ) => {
      STORAGES_BY_NAMESPACE[data.namespace].push(storage(data.name));
      onSuccess();
    },
    isPending: false,
  }),
}));
vi.mock(
  'pages/settings/storage-locations/createEditModal/create-edit-modal',
  () => ({
    CreateEditModalStorage: ({
      handleSubmitModal,
    }: CreateEditModalStorageProps) => (
      <button onClick={() => handleSubmitModal(false, ADDED_STORAGE)}>
        Save storage
      </button>
    ),
  })
);
vi.mock('hooks/api/backups/useBackups', () => ({
  getNamespaceBackupsKeyPrefix: () => ['backups'],
  useNamespaceBackups: () => ({ data: namespaceBackups, isLoading: false }),
}));
vi.mock('hooks/api/backup-imports', () => ({
  useImportNamespaces: () => ({
    namespaces: importNamespaces,
    isLoading: false,
  }),
  useBackupImports: () => ({ data: NO_IMPORTS }),
  useCreateBackupImport: () => ({ mutate: vi.fn(), isPending: false }),
}));

const queryClient = new QueryClient();
// A new element each call, so a rerender reaches the page.
const page = () => (
  <QueryClientProvider client={queryClient}>
    <TestWrapper>
      <DatabaseImportPage />
    </TestWrapper>
  </QueryClientProvider>
);

const pickBackup = async (path: string) => {
  fireEvent.click(await screen.findByText(path));
  await waitFor(() => expect(getContinueButton()).toBeEnabled());
};

const getContinueButton = () => screen.getByTestId('db-wizard-continue-button');

describe('DatabaseImportPage', () => {
  beforeEach(() => {
    importNamespaces = ['ns-a'];
    namespaceBackups = BACKUPS;
    STORAGES_BY_NAMESPACE['ns-a'] = [storage('s3-old')];
    mocks.location = { state: undefined };
    mocks.navigate.mockReset();
  });

  it('continues to the wizard with the chosen backup', async () => {
    render(page());
    expect(getContinueButton()).toBeDisabled();

    await pickBackup('prod/2026-10-01');
    fireEvent.click(getContinueButton());

    await waitFor(() =>
      expect(mocks.navigate).toHaveBeenCalledWith('/databases/new', {
        state: {
          selectedDbProvider: PROVIDERS[0],
          importSource: {
            namespace: 'ns-a',
            backupClassName: 'psmdb-pbm',
            dataSource: {
              type: 'Backup',
              backupName: 'import-abc12-1',
              storageName: 's3-old',
            },
          },
        },
      })
    );
  });

  it("doesn't continue with a backup the current list doesn't have", async () => {
    const { rerender } = render(page());
    await pickBackup('prod/2026-10-01');

    namespaceBackups = OTHER_BACKUPS;
    rerender(page());

    await waitFor(() => expect(getContinueButton()).toBeDisabled());
  });

  it("switches to the new namespace's storage", async () => {
    importNamespaces = ['ns-a', 'ns-b'];
    render(page());
    const storageInput = screen.getByTestId('text-input-storage');
    await waitFor(() => expect(storageInput).toHaveValue('s3-old'));

    fireEvent.keyDown(screen.getByTestId('text-input-namespace'), {
      key: 'ArrowDown',
    });
    fireEvent.click(await screen.findByRole('option', { name: 'ns-b' }));

    await waitFor(() =>
      expect(screen.getByTestId('text-input-storage')).toHaveValue('s3-new')
    );
  });

  it('asks to add a storage when the namespace has none', async () => {
    importNamespaces = ['ns-empty'];
    render(page());

    expect(await screen.findByText(Messages.noStorages)).toBeInTheDocument();
    expect(screen.queryByText('prod/2026-10-01')).not.toBeInTheDocument();
    expect(getContinueButton()).toBeDisabled();
  });

  it('selects a storage added from the page', async () => {
    render(page());
    await waitFor(() =>
      expect(screen.getByTestId('text-input-storage')).toHaveValue('s3-old')
    );

    fireEvent.click(screen.getByTestId('add-storage-button'));
    fireEvent.click(screen.getByText('Save storage'));

    await waitFor(() =>
      expect(screen.getByTestId('text-input-storage')).toHaveValue('s3-added')
    );
  });

  it('restores the previous choice when coming back from the wizard', async () => {
    importNamespaces = ['ns-a', 'ns-b'];
    const importSource = {
      namespace: 'ns-b',
      backupClassName: 'psmdb-pbm',
      dataSource: {
        type: 'Backup' as const,
        backupName: 'import-abc12-1',
        storageName: 's3-new',
      },
    };
    mocks.location = {
      state: buildImportNavigationRouteState(PROVIDERS[0], importSource),
    };
    render(page());

    await waitFor(() =>
      expect(screen.getByTestId('text-input-storage')).toHaveValue('s3-new')
    );
    expect(screen.getByTestId('text-input-namespace')).toHaveValue('ns-b');
    await waitFor(() => expect(getContinueButton()).toBeEnabled());

    fireEvent.click(getContinueButton());
    await waitFor(() =>
      expect(mocks.navigate).toHaveBeenCalledWith('/databases/new', {
        state: buildImportNavigationRouteState(PROVIDERS[0], importSource),
      })
    );
  });
});
