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
import { ClusterInstancePreset } from 'shared-types/api.types';
import { useClusterInstancePresets } from 'hooks/api/cluster-instance-presets';
import { usePresetSelection } from './use-preset-selection';

vi.mock('hooks/api/cluster-instance-presets', () => ({
  useClusterInstancePresets: vi.fn(),
}));

const replicaset: ClusterInstancePreset = {
  metadata: { name: 'psmdb-replicaset' },
  spec: { providerRef: { name: 'psmdb' } },
};

const mockList = (overrides = {}) =>
  vi.mocked(useClusterInstancePresets).mockReturnValue({
    data: [replicaset],
    isLoading: false,
    isError: false,
    ...overrides,
  } as unknown as ReturnType<typeof useClusterInstancePresets>);

describe('usePresetSelection', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('is not selected and not pending when no preset is chosen', () => {
    mockList();

    const { result } = renderHook(() =>
      usePresetSelection('cluster', 'psmdb', '')
    );

    expect(result.current.presetSelected).toBe(false);
    expect(result.current.presetPending).toBe(false);
    expect(result.current.selectedPreset).toBeNull();
  });

  it('selects the chosen preset from the list', () => {
    mockList();

    const { result } = renderHook(() =>
      usePresetSelection('cluster', 'psmdb', 'psmdb-replicaset')
    );

    expect(result.current.presetSelected).toBe(true);
    expect(result.current.presetPending).toBe(false);
    expect(result.current.selectedPreset).toBe(replicaset);
  });

  it('is pending while the chosen preset is not in the list yet', () => {
    mockList({ data: [], isLoading: true });

    const { result } = renderHook(() =>
      usePresetSelection('cluster', 'psmdb', 'psmdb-replicaset')
    );

    expect(result.current.presetPending).toBe(true);
    expect(result.current.selectedPreset).toBeNull();
  });

  it('reports the list error state from the presets query', () => {
    mockList({ isError: true });

    const { result } = renderHook(() =>
      usePresetSelection('cluster', 'psmdb', '')
    );

    expect(result.current.isError).toBe(true);
  });
});
