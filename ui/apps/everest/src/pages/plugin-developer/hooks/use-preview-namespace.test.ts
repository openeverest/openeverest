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

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { usePreviewNamespace } from './use-preview-namespace';
import { useNamespaces } from 'hooks/api/namespaces';

vi.mock('hooks/api/namespaces', () => ({
  useNamespaces: vi.fn(),
}));

const mockUseNamespaces = vi.mocked(useNamespaces);

// Only the two fields the hook reads; cast keeps us off the full query type.
const asQuery = (data: string[] | undefined, isLoading = false) =>
  ({ data, isLoading }) as ReturnType<typeof useNamespaces>;

describe('usePreviewNamespace', () => {
  beforeEach(() => {
    mockUseNamespaces.mockReset();
  });

  it('defaults the selection to the first namespace once loaded', () => {
    mockUseNamespaces.mockReturnValue(asQuery(['ns-a', 'ns-b']));

    const { result } = renderHook(() => usePreviewNamespace());

    expect(result.current.namespaces).toEqual(['ns-a', 'ns-b']);
    expect(result.current.selectedNamespace).toBe('ns-a');
  });

  it('passes through the loading state and selects nothing while empty', () => {
    mockUseNamespaces.mockReturnValue(asQuery(undefined, true));

    const { result } = renderHook(() => usePreviewNamespace());

    expect(result.current.isLoading).toBe(true);
    expect(result.current.namespaces).toEqual([]);
    expect(result.current.selectedNamespace).toBe('');
  });

  it('lets the caller change the selected namespace', () => {
    mockUseNamespaces.mockReturnValue(asQuery(['ns-a', 'ns-b']));

    const { result } = renderHook(() => usePreviewNamespace());

    act(() => result.current.setSelectedNamespace('ns-b'));
    expect(result.current.selectedNamespace).toBe('ns-b');
  });

  it('does not overwrite an explicit choice when the list changes', () => {
    mockUseNamespaces.mockReturnValue(asQuery(['ns-a', 'ns-b']));

    const { result, rerender } = renderHook(() => usePreviewNamespace());
    act(() => result.current.setSelectedNamespace('ns-b'));

    // Default-fill only runs while nothing is selected, so the choice survives.
    mockUseNamespaces.mockReturnValue(asQuery(['ns-c', 'ns-a', 'ns-b']));
    rerender();

    expect(result.current.selectedNamespace).toBe('ns-b');
  });
});
