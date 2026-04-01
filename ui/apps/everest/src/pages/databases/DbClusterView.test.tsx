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

import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { createMemoryRouter, RouterProvider } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { DbClusterView } from './DbClusterView';
vi.mock('hooks/api/db-instances/useDbInstanceList');

vi.mock('hooks/api/namespaces/useNamespaces', () => ({
  useNamespaces: () => ({ data: ['default'], isLoading: false }),
}));

vi.mock('hooks/api/providers/useProviders', () => ({
  useProviders: () => ({ data: [] }),
}));

vi.mock('components/db-actions/db-actions', () => ({
  DbActions: () => null,
}));

const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: false } },
});

function renderWithRouter() {
  const router = createMemoryRouter(
    [
      {
        path: '/databases',
        element: (
          <QueryClientProvider client={queryClient}>
            <DbClusterView />
          </QueryClientProvider>
        ),
      },
      {
        path: '/databases/:namespace/:instanceName/overview',
        element: <div data-testid="overview-page">Instance Overview</div>,
      },
    ],
    { initialEntries: ['/databases'] }
  );

  render(<RouterProvider router={router} />);
  return router;
}

describe('DbClusterView navigation', () => {
  it('navigates to the instance overview when a table row is clicked', async () => {
    const router = renderWithRouter();

    // Wait for the table data row to appear
    await waitFor(() => {
      // The table should have more than just the header row
      const rows = screen.getAllByRole('row');
      expect(rows.length).toBeGreaterThan(1);
    });

    // Click the first data row (index 0 is the header)
    const rows = screen.getAllByRole('row');
    const dataRow =
      rows.find((row) => row.getAttribute('data-index') === '0') ?? rows[1];
    fireEvent.click(dataRow);

    await waitFor(() => {
      expect(router.state.location.pathname).toBe(
        '/databases/default/psmdb-primary/overview'
      );
    });
  });

  it('navigates to the correct overview URL containing namespace and instance name', async () => {
    const router = renderWithRouter();

    await waitFor(() => {
      expect(screen.getAllByRole('row').length).toBeGreaterThan(1);
    });

    const rows = screen.getAllByRole('row');
    const dataRow =
      rows.find((row) => row.getAttribute('data-index') === '0') ?? rows[1];
    fireEvent.click(dataRow);

    await waitFor(() => {
      const { pathname } = router.state.location;
      expect(pathname).toContain('/default/');
      expect(pathname).toContain('/psmdb-primary');
      expect(pathname).toMatch(/\/overview$/);
    });
  });
});
