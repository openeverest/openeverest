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
import { Instance } from 'shared-types/api.types';
import { TestWrapper } from 'utils/test';
import { MonitoringEditModal } from './monitoring-edit-modal';

const mocks = vi.hoisted(() => ({ mutate: vi.fn() }));

vi.mock('hooks/api/monitoring-classes', () => {
  const classes = {
    data: [
      {
        metadata: { name: 'pmm' },
        spec: { displayName: 'PMM', authModes: [{ name: 'Token' }] },
        status: {
          providers: [
            {
              name: 'percona-server-mongodb',
              mode: 'ExtensionManaged',
              disruption: 'RollingRestart',
            },
          ],
        },
      },
      {
        metadata: { name: 'coroot' },
        spec: { displayName: 'Coroot', authModes: [{ name: 'Anonymous' }] },
        status: {
          providers: [
            {
              name: 'percona-server-mongodb',
              mode: 'ExtensionManaged',
              disruption: 'None',
            },
          ],
        },
      },
    ],
    isLoading: false,
  };
  return { useMonitoringClassesForProvider: () => classes };
});

vi.mock('hooks/api/monitoring-destinations', () => {
  const destinations = {
    data: [
      { metadata: { name: 'pmm-a' }, spec: { classRef: { name: 'pmm' } } },
      { metadata: { name: 'coroot' }, spec: { classRef: { name: 'coroot' } } },
    ],
    isLoading: false,
    isSuccess: true,
  };
  return { useMonitoringDestinations: () => destinations };
});

vi.mock('hooks/api/db-instances/useUpdateDbInstance', () => ({
  useUpdateDbInstanceWithConflictRetry: () => ({ mutate: mocks.mutate }),
}));

const instance: Instance = {
  metadata: { name: 'orders-db', namespace: 'team-a', resourceVersion: '3' },
  spec: {
    providerRef: { name: 'percona-server-mongodb' },
    components: { engine: { replicas: 1 } },
    monitoring: {
      destinations: [
        { name: 'coroot', destinationRef: { name: 'coroot' } },
        // Its destination was deleted; editing must not drop it silently.
        { name: 'gone', destinationRef: { name: 'gone' } },
      ],
    },
  },
};

const getCard = (name: string) =>
  screen.getByRole('button', { name: new RegExp(`^${name}`) });

describe('MonitoringEditModal', () => {
  it('warns about the restart an added destination causes and saves the entries', async () => {
    render(
      <TestWrapper>
        <MonitoringEditModal
          instance={instance}
          namespace="team-a"
          onClose={vi.fn()}
        />
      </TestWrapper>
    );

    expect(
      screen.queryByTestId('monitoring-disruption')
    ).not.toBeInTheDocument();

    fireEvent.click(getCard('pmm-a'));

    expect(screen.getByTestId('monitoring-disruption')).toHaveTextContent(
      'Enabling PMM restarts the database pods one at a time; connections briefly drop.' +
        'This database will be briefly unavailable while it restarts.'
    );

    const save = screen.getByTestId('form-dialog-save');
    await waitFor(() => expect(save).toBeEnabled());
    fireEvent.click(save);

    await waitFor(() => expect(mocks.mutate).toHaveBeenCalled());
    expect(mocks.mutate.mock.calls[0][0]).toEqual({
      ...instance,
      spec: {
        ...instance.spec,
        monitoring: {
          destinations: [
            { name: 'coroot', destinationRef: { name: 'coroot' } },
            { name: 'gone', destinationRef: { name: 'gone' } },
            { name: 'pmm-a', destinationRef: { name: 'pmm-a' } },
          ],
        },
      },
    });
  });
});
