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

import { ReactNode } from 'react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useSecretsOptions } from './useSecretsOptions';

const getSecretsFn = vi.fn();
const permissions = { canRead: true };

vi.mock('api/secretApi', () => ({
  getSecretsFn: (cluster: string, namespace: string) =>
    getSecretsFn(cluster, namespace),
}));

vi.mock('hooks/rbac', () => ({
  useRBACPermissions: () => permissions,
}));

const wrapper = ({ children }: { children: ReactNode }) => (
  <QueryClientProvider
    client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}
  >
    {children}
  </QueryClientProvider>
);

const params = { namespace: 'vectors', cluster: 'main' };

describe('useSecretsOptions', () => {
  beforeEach(() => {
    getSecretsFn.mockReset();
    permissions.canRead = true;
  });

  it('offers the named Secrets of the form namespace', async () => {
    getSecretsFn.mockResolvedValue({
      items: [
        { metadata: { name: 's3-keys' } },
        { metadata: {} },
        { metadata: { name: 'tls' } },
      ],
    });

    const { result } = renderHook(() => useSecretsOptions(params), {
      wrapper,
    });

    await waitFor(() =>
      expect(result.current.options).toEqual([
        { label: 's3-keys', value: 's3-keys' },
        { label: 'tls', value: 'tls' },
      ])
    );
    expect(getSecretsFn).toHaveBeenCalledWith('main', 'vectors');
  });

  it('does not ask the API without read permission', () => {
    permissions.canRead = false;

    const { result } = renderHook(() => useSecretsOptions(params), {
      wrapper,
    });

    expect(getSecretsFn).not.toHaveBeenCalled();
    expect(result.current.isEmpty).toBe(true);
  });
});
