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

import React from 'react';
import { act, renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { api } from 'api/api';
import { useClusterInstancePresets } from 'hooks/api/cluster-instance-presets';
import { useInstancePresets } from './useInstancePresets';
import { usePrefetchPresets } from './usePrefetchPresets';

vi.mock('api/api', () => ({ api: { get: vi.fn() } }));
vi.mock('hooks/api/useClusterName', () => ({ useClusterName: () => 'main' }));
// One object for every call, as a query result is: a fresh one per render
// would re-create the prefetch callback each time.
const namespacesQuery = vi.hoisted((): { data: string[] | undefined } => ({
  data: undefined,
}));
vi.mock('hooks/api/namespaces', () => ({
  useNamespaces: () => namespacesQuery,
}));

const PROVIDER = { metadata: { name: 'psmdb' }, spec: {} };

const renderWithClient = <T,>(hook: () => T) => {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  const wrapper = ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
  return renderHook(hook, { wrapper });
};

const requestedPaths = () =>
  vi.mocked(api.get).mock.calls.map(([path]) => path);

describe('usePrefetchPresets', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(api.get).mockResolvedValue({
      data: { items: [{ metadata: { name: 'preset' }, spec: {} }] },
    });
  });

  // The prefetch only pays off if it fills the very entries the wizard reads:
  // the lists must then be there on the first render, with no loading state.
  it('has both lists ready for the wizard to open with', async () => {
    namespacesQuery.data = ['team-a', 'team-b'];
    const { result, rerender } = renderWithClient(
      ({ wizardOpen }: { wizardOpen: boolean } = { wizardOpen: false }) => ({
        prefetch: usePrefetchPresets(),
        cluster: useClusterInstancePresets(
          'main',
          wizardOpen ? 'psmdb' : undefined
        ),
        namespace: useInstancePresets(
          'main',
          'team-a',
          wizardOpen ? 'psmdb' : undefined
        ),
      })
    );

    act(() => result.current.prefetch(PROVIDER));
    await waitFor(() => expect(requestedPaths()).toHaveLength(2));
    rerender({ wizardOpen: true });

    expect(result.current.cluster.isLoading).toBe(false);
    expect(result.current.cluster.data).toHaveLength(1);
    expect(result.current.namespace.isLoading).toBe(false);
    expect(result.current.namespace.data).toHaveLength(1);
  });

  it('prefetches the namespace the wizard defaults to', async () => {
    namespacesQuery.data = ['team-a', 'team-b'];
    const { result } = renderWithClient(() => usePrefetchPresets());

    act(() => result.current(PROVIDER));

    await waitFor(() =>
      expect(requestedPaths()).toEqual([
        'clusters/main/cluster-instance-presets?provider=psmdb',
        'clusters/main/namespaces/team-a/instance-presets?provider=psmdb',
      ])
    );
  });

  it('prefetches only the cluster presets while the namespaces are unknown', async () => {
    namespacesQuery.data = undefined;
    const { result } = renderWithClient(() => usePrefetchPresets());

    act(() => result.current(PROVIDER));

    await waitFor(() => expect(requestedPaths()).toHaveLength(1));
    expect(requestedPaths()[0]).toContain('cluster-instance-presets');
  });
});
