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
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom';
import { ThemeContextProvider, everestThemeOptions } from '@percona/design';
import CreateDbButton from 'components/create-db-button/create-db-button';
import { DatabasePage } from '../database-form';
import { FieldType } from 'components/ui-generator/ui-generator.types';

vi.mock('hooks/api/providers', () => ({
  useProviders: vi.fn(),
}));

vi.mock('hooks/api/instances/useCreateInstance', () => ({
  useCreateInstance: () => ({
    mutate: vi.fn(),
    isPending: false,
  }),
}));

vi.mock('hooks/utils/useActiveBreakpoint', () => ({
  useActiveBreakpoint: () => ({ isDesktop: true }),
}));

vi.mock('hooks', () => ({
  useNamespaces: () => ({ data: [] }),
  useDBClustersForNamespaces: () => ({}),
}));

vi.mock('hooks/index.ts', () => ({
  useNamespaces: () => ({ data: [] }),
  useDBClustersForNamespaces: () => ({}),
}));

vi.mock('hooks/rbac', () => ({
  useNamespacePermissionsForResource: () => ({
    isLoading: false,
    canCreate: ['default'],
  }),
}));

vi.mock('react-router-dom', async () => {
  const actual =
    await vi.importActual<typeof import('react-router-dom')>(
      'react-router-dom'
    );

  return {
    ...actual,
    useBlocker: () => ({
      state: 'unblocked',
      reset: vi.fn(),
      proceed: vi.fn(),
    }),
  };
});

const { useProviders } = await import('hooks/api/providers');

const providerWithMultipathSchema = {
  metadata: { name: 'percona-server-mongodb' },
  spec: {
    uiSchema: {
      replica: {
        sections: {
          resources: {
            label: 'Resources',
            components: {
              engineVersion: {
                uiType: FieldType.Text,
                path: ['spec.engine.version', 'spec.proxy.version'],
                fieldParams: { label: 'Version' },
              },
            },
          },
        },
        sectionsOrder: ['resources'],
      },
    },
  },
};

const CurrentPath = () => {
  const location = useLocation();
  return <div data-testid="current-path">{location.pathname}</div>;
};

describe('Create database flow routing', () => {
  it('opens /databases/new and renders form after clicking Create database', async () => {
    vi.mocked(useProviders).mockReturnValue({
      data: { items: [providerWithMultipathSchema] },
      isLoading: false,
    } as unknown as ReturnType<typeof useProviders>);

    render(
      <ThemeContextProvider themeOptions={everestThemeOptions}>
        <MemoryRouter initialEntries={['/databases']}>
          <Routes>
            <Route
              path="/databases"
              element={
                <>
                  <CreateDbButton />
                  <CurrentPath />
                </>
              }
            />
            <Route
              path="/databases/new"
              element={
                <>
                  <DatabasePage />
                  <CurrentPath />
                </>
              }
            />
          </Routes>
        </MemoryRouter>
      </ThemeContextProvider>
    );

    const openCreateFormBtn = await screen.findByRole('button', {
      name: 'Create database',
    });

    fireEvent.click(openCreateFormBtn);

    await waitFor(() => {
      expect(screen.getByTestId('current-path')).toHaveTextContent(
        '/databases/new'
      );
    });

    await waitFor(() => {
      expect(screen.getByTestId('db-wizard-cancel-button')).toBeInTheDocument();
      expect(screen.getByTestId('db-wizard-submit-button')).toBeInTheDocument();
    });
  });
});
