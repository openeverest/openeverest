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
import { InstanceCondition } from 'shared-types/instance.types';
import { getFixtureInstance } from 'utils/tests/instance-status-fixtures';

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
    providerRef: { name: 'test-provider' },
    topology: { type: 'ha' },
  },
  status: getFixtureInstance('healthy').status,
};

const connectionDetails = (
  status: InstanceCondition['status']
): InstanceCondition => ({
  type: 'ConnectionDetailsReady',
  status,
  reason: status === 'True' ? 'Published' : 'NotReported',
  message: '',
  lastTransitionTime: '2026-03-01T11:00:00Z',
});

// The database in the state a status fixture describes, with or without
// its connection details published.
const instanceIn = (
  fixture: string,
  published: InstanceCondition['status']
): Instance => {
  const { status } = getFixtureInstance(fixture);

  return {
    ...mockInstance,
    status: {
      ...status,
      conditions: [...(status?.conditions ?? []), connectionDetails(published)],
    },
  };
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
    isCredentialsPermissionLoading: false,
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

  it('loads credentials once the connection details are published', () => {
    renderOverview('my-ns', 'my-test-db', {
      instance: instanceIn('healthy', 'True'),
    });

    expect(useDbInstanceCredentials).toHaveBeenCalledWith(
      'my-test-db',
      'my-ns',
      expect.objectContaining({
        enabled: true,
      })
    );
  });

  it('keeps the credentials while a change rolls out', () => {
    renderOverview('my-ns', 'my-test-db', {
      instance: instanceIn('rolling-upgrade', 'True'),
    });

    expect(useDbInstanceCredentials).toHaveBeenCalledWith(
      'my-test-db',
      'my-ns',
      expect.objectContaining({
        enabled: true,
      })
    );
  });

  it.each([
    ['are not published yet', instanceIn('creating', 'False')],
    ['are not reported at all', mockInstance],
  ])(
    'does not load credentials while the connection details %s',
    (_, instance) => {
      renderOverview('my-ns', 'my-test-db', { instance });

      expect(useDbInstanceCredentials).toHaveBeenCalledWith(
        'my-test-db',
        'my-ns',
        expect.objectContaining({
          enabled: false,
        })
      );
    }
  );

  it('opens with the status of the database', () => {
    renderOverview('my-ns', 'my-test-db', {
      instance: instanceIn('rolling-upgrade', 'True'),
    });

    expect(screen.getByTestId('status-card-status')).toHaveTextContent(
      'Up \u00b7 Upgrading'
    );
  });

  it.each([
    ['may be edited', 'healthy', true],
    // Converging does not lock the database: the next change queues up.
    ['may be edited', 'rolling-upgrade', true],
    ['may not be edited', 'deleting', false],
  ])('%s when %s', (_, fixture, editable) => {
    vi.mocked(useProviders).mockReturnValue({
      data: [
        {
          metadata: { name: 'test-provider' },
          spec: {
            uiSchema: {
              ha: {
                sections: {
                  resources: {
                    label: 'Resources',
                    components: {
                      replicas: {
                        uiType: FieldType.Number,
                        path: 'spec.components.engine.replicas',
                        fieldParams: { label: 'Nodes' },
                      },
                    },
                  },
                },
                sectionsOrder: ['resources'],
              },
            },
          },
        },
      ],
      isLoading: false,
    } as unknown as ReturnType<typeof useProviders>);

    renderOverview('my-ns', 'my-test-db', {
      instance: {
        ...getFixtureInstance(fixture),
        spec: mockInstance.spec,
      },
    });

    expect(screen.getByTestId('resources-details')).toBeInTheDocument();
    expect(screen.queryAllByTestId('resources-edit-button')).toHaveLength(
      editable ? 1 : 0
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
