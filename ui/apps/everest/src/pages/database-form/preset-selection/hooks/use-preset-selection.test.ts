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
import { useClusterInstancePresets } from 'hooks/api/cluster-instance-presets';
import { useInstancePresets } from 'hooks/api/instance-presets';
import {
  PRESET_NAME,
  PRESET_PROVIDER,
  TEAM_A,
  TEAM_B,
  clusterPreset,
  clusterPresetItem,
  teamAPreset,
  teamAPresetItem,
  teamBPresetItem,
} from '../__mocks__/presets';
import { PresetScope } from '../preset-selection.types';
import { usePresetSelection } from './use-preset-selection';

vi.mock('hooks/api/cluster-instance-presets', () => ({
  useClusterInstancePresets: vi.fn(),
}));
vi.mock('hooks/api/instance-presets', () => ({
  useInstancePresets: vi.fn(),
}));

const settled = { isLoading: false, isFetching: false, isError: false };

const mockClusterList = (overrides = {}) =>
  vi.mocked(useClusterInstancePresets).mockReturnValue({
    data: [clusterPresetItem],
    ...settled,
    ...overrides,
  } as unknown as ReturnType<typeof useClusterInstancePresets>);

const mockNamespaceLists = (
  lists: Record<string, object> = {
    [TEAM_A]: { data: [teamAPresetItem], ...settled },
    [TEAM_B]: { data: [teamBPresetItem], ...settled },
  }
) =>
  vi.mocked(useInstancePresets).mockImplementation(
    (_cluster, namespace) =>
      (lists[namespace] ?? {
        data: undefined,
        ...settled,
      }) as unknown as ReturnType<typeof useInstancePresets>
  );

const CLUSTER_ID = clusterPreset.id;
const TEAM_A_ID = teamAPreset.id;

const setup = (namespace: string, presetId: string) =>
  renderHook(
    (props: { namespace: string; presetId: string }) =>
      usePresetSelection(
        'main',
        PRESET_PROVIDER,
        props.namespace,
        props.presetId
      ),
    { initialProps: { namespace, presetId } }
  );

describe('usePresetSelection', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockClusterList();
    mockNamespaceLists();
  });

  it('is not selected and not pending when no preset is chosen', () => {
    const { result } = setup(TEAM_A, '');

    expect(result.current.presetSelected).toBe(false);
    expect(result.current.presetPending).toBe(false);
    expect(result.current.selectedPreset).toBeNull();
  });

  it('offers cluster presets first, then the namespace ones', () => {
    const { result } = setup(TEAM_A, '');

    expect(result.current.presets.map((preset) => preset.scope)).toEqual([
      PresetScope.Cluster,
      PresetScope.Namespace,
    ]);
  });

  it('selects the namespace preset, not the cluster one sharing its name', () => {
    const { result } = setup(TEAM_A, TEAM_A_ID);

    expect(result.current.selectedPreset).toEqual(teamAPreset);
    expect(result.current.presetPending).toBe(false);
  });

  it('is pending, not unavailable, while the picked preset list is loading', () => {
    mockClusterList({ data: undefined, isLoading: true, isFetching: true });

    const { result } = setup(TEAM_A, CLUSTER_ID);

    expect(result.current.presetPending).toBe(true);
    expect(result.current.presetUnavailable).toBe(false);
  });

  it('drops a namespace preset on a namespace switch instead of resolving the same name there', () => {
    const { result, rerender } = setup(TEAM_A, TEAM_A_ID);

    rerender({ namespace: TEAM_B, presetId: TEAM_A_ID });

    expect(result.current.selectedPreset).toBeNull();
    expect(result.current.presetUnavailable).toBe(true);
    expect(result.current.droppedPreset).toEqual({
      name: PRESET_NAME,
      wizardNamespace: TEAM_B,
      ownerNamespace: TEAM_A,
    });
  });

  it('does not wait for the new namespace list to drop a preset from another namespace', () => {
    mockNamespaceLists({
      [TEAM_A]: { data: [teamAPresetItem], ...settled },
      [TEAM_B]: { data: undefined, ...settled, isFetching: true },
    });
    const { result, rerender } = setup(TEAM_A, TEAM_A_ID);

    rerender({ namespace: TEAM_B, presetId: TEAM_A_ID });

    expect(result.current.presetUnavailable).toBe(true);
  });

  it('keeps a cluster preset selected across a namespace switch', () => {
    const { result, rerender } = setup(TEAM_A, CLUSTER_ID);

    rerender({ namespace: TEAM_B, presetId: CLUSTER_ID });

    expect(result.current.selectedPreset).toEqual(clusterPreset);
    expect(result.current.presetUnavailable).toBe(false);
    expect(result.current.droppedPreset).toBeNull();
  });

  it('reports a preset that is gone from its settled list as unavailable', () => {
    mockClusterList({ data: [] });

    const { result } = setup(TEAM_A, CLUSTER_ID);

    expect(result.current.presetUnavailable).toBe(true);
  });

  it('keeps the dropped-preset notice until the next pick or namespace change', () => {
    const { result, rerender } = setup(TEAM_A, TEAM_A_ID);
    rerender({ namespace: TEAM_B, presetId: TEAM_A_ID });

    // The form sync clears the pick; the notice must outlive that.
    rerender({ namespace: TEAM_B, presetId: '' });
    expect(result.current.droppedPreset?.name).toBe(PRESET_NAME);

    rerender({ namespace: TEAM_B, presetId: CLUSTER_ID });
    expect(result.current.droppedPreset).toBeNull();

    rerender({ namespace: TEAM_A, presetId: TEAM_A_ID });
    rerender({ namespace: TEAM_B, presetId: TEAM_A_ID });
    rerender({ namespace: TEAM_B, presetId: '' });
    rerender({ namespace: TEAM_A, presetId: '' });
    expect(result.current.droppedPreset).toBeNull();
  });

  it('waits for both lists before the cards first appear', () => {
    mockNamespaceLists({
      [TEAM_A]: { data: undefined, ...settled, isLoading: true },
    });
    const { result, rerender } = setup(TEAM_A, '');

    expect(result.current.isLoadingPresets).toBe(true);

    mockNamespaceLists();
    rerender({ namespace: TEAM_A, presetId: '' });

    expect(result.current.isLoadingPresets).toBe(false);
    expect(result.current.presets).toHaveLength(2);
  });

  it('keeps the cluster presets on screen while another namespace loads', () => {
    mockNamespaceLists({
      [TEAM_A]: { data: [teamAPresetItem], ...settled },
      [TEAM_B]: { data: undefined, ...settled, isLoading: true },
    });
    const { result, rerender } = setup(TEAM_A, '');

    rerender({ namespace: TEAM_B, presetId: '' });

    expect(result.current.isLoadingPresets).toBe(false);
    expect(result.current.presets).toEqual([clusterPreset]);
  });

  it('still offers cluster presets when the namespace list fails to load', () => {
    mockNamespaceLists({
      [TEAM_A]: { data: undefined, ...settled, isError: true },
    });

    const { result } = setup(TEAM_A, '');

    expect(result.current.isError).toBe(true);
    expect(result.current.isLoadingPresets).toBe(false);
    expect(result.current.presets.map((preset) => preset.id)).toEqual([
      CLUSTER_ID,
    ]);
  });
});
