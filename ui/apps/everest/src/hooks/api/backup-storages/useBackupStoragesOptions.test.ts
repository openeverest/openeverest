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
import { renderHook } from '@testing-library/react';
import { useBackupStoragesOptions } from './useBackupStoragesOptions';

const loaded = {
  data: [
    { metadata: { name: 's3-prod' } },
    { metadata: {} },
    { metadata: { name: 'minio-dev' } },
  ],
  isLoading: false,
  error: null,
};
const loadedEmpty = { data: [], isLoading: false, error: null };
const loading = { data: undefined, isLoading: true, error: null };

const useBackupStoragesByNamespace = vi.fn();

vi.mock('./useBackupStorages', () => ({
  useBackupStoragesByNamespace: (namespace: string) =>
    useBackupStoragesByNamespace(namespace),
}));

const params = { namespace: 'vectors', cluster: 'main' };

describe('useBackupStoragesOptions', () => {
  beforeEach(() => {
    useBackupStoragesByNamespace.mockReset();
  });

  it('offers the named storages of the form namespace', () => {
    useBackupStoragesByNamespace.mockReturnValue(loaded);

    const { result } = renderHook(() => useBackupStoragesOptions(params));

    expect(useBackupStoragesByNamespace).toHaveBeenCalledWith('vectors');
    expect(result.current.options).toEqual([
      { label: 's3-prod', value: 's3-prod' },
      { label: 'minio-dev', value: 'minio-dev' },
    ]);
    expect(result.current.isEmpty).toBe(false);
  });

  it('is empty once loaded without storages, so the fallback shows', () => {
    useBackupStoragesByNamespace.mockReturnValue(loadedEmpty);

    const { result } = renderHook(() => useBackupStoragesOptions(params));

    expect(result.current.isEmpty).toBe(true);
  });

  it('is not empty while loading', () => {
    useBackupStoragesByNamespace.mockReturnValue(loading);

    const { result } = renderHook(() => useBackupStoragesOptions(params));

    expect(result.current.isEmpty).toBe(false);
    expect(result.current.options).toEqual([]);
  });
});
