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
import { ClusterInstancePreset, InstancePreset } from 'shared-types/api.types';
import { useClusterInstancePresets } from 'hooks/api/cluster-instance-presets';
import { useInstancePresets } from 'hooks/api/instance-presets';
import { PresetScope } from '../preset-selection.types';
import {
  getClusterPresetId,
  getNamespacePresetId,
} from '../preset-selection.utils';
import { usePresetSelection } from './use-preset-selection';

vi.mock('hooks/api/cluster-instance-presets', () => ({
  useClusterInstancePresets: vi.fn(),
}));
vi.mock('hooks/api/instance-presets', () => ({
  useInstancePresets: vi.fn(),
}));

const clusterPreset: ClusterInstancePreset = {
  metadata: { name: 'psmdb-default' },
  spec: { providerRef: { name: 'psmdb' }, version: '8.0.12' },
};
// Same name as the cluster preset, and as each other across namespaces.
const teamAPreset: InstancePreset = {
  metadata: { name: 'psmdb-default', namespace: 'team-a' },
  spec: { providerRef: { name: 'psmdb' }, version: '7.0.1' },
};
const teamBPreset: InstancePreset = {
  metadata: { name: 'psmdb-default', namespace: 'team-b' },
  spec: { providerRef: { name: 'psmdb' }, version: '6.0.9' },
};

const settled = { isLoading: false, isFetching: false, isError: false };

const mockClusterList = (overrides = {}) =>
  vi.mocked(useClusterInstancePresets).mockReturnValue({
    data: [clusterPreset],
    ...settled,
    ...overrides,
  } as unknown as ReturnType<typeof useClusterInstancePresets>);

const mockNamespaceLists = (
  lists: Record<string, object> = {
    'team-a': { data: [teamAPreset], ...settled },
    'team-b': { data: [teamBPreset], ...settled },
  }
) =>
  vi.mocked(useInstancePresets).mockImplementation(
    (_cluster, namespace) =>
      (lists[namespace] ?? {
        data: undefined,
        ...settled,
      }) as unknown as ReturnType<typeof useInstancePresets>
  );

const CLUSTER_ID = getClusterPresetId('psmdb-default');
const TEAM_A_ID = getNamespacePresetId('team-a', 'psmdb-default');

const setup = (namespace: string, presetId: string) =>
  renderHook(
    (props: { namespace: string; presetId: string }) =>
      usePresetSelection('main', 'psmdb', props.namespace, props.presetId),
    { initialProps: { namespace, presetId } }
  );

describe('usePresetSelection', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockClusterList();
    mockNamespaceLists();
  });

  it('is not selected and not pending when no preset is chosen', () => {
    const { result } = setup('team-a', '');

    expect(result.current.presetSelected).toBe(false);
    expect(result.current.presetPending).toBe(false);
    expect(result.current.selectedPreset).toBeNull();
  });

  it('offers cluster presets first, then the namespace ones', () => {
    const { result } = setup('team-a', '');

    expect(result.current.presets.map((preset) => preset.scope)).toEqual([
      PresetScope.Cluster,
      PresetScope.Namespace,
    ]);
  });

  it('selects the namespace preset, not the cluster one sharing its name', () => {
    const { result } = setup('team-a', TEAM_A_ID);

    expect(result.current.selectedPreset).toMatchObject({
      scope: PresetScope.Namespace,
      name: 'psmdb-default',
    });
    expect(result.current.selectedPreset?.spec).toBe(teamAPreset.spec);
    expect(result.current.presetPending).toBe(false);
  });

  it('is pending, not unavailable, while the picked preset list is loading', () => {
    mockClusterList({ data: undefined, isLoading: true, isFetching: true });

    const { result } = setup('team-a', CLUSTER_ID);

    expect(result.current.presetPending).toBe(true);
    expect(result.current.presetUnavailable).toBe(false);
  });

  it('drops a namespace preset on a namespace switch instead of resolving the same name there', () => {
    const { result, rerender } = setup('team-a', TEAM_A_ID);

    rerender({ namespace: 'team-b', presetId: TEAM_A_ID });

    expect(result.current.selectedPreset).toBeNull();
    expect(result.current.presetUnavailable).toBe(true);
    expect(result.current.droppedPreset).toEqual({
      name: 'psmdb-default',
      namespace: 'team-b',
      presetNamespace: 'team-a',
    });
  });

  it('does not wait for the new namespace list to drop a preset from another namespace', () => {
    mockNamespaceLists({
      'team-a': { data: [teamAPreset], ...settled },
      'team-b': { data: undefined, ...settled, isFetching: true },
    });
    const { result, rerender } = setup('team-a', TEAM_A_ID);

    rerender({ namespace: 'team-b', presetId: TEAM_A_ID });

    expect(result.current.presetUnavailable).toBe(true);
  });

  it('keeps a cluster preset selected across a namespace switch', () => {
    const { result, rerender } = setup('team-a', CLUSTER_ID);

    rerender({ namespace: 'team-b', presetId: CLUSTER_ID });

    expect(result.current.selectedPreset?.spec).toBe(clusterPreset.spec);
    expect(result.current.presetUnavailable).toBe(false);
    expect(result.current.droppedPreset).toBeNull();
  });

  it('reports a preset that is gone from its settled list as unavailable', () => {
    mockClusterList({ data: [] });

    const { result } = setup('team-a', CLUSTER_ID);

    expect(result.current.presetUnavailable).toBe(true);
  });

  it('keeps the dropped-preset notice until the next pick or namespace change', () => {
    const { result, rerender } = setup('team-a', TEAM_A_ID);
    rerender({ namespace: 'team-b', presetId: TEAM_A_ID });

    // The form sync clears the pick; the notice must outlive that.
    rerender({ namespace: 'team-b', presetId: '' });
    expect(result.current.droppedPreset?.name).toBe('psmdb-default');

    rerender({ namespace: 'team-b', presetId: CLUSTER_ID });
    expect(result.current.droppedPreset).toBeNull();

    rerender({ namespace: 'team-a', presetId: TEAM_A_ID });
    rerender({ namespace: 'team-b', presetId: TEAM_A_ID });
    rerender({ namespace: 'team-b', presetId: '' });
    rerender({ namespace: 'team-a', presetId: '' });
    expect(result.current.droppedPreset).toBeNull();
  });

  it('still offers cluster presets when the namespace list fails to load', () => {
    mockNamespaceLists({
      'team-a': { data: undefined, ...settled, isError: true },
    });

    const { result } = setup('team-a', '');

    expect(result.current.isError).toBe(true);
    expect(result.current.isLoadingPresets).toBe(false);
    expect(result.current.presets.map((preset) => preset.id)).toEqual([
      CLUSTER_ID,
    ]);
  });
});
