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
import { describe, expect, it } from 'vitest';
import { DbInstanceContext } from 'pages/db-cluster-details/dbCluster.context';
import ConnectionSection from './connection-section';

const renderWithContext = (
  canReadCredentials: boolean,
  credentials?: {
    host?: string;
    port?: string;
    username?: string;
    password?: string;
    uri?: string;
    type?: string;
  }
) =>
  render(
    <DbInstanceContext.Provider
      value={{
        instance: undefined,
        isLoading: false,
        instanceDeleted: false,
        canReadCredentials,
      }}
    >
      <ConnectionSection credentials={credentials} loading={false} />
    </DbInstanceContext.Provider>
  );

describe('ConnectionSection', () => {
  it('renders split hosts and credentials when allowed', () => {
    renderWithContext(true, {
      host: 'host-a, host-b',
      port: '3306',
      username: 'admin',
      password: 'secret',
      uri: 'postgresql://example-url',
      type: 'postgresql',
    });

    expect(screen.getByText('host-a')).toBeInTheDocument();
    expect(screen.getByText('host-b')).toBeInTheDocument();
    expect(screen.getByText('admin')).toBeInTheDocument();
    expect(screen.getByTestId('hidden-row')).toBeInTheDocument();
    expect(screen.getByLabelText('Connection URL')).toBeInTheDocument();
  });

  it('shows waiting message without credentials', () => {
    renderWithContext(true, undefined);

    expect(
      screen.getByText('Waiting for instance to be ready...')
    ).toBeInTheDocument();
  });
});
