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

import { fireEvent, render, screen } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { DbClusterDetails } from './db-cluster-details';
import { DbInstanceContext } from './dbCluster.context';
import type { DbInstanceContextProps } from './dbCluster.context.types';
import type { Instance } from 'shared-types/api.types';

vi.mock('components/db-actions/db-actions', () => ({
  default: () => <div data-testid="db-actions" />,
}));

vi.mock('contexts/plugins', () => ({
  usePlugins: () => ({ plugins: [] }),
}));

const mockInstance: Instance = {
  apiVersion: 'core.openeverest.io/v1alpha1',
  kind: 'Instance',
  metadata: { name: 'my-test-db' } as unknown as Record<string, never>,
  spec: {
    providerRef: { name: 'test-provider' },
    topology: { type: 'ha' },
  },
  status: { phase: 'Ready' },
};

const renderDetails = (contextValue?: Partial<DbInstanceContextProps>) => {
  const value: DbInstanceContextProps = {
    instance: mockInstance,
    isLoading: false,
    instanceDeleted: false,
    canReadCredentials: true,
    ...contextValue,
  };

  return render(
    <MemoryRouter initialEntries={['/databases/test-ns/my-test-db/overview']}>
      <Routes>
        <Route
          path="/databases/:namespace/:instanceName/:tabs"
          element={
            <DbInstanceContext.Provider value={value}>
              <DbClusterDetails />
            </DbInstanceContext.Provider>
          }
        >
          <Route path="overview" element={<div>Overview content</div>} />
        </Route>
      </Routes>
    </MemoryRouter>
  );
};

describe('DbClusterDetails', () => {
  it('renders the current db instance phase in the page header', () => {
    renderDetails({
      instance: {
        ...mockInstance,
        status: { phase: 'Restoring' },
      },
    });

    expect(screen.getByText('Restoring')).toBeInTheDocument();
  });

  it('falls back to unknown status when phase is missing', () => {
    renderDetails({
      instance: {
        ...mockInstance,
        status: undefined,
      },
    });

    expect(screen.getByText('Unknown')).toBeInTheDocument();
  });

  it('warns about pods the scheduler cannot place, quoting the reason', () => {
    const message =
      "engine: 1 of 3 pods cannot be scheduled: 0/2 nodes are available: 2 node(s) didn't match pod anti-affinity rules.";
    renderDetails({
      instance: {
        ...mockInstance,
        status: {
          phase: 'Initializing',
          conditions: [
            {
              type: 'PodsScheduled',
              status: 'False',
              reason: 'Unschedulable',
              message,
              lastTransitionTime: '2026-10-02T12:00:00Z',
            },
          ],
        },
      },
    });

    expect(
      screen.getByText('Some pods cannot be scheduled')
    ).toBeInTheDocument();
    expect(screen.queryByText(message)).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Show details' }));

    expect(screen.getByText(message)).toBeInTheDocument();
  });

  it('shows no scheduling warning once every pod has a node', () => {
    renderDetails({
      instance: {
        ...mockInstance,
        status: {
          phase: 'Ready',
          conditions: [
            {
              type: 'PodsScheduled',
              status: 'True',
              reason: 'Scheduled',
              message: '',
              lastTransitionTime: '2026-10-02T12:00:00Z',
            },
          ],
        },
      },
    });

    expect(screen.queryByTestId('pods-alert')).not.toBeInTheDocument();
  });

  it('warns about crashing pods, quoting the reason', () => {
    const message =
      'engine: 1 of 3 pods keep crashing: container mongod last exited with OOMKilled (exit code 137)';
    renderDetails({
      instance: {
        ...mockInstance,
        status: {
          phase: 'Initializing',
          conditions: [
            {
              type: 'PodsReady',
              status: 'False',
              reason: 'CrashLoopBackOff',
              message,
              lastTransitionTime: '2026-10-02T12:00:00Z',
            },
          ],
        },
      },
    });

    expect(screen.getByText('Some pods keep crashing')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Show details' }));
    expect(screen.getByText(message)).toBeInTheDocument();
  });

  it('does not warn about pods that are only starting', () => {
    renderDetails({
      instance: {
        ...mockInstance,
        status: {
          phase: 'Initializing',
          conditions: [
            {
              type: 'PodsReady',
              status: 'False',
              reason: 'NotReady',
              message: 'engine: 1 of 3 pods are not ready',
              lastTransitionTime: '2026-10-02T12:00:00Z',
            },
          ],
        },
      },
    });

    expect(screen.queryByTestId('pods-alert')).not.toBeInTheDocument();
  });
});
