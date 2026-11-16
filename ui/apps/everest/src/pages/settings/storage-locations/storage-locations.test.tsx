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

import {
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { TestWrapper } from 'utils/test';
import { Messages as StatusMessages } from 'utils/backup-storages.messages';
import { StorageLocations } from './storage-locations';

const mocks = vi.hoisted(() => ({ deleteStorage: vi.fn() }));

// Update and delete are granted per storage, not on whole namespaces.
vi.mock('hooks/rbac', () => {
  const editable = { canUpdate: true, canDelete: true };
  const grants: Record<string, object> = {
    'team-a/s3-prod': editable,
    'team-a/s3-staging': { canDelete: true },
    'team-a/s3-old': editable,
  };
  const none = {};
  const classes = { canRead: true };
  const namespacePermissions = { canCreate: ['team-a'] };
  return {
    useRBACPermissions: (resource: string, object?: string) =>
      resource === 'backup-storage-classes'
        ? classes
        : (object && grants[object]) || none,
    useNamespacePermissionsForResource: () => namespacePermissions,
  };
});

vi.mock('hooks/api/backup-storage-classes', () => {
  const classes = {
    data: [
      {
        metadata: { name: 's3' },
        spec: {
          displayName: 'S3-compatible object storage',
          authModes: [{ name: 'AccessKey' }],
        },
      },
    ],
  };
  return { useBackupStorageClasses: () => classes };
});

vi.mock('hooks/api/backup-storages', () => {
  const accepted = (status: 'True' | 'False', reason: string) => ({
    type: 'Accepted',
    status,
    reason,
    message: '',
    lastTransitionTime: '',
  });
  const storage = (
    namespace: string,
    name: string,
    conditions: ReturnType<typeof accepted>[],
    extra: object = {}
  ) => ({
    metadata: { name, namespace },
    spec: {
      backupStorageClassRef: { name: 's3' },
      authentication: {
        mode: 'AccessKey',
        credentialsSecretRef: { name: `${name}-credentials` },
      },
      location: { objectStore: { bucket: `${name}-bucket` } },
      ...extra,
    },
    status: { conditions },
  });
  const results = [
    {
      data: [
        storage('team-a', 's3-prod', [accepted('True', 'Accepted')], {
          location: { objectStore: { bucket: 'backups', prefix: 'prod' } },
          parameters: {
            endpointURL: 'https://minio.example.com',
            forcePathStyle: true,
          },
          tls: { insecureSkipVerify: true },
        }),
        storage('team-a', 's3-staging', [
          accepted('False', 'CredentialsNotFound'),
        ]),
        storage('team-a', 's3-old', [
          accepted('True', 'Accepted'),
          {
            type: 'Deleting',
            status: 'True',
            reason: 'WaitingForReferrers',
            message: '',
            lastTransitionTime: '',
          },
        ]),
      ],
      isLoading: false,
    },
    {
      data: [storage('team-b', 's3-shared', [accepted('True', 'Accepted')])],
      isLoading: false,
    },
  ];
  const deletion = { mutate: mocks.deleteStorage, isPending: false };
  return {
    useBackupStorages: () => results,
    useDeleteBackupStorage: () => deletion,
  };
});

const renderPage = () =>
  render(
    <TestWrapper>
      <StorageLocations />
    </TestWrapper>
  );

const getRow = async (name: string) => {
  const cell = await screen.findByText(name);
  const row = cell.closest('tr');
  if (!row) {
    throw new Error(`no row for ${name}`);
  }
  return within(row);
};

describe('StorageLocations', () => {
  beforeEach(() => {
    mocks.deleteStorage.mockReset();
  });

  it('lists each storage with its type, location and status', async () => {
    renderPage();

    const prod = await getRow('s3-prod');
    expect(prod.getByText('S3-compatible object storage')).toBeInTheDocument();
    expect(prod.getByText('backups/prod')).toBeInTheDocument();
    expect(prod.getByText('Valid')).toBeInTheDocument();

    const staging = await getRow('s3-staging');
    expect(staging.getByText('Not valid')).toBeInTheDocument();
    expect(
      staging.getByText(StatusMessages.reasons.CredentialsNotFound)
    ).toBeInTheDocument();

    const old = await getRow('s3-old');
    expect(old.getByText('Deleting')).toBeInTheDocument();
  });

  it('shows how a storage connects once expanded', async () => {
    renderPage();
    await screen.findByText('s3-prod');

    fireEvent.click(screen.getAllByRole('button', { name: 'Expand' })[0]);

    expect(await screen.findByText('s3-prod-credentials')).toBeInTheDocument();
    expect(
      screen.getByText('Certificate verification skipped')
    ).toBeInTheDocument();
    expect(screen.getByText('https://minio.example.com')).toBeInTheDocument();
  });

  it('offers each storage only the actions the user may take on it', async () => {
    renderPage();

    expect(
      (await getRow('s3-old')).queryByTestId('row-actions-menu-button')
    ).not.toBeInTheDocument();
    expect(
      (await getRow('s3-shared')).queryByTestId('row-actions-menu-button')
    ).not.toBeInTheDocument();

    fireEvent.click(
      (await getRow('s3-staging')).getByTestId('row-actions-menu-button')
    );
    expect(
      screen.getByRole('menuitem', { name: /Delete/ })
    ).toBeInTheDocument();
    expect(
      screen.queryByRole('menuitem', { name: /Edit/ })
    ).not.toBeInTheDocument();
    fireEvent.keyDown(screen.getByRole('menu'), { key: 'Escape' });

    fireEvent.click(
      (await getRow('s3-prod')).getByTestId('row-actions-menu-button')
    );
    expect(
      await screen.findByRole('menuitem', { name: /Edit/ })
    ).toBeInTheDocument();
  });

  it('deletes a storage once the user confirms, saying it waits for its users', async () => {
    renderPage();

    fireEvent.click(
      (await getRow('s3-prod')).getByTestId('row-actions-menu-button')
    );
    fireEvent.click(screen.getByRole('menuitem', { name: /Delete/ }));
    expect(
      await screen.findByText(/shows as Deleting until then/)
    ).toBeInTheDocument();
    fireEvent.click(screen.getByTestId('confirm-dialog-delete'));

    await waitFor(() =>
      expect(mocks.deleteStorage).toHaveBeenCalledWith(
        { namespace: 'team-a', name: 's3-prod' },
        expect.anything()
      )
    );
  });
});
