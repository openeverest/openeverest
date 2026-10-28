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
import {
  SECRETS_QUERY_KEY,
  useCreateSecret,
  useDeleteSecret,
  useSecrets,
} from './useSecrets';

vi.mock('api/api', () => ({
  api: { get: vi.fn(), post: vi.fn(), delete: vi.fn() },
}));

const renderWithClient = <T,>(hook: () => T) => {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  const wrapper = ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
  return { queryClient, ...renderHook(hook, { wrapper }) };
};

describe('useSecrets', () => {
  it('lists secrets of one definition and hides those being deleted', async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: {
        items: [
          { metadata: { name: 'creds-a' } },
          {
            metadata: {
              name: 'creds-b',
              deletionTimestamp: '2026-10-08T00:00:00Z',
            },
          },
        ],
      },
    });

    const { result } = renderWithClient(() =>
      useSecrets('main', 'ns', { provider: 'psmdb', definition: 'user' })
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(api.get).toHaveBeenCalledWith(
      'clusters/main/namespaces/ns/secrets',
      { params: { provider: 'psmdb', definition: 'user' } }
    );
    expect(result.current.data?.map((s) => s.metadata?.name)).toEqual([
      'creds-a',
    ]);
  });
});

describe('secret mutations', () => {
  it.each([
    {
      name: 'create',
      run: () => {
        vi.mocked(api.post).mockResolvedValue({ data: {} });
        const hook = renderWithClient(() => useCreateSecret('main', 'ns'));
        return {
          hook,
          mutate: () =>
            hook.result.current.mutateAsync({ metadata: { name: 'creds-a' } }),
        };
      },
    },
    {
      name: 'delete',
      run: () => {
        vi.mocked(api.delete).mockResolvedValue({ data: undefined });
        const hook = renderWithClient(() => useDeleteSecret('main', 'ns'));
        return {
          hook,
          mutate: () => hook.result.current.mutateAsync('creds-a'),
        };
      },
    },
  ])(
    '$name refreshes the namespace secret lists on success',
    async ({ run }) => {
      const { hook, mutate } = run();
      const invalidate = vi.spyOn(hook.queryClient, 'invalidateQueries');

      await act(async () => {
        await mutate();
      });

      expect(invalidate).toHaveBeenCalledWith({
        queryKey: [SECRETS_QUERY_KEY, 'main', 'ns'],
      });
    }
  );
});
