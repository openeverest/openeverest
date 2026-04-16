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
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { DbInstanceContext } from '../dbCluster.context';
import { DbInstanceContextProps } from '../dbCluster.context.types';
import { ClusterOverview } from './cluster-overview';
import { FieldType } from 'components/ui-generator/ui-generator.types';
import { Instance } from 'shared-types/api.types';

const { useDbInstanceCredentials } = vi.hoisted(() => ({
  useDbInstanceCredentials: vi.fn(() => ({ data: undefined })),
}));

vi.mock('hooks/api/db-instances/useCreateDbInstance', () => ({
  useDbInstanceCredentials,
}));

vi.mock('hooks/api/providers', () => ({
  useProviders: vi.fn(() => ({ data: undefined })),
}));

const { useProviders } = await import('hooks/api/providers');

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

const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: false } },
});

function renderOverview(
  namespace = 'my-ns',
  instanceName = 'my-test-db',
  contextValue?: Partial<DbInstanceContextProps>
) {
  const value: DbInstanceContextProps = {
    instance: mockInstance,
    isLoading: false,
    instanceDeleted: false,
    canReadCredentials: true,
    ...contextValue,
  };

  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter
        initialEntries={[`/databases/${namespace}/${instanceName}/overview`]}
      >
        <Routes>
          <Route
            path="/databases/:namespace/:instanceName/overview"
            element={
              <DbInstanceContext.Provider value={value}>
                <ClusterOverview />
              </DbInstanceContext.Provider>
            }
          />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>
  );
}

describe('ClusterOverview', () => {
  beforeEach(() => {
    useDbInstanceCredentials.mockClear();
  });

  it('renders the database name from instance metadata', () => {
    renderOverview('my-ns', 'my-test-db');
    expect(screen.getByText('my-test-db')).toBeInTheDocument();
  });

  it('renders the namespace from URL params', () => {
    renderOverview('production', 'prod-db');
    expect(screen.getByText('production')).toBeInTheDocument();
  });

  it('renders the overview container', () => {
    renderOverview();
    expect(screen.getByTestId('cluster-overview')).toBeInTheDocument();
  });

  it('shows the basic information section with name and namespace labels', () => {
    renderOverview('staging', 'staging-db', {
      instance: {
        ...mockInstance,
        metadata: { name: 'staging-db' } as unknown as Record<string, never>,
      },
    });
    expect(screen.getByText('Name')).toBeInTheDocument();
    expect(screen.getByText('Namespace')).toBeInTheDocument();
    expect(screen.getByText('staging-db')).toBeInTheDocument();
    expect(screen.getByText('staging')).toBeInTheDocument();
  });

  it('returns null while loading', () => {
    renderOverview('my-ns', 'my-test-db', {
      isLoading: true,
      instance: undefined,
    });
    expect(screen.queryByTestId('cluster-overview')).not.toBeInTheDocument();
  });

  it('returns null when instance is not available', () => {
    renderOverview('my-ns', 'my-test-db', { instance: undefined });
    expect(screen.queryByTestId('cluster-overview')).not.toBeInTheDocument();
  });

  it('polls credentials when instance phase is ready', () => {
    renderOverview('my-ns', 'my-test-db');

    expect(useDbInstanceCredentials).toHaveBeenCalledWith(
      'my-test-db',
      'my-ns',
      expect.objectContaining({
        enabled: true,
      })
    );
  });

  // TODO recheck condition, probably we should show oriented on the plag int he conditions in status
  it('does not query credentials before instance is ready', () => {
    renderOverview('my-ns', 'my-test-db', {
      instance: {
        ...mockInstance,
        status: { phase: 'Initializing' },
      },
    });

    expect(useDbInstanceCredentials).toHaveBeenCalledWith(
      'my-test-db',
      'my-ns',
      expect.objectContaining({
        enabled: false,
      })
    );
  });

  it('renders schema-driven card title only once', () => {
    vi.mocked(useProviders).mockReturnValue({
      data: [
        {
          metadata: { name: 'test-provider' },
          spec: {
            uiSchema: {
              ha: {
                sections: {
                  databaseVersion: {
                    label: 'Database Version',
                    components: {
                      version: {
                        uiType: FieldType.Text,
                        path: 'spec.database.version',
                        fieldParams: { label: 'Database Version' },
                      },
                    },
                  },
                },
                sectionsOrder: ['databaseVersion'],
              },
            },
          },
        },
      ],
      isLoading: false,
    } as unknown as ReturnType<typeof useProviders>);

    renderOverview('qa', 'mongo-db', {
      instance: {
        ...mockInstance,
        spec: {
          ...mockInstance.spec,
          database: { version: '6.0.19-16' },
        } as unknown as Instance['spec'],
      },
    });

    expect(screen.getAllByText('Database Version')).toHaveLength(2);
    expect(
      screen.queryByTestId('databaseVersion-overview-section')
    ).toBeInTheDocument();
    expect(screen.getByText('6.0.19-16')).toBeInTheDocument();
  });
});
