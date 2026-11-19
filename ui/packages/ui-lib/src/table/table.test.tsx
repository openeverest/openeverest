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
import { ThemeProvider, createTheme } from '@mui/material';
import { everestThemeOptions } from '@percona/design';
import Table from './table';

const theme = createTheme(everestThemeOptions('light'));

describe('Table', () => {
  it('shows quick filters among its own toolbar buttons', () => {
    render(
      <ThemeProvider theme={theme}>
        <Table
          tableName="quick-filter-test"
          columns={[{ accessorKey: 'name', header: 'Name' }]}
          data={[{ name: 'Apple' }]}
          renderToolbarFilters={({ table }) => (
            <span>{`${table.getRowModel().rows.length} rows`}</span>
          )}
        />
      </ThemeProvider>
    );

    const filtersButton = screen.getByRole('button', {
      name: 'Show/Hide filters',
    });
    const toolbarGroup = filtersButton.closest(
      '.percona-table-internal-actions'
    );

    expect(toolbarGroup).toContainElement(screen.getByText('1 rows'));
  });
});
