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

import { render, screen, within } from '@testing-library/react';
import type { ResolvedExtensionMeta } from 'shared-types/extension-catalog.types';
import { TestWrapper } from 'utils/test';
import { NoProviders } from './no-providers';
import { EXTENSION_HUB_WEBSITE_URL } from './no-providers.constants';

const pluginsState = vi.hoisted(() => ({
  value: { plugins: [{ name: 'plugin-hub', extensions: [] }], loading: false },
}));

vi.mock('contexts/plugins', () => ({
  usePlugins: () => pluginsState.value,
}));

const catalog: ResolvedExtensionMeta[] = [
  { name: 'postgres', displayName: 'PostgreSQL', categories: ['database'] },
  { name: 'mysql', displayName: 'MySQL', categories: ['database'] },
  { name: 's3', displayName: 'Object Storage', categories: ['storage'] },
  { name: 'kserve', displayName: 'KServe', categories: ['ai'] },
];

const renderNoProviders = (catalogProviders: ResolvedExtensionMeta[]) =>
  render(
    <TestWrapper>
      <NoProviders catalogProviders={catalogProviders} />
    </TestWrapper>
  );

describe('NoProviders', () => {
  beforeEach(() => {
    pluginsState.value = {
      plugins: [{ name: 'plugin-hub', extensions: [] }],
      loading: false,
    };
  });

  it('suggests catalog providers linking into the plugin-hub, plus a browse tile for the rest', () => {
    renderNoProviders(catalog);

    expect(
      screen.getByTestId('available-provider-tile-postgres')
    ).toHaveAttribute('href', '/plugins/plugin-hub/providers/postgres');
    expect(screen.getByTestId('available-provider-tile-s3')).toHaveAttribute(
      'href',
      '/plugins/plugin-hub/providers/s3'
    );
    expect(
      screen.queryByTestId('available-provider-tile-mysql')
    ).not.toBeInTheDocument();

    const browseTile = screen.getByTestId('browse-extension-hub-tile');
    expect(browseTile).toHaveAttribute('href', '/plugins/plugin-hub/providers');
    expect(browseTile).toHaveTextContent('+2 more providers');
    expect(browseTile).toHaveTextContent('Browse Plugin Hub');
    expect(
      within(browseTile).getByTestId('ArrowForwardIcon')
    ).toBeInTheDocument();
  });

  it('leaves deprecated providers out of the "more providers" count', () => {
    renderNoProviders([
      ...catalog.slice(0, 3),
      { name: 'old-db', displayName: 'Old DB', maturity: 'deprecated' },
      { name: 'old-ai', displayName: 'Old AI', maturity: 'deprecated' },
    ]);

    expect(screen.getByTestId('browse-extension-hub-tile')).toHaveTextContent(
      /\+1 more provider(?!s)/
    );
  });

  it('links to the public Extension Hub when the plugin-hub UI is not loaded', () => {
    pluginsState.value = { plugins: [], loading: false };
    renderNoProviders(catalog);

    const tile = screen.getByTestId('available-provider-tile-postgres');
    expect(tile).toHaveAttribute('href', EXTENSION_HUB_WEBSITE_URL);
    expect(tile).toHaveAttribute('target', '_blank');
    expect(within(tile).getByTestId('OpenInNewIcon')).toBeInTheDocument();

    const browseTile = screen.getByTestId('browse-extension-hub-tile');
    expect(browseTile).toHaveTextContent('Browse Extension Hub');
    expect(within(browseTile).getByTestId('OpenInNewIcon')).toBeInTheDocument();
  });

  it('falls back to a single Extension Hub link without a catalog', () => {
    renderNoProviders([]);

    expect(screen.getByTestId('browse-extension-hub-button')).toHaveAttribute(
      'href',
      EXTENSION_HUB_WEBSITE_URL
    );
    expect(
      screen.queryByTestId('browse-extension-hub-tile')
    ).not.toBeInTheDocument();
  });
});
