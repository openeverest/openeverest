// everest
// Copyright (C) 2023 Percona LLC
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
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it } from 'vitest';
import EmptyStateWithLink from './empty-state-with-link';

const renderWithRouter = (ui: React.ReactElement) =>
  render(<MemoryRouter>{ui}</MemoryRouter>);

describe('EmptyStateWithLink', () => {
  it('renders the message text', () => {
    renderWithRouter(
      <EmptyStateWithLink
        message="No backup storages configured."
        linkLabel="Go to Storage Settings"
        to="/settings/storage-locations"
      />
    );
    expect(
      screen.getByText('No backup storages configured.')
    ).toBeInTheDocument();
  });

  it('renders a link with the correct label', () => {
    renderWithRouter(
      <EmptyStateWithLink
        message="No backup storages configured."
        linkLabel="Go to Storage Settings"
        to="/settings/storage-locations"
      />
    );
    const link = screen.getByRole('link', { name: /go to storage settings/i });
    expect(link).toBeInTheDocument();
    expect(link).toHaveAttribute('href', '/settings/storage-locations');
  });

  it('applies the dataTestId to the root element and CTA', () => {
    renderWithRouter(
      <EmptyStateWithLink
        message="msg"
        linkLabel="Go"
        to="/somewhere"
        dataTestId="my-empty-state"
      />
    );
    expect(screen.getByTestId('my-empty-state')).toBeInTheDocument();
    expect(screen.getByTestId('my-empty-state-cta')).toBeInTheDocument();
  });

  it('uses the default dataTestId when none is provided', () => {
    renderWithRouter(
      <EmptyStateWithLink
        message="msg"
        linkLabel="Go"
        to="/somewhere"
      />
    );
    expect(screen.getByTestId('empty-state-with-link')).toBeInTheDocument();
  });
});
