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

import { renderHook } from '@testing-library/react';
import { InstancePreset } from 'shared-types/api.types';
import {
  useInstancePresets,
  useResolveInstancePreset,
} from 'hooks/api/instance-presets';
import { usePresetSelection } from './use-preset-selection';

vi.mock('hooks/api/instance-presets', () => ({
  useInstancePresets: vi.fn(),
  useResolveInstancePreset: vi.fn(),
}));

const resolved = { spec: { version: '8.0.12' } } as unknown as InstancePreset;

const mockList = (overrides = {}) =>
  vi.mocked(useInstancePresets).mockReturnValue({
    data: [],
    isLoading: false,
    isError: false,
    ...overrides,
  } as unknown as ReturnType<typeof useInstancePresets>);

const mockResolve = (overrides = {}) =>
  vi.mocked(useResolveInstancePreset).mockReturnValue({
    data: undefined,
    isFetching: false,
    isError: false,
    ...overrides,
  } as unknown as ReturnType<typeof useResolveInstancePreset>);

describe('usePresetSelection', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('is not selected and not pending when no preset is chosen', () => {
    mockList();
    mockResolve();

    const { result } = renderHook(() =>
      usePresetSelection('cluster', 'psmdb', 'ns', '')
    );

    expect(result.current.presetSelected).toBe(false);
    expect(result.current.presetPending).toBe(false);
  });

  it('is pending while a chosen preset is still resolving', () => {
    mockList();
    mockResolve({ data: undefined, isFetching: true });

    const { result } = renderHook(() =>
      usePresetSelection('cluster', 'psmdb', 'ns', 'psmdb-replicaset')
    );

    expect(result.current.presetSelected).toBe(true);
    expect(result.current.presetPending).toBe(true);
  });

  it('is pending when a chosen preset has not resolved yet', () => {
    mockList();
    mockResolve({ data: undefined, isFetching: false });

    const { result } = renderHook(() =>
      usePresetSelection('cluster', 'psmdb', 'ns', 'psmdb-replicaset')
    );

    expect(result.current.presetPending).toBe(true);
    expect(result.current.resolvedPreset).toBeNull();
  });

  it('stops being pending once the preset resolves', () => {
    mockList();
    mockResolve({ data: resolved, isFetching: false });

    const { result } = renderHook(() =>
      usePresetSelection('cluster', 'psmdb', 'ns', 'psmdb-replicaset')
    );

    expect(result.current.presetPending).toBe(false);
    expect(result.current.resolvedPreset).toBe(resolved);
  });

  it('surfaces a resolve error only when a preset is chosen', () => {
    mockList();
    mockResolve({ data: undefined, isFetching: false, isError: true });

    const { result } = renderHook(() =>
      usePresetSelection('cluster', 'psmdb', 'ns', 'psmdb-replicaset')
    );

    expect(result.current.resolveError).toBeTruthy();
  });

  it('reports the list error state from the presets query', () => {
    mockList({ isError: true });
    mockResolve();

    const { result } = renderHook(() =>
      usePresetSelection('cluster', 'psmdb', 'ns', '')
    );

    expect(result.current.isError).toBe(true);
  });
});
