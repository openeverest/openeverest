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

import { act, renderHook, waitFor } from '@testing-library/react';
import { useForm, useWatch } from 'react-hook-form';
import {
  FormMode,
  TopologyUISchemas,
} from 'components/ui-generator/ui-generator.types';
import { getByPath } from 'components/ui-generator/utils/object-path/object-path';
import { DbWizardFormFields } from 'consts';
import { DbWizardType } from '../database-form-schema';
import { useDatabaseFormSync } from '../hooks/use-database-form-sync';
import { PresetScope, usePresetSelection } from '../preset-selection';
import {
  PRESET_NAME,
  PRESET_PROVIDER,
  TEAM_A,
  TEAM_B,
  clusterPreset,
  teamAPreset,
} from '../preset-selection/__mocks__/presets';

// The wizard page wiring, without the page: the namespace default, the preset
// selection and the form sync only work together, so they are run together.
const requestedNamespaces = vi.hoisted(() => new Set<string>());

vi.mock('hooks/api/cluster-instance-presets', async () => {
  const { clusterPresetItem } = await import(
    '../preset-selection/__mocks__/presets'
  );
  const list = {
    data: [clusterPresetItem],
    isLoading: false,
    isFetching: false,
    isError: false,
  };
  return { useClusterInstancePresets: () => list };
});

// Both namespaces have a preset of the same name: the one of the other
// namespace must never be picked up silently.
vi.mock('hooks/api/instance-presets', async () => {
  const { TEAM_A, TEAM_B, teamAPresetItem, teamBPresetItem } = await import(
    '../preset-selection/__mocks__/presets'
  );
  const settled = { isLoading: false, isFetching: false, isError: false };
  const lists: Record<string, object> = {
    [TEAM_A]: { data: [teamAPresetItem], ...settled },
    [TEAM_B]: { data: [teamBPresetItem], ...settled },
  };
  const noList = { data: undefined, ...settled };
  return {
    useInstancePresets: (_clusterName: string, namespace: string) => {
      requestedNamespaces.add(namespace);
      return lists[namespace] ?? noList;
    },
  };
});

const uiSchema: TopologyUISchemas = { replicaSet: { sections: {} } };
const FORM_DEFAULTS = { backup: { schedules: [] } };

const setup = () => {
  const view = renderHook(() => {
    const methods = useForm<DbWizardType>({
      defaultValues: {
        provider: PRESET_PROVIDER,
        dbName: 'my-db',
        k8sNamespace: '',
        presetId: '',
        topology: { type: 'replicaSet' },
      },
    });
    const { control, reset, getValues, setValue } = methods;
    const selectedNamespace = useWatch({
      control,
      name: DbWizardFormFields.k8sNamespace,
    });
    const presetId = useWatch({ control, name: DbWizardFormFields.presetId });
    const selection = usePresetSelection(
      'main',
      PRESET_PROVIDER,
      selectedNamespace || '',
      presetId || ''
    );
    const notice = useDatabaseFormSync({
      mode: FormMode.New,
      uiSchema,
      defaultValues: FORM_DEFAULTS,
      defaultTopology: 'replicaSet',
      selectedTopology: 'replicaSet',
      defaultNamespace: TEAM_A,
      namespace: selectedNamespace || '',
      preset: selection,
      reset,
      getValues,
      setValue,
    });
    return { methods, selection, notice };
  });

  const read = (path: string): unknown =>
    getByPath(view.result.current.methods.getValues(), path);
  const pickPreset = (id: string) =>
    act(() => {
      view.result.current.methods.setValue(DbWizardFormFields.presetId, id);
    });
  const switchNamespace = (namespace: string) =>
    act(() => {
      view.result.current.methods.setValue(
        DbWizardFormFields.k8sNamespace,
        namespace
      );
    });

  // Picks team-a's preset and switches to team-b, which drops it.
  const dropNamespacePreset = async () => {
    pickPreset(teamAPreset.id);
    await waitFor(() => expect(read('backup.schedules')).toHaveLength(1));
    switchNamespace(TEAM_B);
    await waitFor(() =>
      expect(view.result.current.notice.droppedPreset).not.toBeNull()
    );
  };

  return { ...view, read, pickPreset, switchNamespace, dropNamespacePreset };
};

describe('presets across namespaces in the create wizard', () => {
  beforeEach(() => {
    requestedNamespaces.clear();
  });

  it('lists the presets of the namespace the wizard defaults to', () => {
    const { result } = setup();

    expect(requestedNamespaces).toContain(TEAM_A);
    expect(result.current.selection.presets.map(({ id }) => id)).toEqual([
      clusterPreset.id,
      teamAPreset.id,
    ]);
  });

  it('drops a namespace preset on a namespace switch and keeps the new namespace', async () => {
    const { result, read, pickPreset, switchNamespace } = setup();
    pickPreset(teamAPreset.id);
    await waitFor(() => expect(read('backup.schedules')).toHaveLength(1));

    switchNamespace(TEAM_B);

    await waitFor(() => expect(read('presetId')).toBe(''));
    // Back to the defaults, not to team-b's preset of the same name.
    expect(read('backup.schedules')).toEqual([]);
    expect(read('k8sNamespace')).toBe(TEAM_B);
    expect(result.current.notice.droppedPreset).toEqual({
      name: PRESET_NAME,
      wizardNamespace: TEAM_B,
      ownerNamespace: TEAM_A,
    });
  });

  it('keeps a cluster preset through a namespace switch', async () => {
    const { result, read, pickPreset, switchNamespace } = setup();
    pickPreset(clusterPreset.id);
    await waitFor(() =>
      expect(result.current.selection.selectedPreset?.scope).toBe(
        PresetScope.Cluster
      )
    );

    switchNamespace(TEAM_B);

    await waitFor(() => expect(requestedNamespaces).toContain(TEAM_B));
    expect(read('presetId')).toBe(clusterPreset.id);
    expect(result.current.notice.droppedPreset).toBeNull();
  });

  describe('the notice about a dropped preset', () => {
    it('outlives the drop itself', async () => {
      const { result, read, dropNamespacePreset } = setup();

      await dropNamespacePreset();

      // The pick is gone from the form by now; the notice must not go with it.
      await waitFor(() => expect(read('presetId')).toBe(''));
      expect(result.current.notice.droppedPreset?.name).toBe(PRESET_NAME);
    });

    it('ends with the next pick', async () => {
      const { result, pickPreset, dropNamespacePreset } = setup();
      await dropNamespacePreset();

      pickPreset(clusterPreset.id);

      await waitFor(() =>
        expect(result.current.notice.droppedPreset).toBeNull()
      );
    });

    it('ends when the namespace changes, and does not come back with it', async () => {
      const { result, switchNamespace, dropNamespacePreset } = setup();
      await dropNamespacePreset();

      switchNamespace(TEAM_A);
      await waitFor(() =>
        expect(result.current.notice.droppedPreset).toBeNull()
      );

      switchNamespace(TEAM_B);
      expect(result.current.notice.droppedPreset).toBeNull();
    });

    it('ends when the user dismisses it', async () => {
      const { result, dropNamespacePreset } = setup();
      await dropNamespacePreset();

      act(() => result.current.notice.dismissDroppedPreset());

      expect(result.current.notice.droppedPreset).toBeNull();
    });
  });
});
