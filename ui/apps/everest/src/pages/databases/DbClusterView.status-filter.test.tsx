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

import {
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { Instance } from 'shared-types/api.types';
import { InstanceCondition } from 'shared-types/instance.types';
import { DbClusterView } from './DbClusterView';

const { instancesResults } = vi.hoisted(() => {
  const condition = (
    type: string,
    status: InstanceCondition['status'],
    reason = type
  ): InstanceCondition => ({
    type,
    status,
    reason,
    message: '',
    lastTransitionTime: '2026-03-01T11:00:00Z',
  });
  const instance = (
    name: string,
    conditions: InstanceCondition[],
    phase: 'Pending' | 'Provisioned' = 'Provisioned'
  ): Instance => ({
    apiVersion: 'core.openeverest.io/v1alpha1',
    kind: 'Instance',
    metadata: { name, namespace: 'default' },
    spec: { providerRef: { name: 'psmdb' } },
    status: { phase, conditions },
  });
  const crashLoop = (stalled: boolean) => [
    condition('Ready', 'False', 'CrashLoopBackOff'),
    condition('Stalled', stalled ? 'True' : 'False'),
    condition('NeedsAttention', 'True', 'CrashLoopBackOff'),
  ];
  const instances = [
    instance('orders', [condition('Ready', 'True')]),
    instance('legacy', crashLoop(false)),
    instance('legacy-old', crashLoop(true)),
    instance('staging', [condition('Ready', 'False', 'Creating')], 'Pending'),
  ];

  // Stable across renders, as react-query returns them.
  return {
    instancesResults: [
      {
        namespace: 'default',
        queryResult: {
          data: instances,
          isSuccess: true,
          isLoading: false,
          isFetchedAfterMount: true,
        },
      },
    ],
  };
});

vi.mock('hooks/api/db-instances/useDbInstanceList', () => ({
  useInstancesForNamespaces: () => instancesResults,
}));

vi.mock('hooks/api/namespaces/useNamespaces', () => {
  const result = { data: ['default'], isLoading: false };
  return { useNamespaces: () => result };
});

vi.mock('hooks/api/providers/useProviders', () => ({
  useProviders: () => ({ data: [] }),
}));

vi.mock('components/db-actions/db-actions', () => ({
  DbActions: () => null,
}));

const renderList = () =>
  render(
    <QueryClientProvider
      client={
        new QueryClient({ defaultOptions: { queries: { retry: false } } })
      }
    >
      <MemoryRouter>
        <DbClusterView />
      </MemoryRouter>
    </QueryClientProvider>
  );

const shownInstances = () =>
  ['orders', 'legacy', 'legacy-old', 'staging'].filter(
    (name) => screen.queryByTestId(`${name}-status`) !== null
  );

describe('DbClusterView status filtering', () => {
  it('narrows the list to the Instances that need attention and back', () => {
    renderList();
    const chip = screen.getByTestId('needs-attention-chip');

    expect(chip).toHaveTextContent('Needs attention (2)');

    fireEvent.click(chip);
    expect(shownInstances()).toEqual(['legacy', 'legacy-old']);

    fireEvent.click(chip);
    expect(shownInstances()).toHaveLength(4);
  });

  it('offers the words the status column shows, the most urgent first', async () => {
    renderList();

    fireEvent.click(screen.getByRole('button', { name: 'Show/Hide filters' }));
    fireEvent.mouseDown(
      screen.getByRole('combobox', { name: 'Filter by Status' })
    );
    const options = within(screen.getByRole('listbox'))
      .getAllByRole('option')
      .map((option) => option.textContent?.trim());

    expect(options).toEqual(['Crash loop', 'Creating', 'Ready']);

    fireEvent.click(screen.getByRole('option', { name: 'Crash loop' }));
    // The column filter applies after a debounce.
    await waitFor(() =>
      expect(shownInstances()).toEqual(['legacy', 'legacy-old'])
    );
  });
});
