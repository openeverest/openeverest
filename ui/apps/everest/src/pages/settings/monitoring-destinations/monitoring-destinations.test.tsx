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
import { TestWrapper } from 'utils/test';
import { MonitoringDestinations } from './monitoring-destinations';

const mocks = vi.hoisted(() => ({
  fetchBindings: vi.fn(),
  deleteDestination: vi.fn(),
}));

vi.mock('hooks/rbac', () => {
  const namespaces = ['team-a'];
  const permissions = {
    canRead: namespaces,
    canCreate: namespaces,
    canUpdate: namespaces,
    canDelete: namespaces,
    isPermissionsLoading: false,
  };
  const classPermissions = { canRead: true };
  return {
    useNamespacePermissionsForResource: () => permissions,
    useRBACPermissions: () => classPermissions,
  };
});

vi.mock('hooks/api/monitoring-classes', async (importOriginal) => {
  const classes = {
    data: [
      {
        metadata: { name: 'pmm' },
        spec: {
          displayName: 'PMM',
          authModes: [{ name: 'ServiceAccountToken' }],
          uiSchema: {
            destination: {
              components: { url: { path: 'parameters.url', uiType: 'text' } },
            },
          },
        },
      },
    ],
  };
  return {
    ...(await importOriginal<object>()),
    useMonitoringClasses: () => classes,
  };
});

vi.mock('hooks/api/monitoring-destinations', async (importOriginal) => {
  const destinations = [
    {
      data: [
        {
          metadata: { name: 'pmm-prod', namespace: 'team-a' },
          spec: {
            classRef: { name: 'pmm' },
            authentication: {
              mode: 'ServiceAccountToken',
              credentialsSecretRef: { name: 'pmm-prod-credentials' },
            },
            parameters: { url: 'https://pmm.example.com' },
          },
          status: {
            serverVersion: '3.1.0',
            conditions: [
              { type: 'Ready', status: 'True', reason: 'Ready', message: '' },
            ],
          },
        },
      ],
      isLoading: false,
    },
  ];
  const deletion = { mutate: mocks.deleteDestination, isPending: false };
  return {
    ...(await importOriginal<object>()),
    useMonitoringDestinationsForNamespaces: () => destinations,
    useDeleteMonitoringDestination: () => deletion,
  };
});

vi.mock('hooks/api/monitoring-bindings', () => {
  const bindings = [
    {
      data: [
        {
          spec: {
            destinationRef: { name: 'pmm-prod' },
            instanceRef: { name: 'orders-db' },
          },
        },
      ],
    },
  ];
  return {
    useMonitoringBindingsForNamespaces: () => bindings,
    useFetchMonitoringBindings: () => mocks.fetchBindings,
  };
});

const binding = (instance: string) => ({
  spec: {
    destinationRef: { name: 'pmm-prod' },
    instanceRef: { name: instance },
  },
});

const renderPage = () =>
  render(
    <TestWrapper>
      <MonitoringDestinations />
    </TestWrapper>
  );

const clickDelete = async () => {
  fireEvent.click(await screen.findByTestId('row-actions-menu-button'));
  fireEvent.click(screen.getByRole('menuitem', { name: /Delete/ }));
};

describe('MonitoringDestinations', () => {
  beforeEach(() => {
    mocks.fetchBindings.mockReset();
    mocks.deleteDestination.mockReset();
  });

  it('lists each destination with its type, address, status and users', async () => {
    renderPage();

    expect(await screen.findByText('pmm-prod')).toBeInTheDocument();
    expect(screen.getByText('PMM')).toBeInTheDocument();
    expect(screen.getByText('https://pmm.example.com')).toBeInTheDocument();
    expect(screen.getByText('Ready · v3.1.0')).toBeInTheDocument();
    expect(screen.getByText('1 database')).toBeInTheDocument();
  });

  it('lists the databases using a destination in a dialog', async () => {
    renderPage();

    fireEvent.click(await screen.findByTestId('used-by-pmm-prod-toggle'));

    const dialog = await screen.findByRole('dialog');
    expect(dialog).toHaveTextContent('Databases using pmm-prod');
    expect(dialog).toHaveTextContent('orders-db');
  });

  it('names a few users of a busy destination and counts the rest', async () => {
    mocks.fetchBindings.mockResolvedValue(
      ['a', 'b', 'c', 'd', 'e', 'f', 'g'].map((name) => binding(`${name}-db`))
    );
    renderPage();
    await clickDelete();

    expect(
      await screen.findByText(
        'pmm-prod is used by a-db, b-db, c-db, d-db, e-db and 2 more. Remove it from those databases first.'
      )
    ).toBeInTheDocument();
  });

  it('refuses to delete a destination databases still use', async () => {
    mocks.fetchBindings.mockResolvedValue([
      binding('orders-db'),
      binding('billing-db'),
    ]);
    renderPage();
    await clickDelete();

    expect(
      await screen.findByText(
        'pmm-prod is used by orders-db, billing-db. Remove it from those databases first.'
      )
    ).toBeInTheDocument();
    expect(mocks.fetchBindings).toHaveBeenCalledWith('team-a', 'pmm-prod');
    expect(
      screen.queryByRole('button', { name: 'Delete' })
    ).not.toBeInTheDocument();
  });

  it('deletes an unused destination with the Secret created for it', async () => {
    mocks.fetchBindings.mockResolvedValue([]);
    renderPage();
    await clickDelete();

    expect(
      await screen.findByText(
        /Its credentials Secret pmm-prod-credentials is deleted with it/
      )
    ).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Delete' }));

    await waitFor(() =>
      expect(mocks.deleteDestination).toHaveBeenCalledWith(
        expect.objectContaining({
          namespace: 'team-a',
          destination: expect.objectContaining({
            metadata: { name: 'pmm-prod', namespace: 'team-a' },
          }),
        }),
        expect.anything()
      )
    );
  });
});
