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
import { AxiosError, AxiosHeaders } from 'axios';
import { TestWrapper } from 'utils/test';
import { MonitoringDestinations } from './monitoring-destinations';

const mocks = vi.hoisted(() => ({
  fetchBindings: vi.fn(),
  deleteDestination: vi.fn(),
  canDelete: vi.fn(),
  // The destination the page lists; tests set how many Instances core counts.
  destinationStatus: {
    serverVersion: '3.1.0',
    attachedInstances: 1,
    conditions: [
      { type: 'Ready', status: 'True', reason: 'Ready', message: '' },
    ],
  },
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
          metadata: {
            name: 'pmm-prod',
            namespace: 'team-a',
            annotations: {
              'monitoring.openeverest.io/ui-created-credentials-secret':
                'monitoring-destination-pmm-prod-credentials',
            },
          },
          spec: {
            classRef: { name: 'pmm' },
            authentication: {
              mode: 'ServiceAccountToken',
              credentialsSecretRef: {
                name: 'monitoring-destination-pmm-prod-credentials',
              },
            },
            parameters: { url: 'https://pmm.example.com' },
          },
          status: mocks.destinationStatus,
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
    useCanDeleteMonitoringDestination: () => mocks.canDelete,
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

const inUse = new AxiosError('in use', '409', undefined, undefined, {
  status: 409,
  statusText: 'Conflict',
  data: { reason: 'InUse', message: 'in use by 1 databases' },
  headers: {},
  config: { headers: new AxiosHeaders() },
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
    mocks.canDelete.mockReset();
    mocks.destinationStatus.attachedInstances = 1;
  });

  it('lists each destination with its type, address, status and users', async () => {
    renderPage();

    expect(await screen.findByText('pmm-prod')).toBeInTheDocument();
    expect(screen.getByText('PMM')).toBeInTheDocument();
    expect(screen.getByText('https://pmm.example.com')).toBeInTheDocument();
    expect(screen.getByText('Ready · v3.1.0')).toBeInTheDocument();
    expect(screen.getByText('1 instance')).toBeInTheDocument();
  });

  it('lists the instances using a destination in a dialog', async () => {
    renderPage();

    fireEvent.click(await screen.findByTestId('used-by-pmm-prod-toggle'));

    const dialog = await screen.findByRole('dialog');
    expect(dialog).toHaveTextContent('Instances using pmm-prod');
    expect(dialog).toHaveTextContent('orders-db');
  });

  it('names a few users of a busy destination and counts the rest', async () => {
    mocks.canDelete.mockResolvedValue(false);
    mocks.fetchBindings.mockResolvedValue(
      ['a', 'b', 'c', 'd', 'e', 'f', 'g'].map((name) => binding(`${name}-db`))
    );
    renderPage();
    await clickDelete();

    expect(
      await screen.findByText(
        'pmm-prod is used by a-db, b-db, c-db, d-db, e-db and 2 more. Remove it from those instances first.'
      )
    ).toBeInTheDocument();
  });

  it('refuses to delete a destination the server reports in use', async () => {
    mocks.canDelete.mockResolvedValue(false);
    mocks.fetchBindings.mockResolvedValue([
      binding('orders-db'),
      binding('billing-db'),
    ]);
    renderPage();
    await clickDelete();

    expect(
      await screen.findByText(
        'pmm-prod is used by orders-db, billing-db. Remove it from those instances first.'
      )
    ).toBeInTheDocument();
    expect(mocks.canDelete).toHaveBeenCalledWith('team-a', 'pmm-prod');
    expect(
      screen.queryByRole('button', { name: 'Delete' })
    ).not.toBeInTheDocument();
  });

  it("counts users the user can't see from core's count", async () => {
    mocks.destinationStatus.attachedInstances = 4;
    mocks.canDelete.mockResolvedValue(false);
    mocks.fetchBindings.mockResolvedValue([binding('orders-db')]);
    renderPage();
    await clickDelete();

    expect(
      await screen.findByText(
        'pmm-prod is used by orders-db and 3 more. Remove it from those instances first.'
      )
    ).toBeInTheDocument();
  });

  it('deletes an unused destination with the Secret created for it', async () => {
    mocks.canDelete.mockResolvedValue(true);
    renderPage();
    await clickDelete();

    expect(
      await screen.findByText(
        /Its credentials Secret monitoring-destination-pmm-prod-credentials is deleted with it/
      )
    ).toBeInTheDocument();
    expect(mocks.fetchBindings).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole('button', { name: 'Delete' }));

    await waitFor(() =>
      expect(mocks.deleteDestination).toHaveBeenCalledWith(
        expect.objectContaining({
          namespace: 'team-a',
          destination: expect.objectContaining({
            metadata: expect.objectContaining({
              name: 'pmm-prod',
              namespace: 'team-a',
            }),
          }),
        }),
        expect.anything()
      )
    );
  });

  it('explains a refusal when the destination came into use after the dry run', async () => {
    mocks.canDelete.mockResolvedValueOnce(true).mockResolvedValue(false);
    mocks.fetchBindings.mockResolvedValue([binding('orders-db')]);
    mocks.deleteDestination.mockImplementation(
      (_variables, { onError }: { onError: (error: unknown) => void }) =>
        Promise.resolve().then(() => onError(inUse))
    );
    renderPage();
    await clickDelete();
    fireEvent.click(await screen.findByTestId('confirm-dialog-delete'));

    expect(
      await screen.findByText(
        'pmm-prod is used by orders-db. Remove it from those instances first.'
      )
    ).toBeInTheDocument();
  });
});
