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
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { DbClusterDetails } from './db-cluster-details';
import { DbInstanceContext } from './dbCluster.context';
import type { DbInstanceContextProps } from './dbCluster.context.types';
import type { Instance } from 'shared-types/api.types';

vi.mock('components/db-actions/db-actions', () => ({
  default: () => <div data-testid="db-actions" />,
}));

const mockInstance: Instance = {
  apiVersion: 'core.openeverest.io/v1alpha1',
  kind: 'Instance',
  metadata: { name: 'my-test-db' } as unknown as Record<string, never>,
  spec: {
    provider: 'test-provider',
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
});
