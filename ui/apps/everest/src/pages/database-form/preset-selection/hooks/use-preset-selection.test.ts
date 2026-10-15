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
import { ResolvedPreset } from 'shared-types/api.types';
import { useInstancePresets } from 'hooks/api/instance-presets';
import { usePresetSelection } from './use-preset-selection';

vi.mock('hooks/api/instance-presets', () => ({
  useInstancePresets: vi.fn(),
}));

const spec = { providerRef: { name: 'psmdb' } };
const clusterPreset: ResolvedPreset = {
  kind: 'ClusterInstancePreset',
  metadata: { name: 'psmdb-replicaset' },
  spec,
};
const namespacePreset: ResolvedPreset = {
  kind: 'InstancePreset',
  metadata: { name: 'my-preset', namespace: 'ns' },
  spec,
};
const presets = [clusterPreset, namespacePreset];

const mockList = (overrides = {}) =>
  vi.mocked(useInstancePresets).mockReturnValue({
    data: presets,
    isLoading: false,
    isFetching: false,
    isError: false,
    ...overrides,
  } as unknown as ReturnType<typeof useInstancePresets>);

describe('usePresetSelection', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('queries presets for the namespace and provider', () => {
    mockList();

    renderHook(() => usePresetSelection('cluster', 'psmdb', 'ns', ''));

    expect(useInstancePresets).toHaveBeenCalledWith('cluster', 'ns', 'psmdb');
  });

  it('is not selected and not pending when no preset is chosen', () => {
    mockList();

    const { result } = renderHook(() =>
      usePresetSelection('cluster', 'psmdb', 'ns', '')
    );

    expect(result.current.presetSelected).toBe(false);
    expect(result.current.presetPending).toBe(false);
    expect(result.current.resolvedPreset).toBeNull();
  });

  it.each([
    ['cluster-scoped', clusterPreset],
    ['namespace-scoped', namespacePreset],
  ])('picks the chosen %s preset from the list', (_, preset) => {
    mockList();

    const { result } = renderHook(() =>
      usePresetSelection('cluster', 'psmdb', 'ns', preset.metadata?.name ?? '')
    );

    expect(result.current.resolvedPreset).toBe(preset);
    expect(result.current.presetPending).toBe(false);
  });

  it('is pending while the list is refetching for a chosen preset', () => {
    mockList({ isFetching: true });

    const { result } = renderHook(() =>
      usePresetSelection('cluster', 'psmdb', 'ns', 'psmdb-replicaset')
    );

    expect(result.current.presetPending).toBe(true);
  });

  it('is pending when the chosen preset is not in the list', () => {
    mockList();

    const { result } = renderHook(() =>
      usePresetSelection('cluster', 'psmdb', 'ns', 'missing')
    );

    expect(result.current.resolvedPreset).toBeNull();
    expect(result.current.presetPending).toBe(true);
  });

  it('surfaces a resolve error only when a preset is chosen', () => {
    mockList({ data: [], isError: true });

    const { result: unselected } = renderHook(() =>
      usePresetSelection('cluster', 'psmdb', 'ns', '')
    );
    const { result: selected } = renderHook(() =>
      usePresetSelection('cluster', 'psmdb', 'ns', 'psmdb-replicaset')
    );

    expect(unselected.current.isError).toBe(true);
    expect(unselected.current.resolveError).toBeNull();
    expect(selected.current.resolveError).toBeTruthy();
  });
});

