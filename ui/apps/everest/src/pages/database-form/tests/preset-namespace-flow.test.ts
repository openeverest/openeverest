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
import { ClusterInstancePreset, InstancePreset } from 'shared-types/api.types';
import { DbWizardType } from '../database-form-schema';
import { useDatabaseFormSync } from '../hooks/use-database-form-sync';
import {
  getClusterPresetId,
  getNamespacePresetId,
  usePresetSelection,
} from '../preset-selection';

// The wizard page wiring, without the page: the namespace default, the preset
// selection and the form sync only work together, so they are run together.
const queries = vi.hoisted(() => {
  const settled = { isLoading: false, isFetching: false, isError: false };
  const spec = (version: string) => ({
    providerRef: { name: 'psmdb' },
    version,
    topology: { type: 'replicaSet' },
  });
  const clusterPreset: ClusterInstancePreset = {
    metadata: { name: 'psmdb-default' },
    spec: spec('8.0.12'),
  };
  const teamAPreset: InstancePreset = {
    metadata: { name: 'team-default', namespace: 'team-a' },
    spec: {
      ...spec('7.0.18'),
      backup: {
        enabled: true,
        classRef: { name: 'psmdb-backup' },
        storages: [
          {
            storageRef: { name: 'bs-team-a' },
            schedules: [{ name: 'daily', cron: '0 0 * * *', enabled: true }],
          },
        ],
      },
    },
  };
  // Same name as in team-a, on purpose: it must never be picked up silently.
  const teamBPreset: InstancePreset = {
    metadata: { name: 'team-default', namespace: 'team-b' },
    spec: spec('6.0.21'),
  };
  const namespaceLists: Record<string, object> = {
    'team-a': { data: [teamAPreset], ...settled },
    'team-b': { data: [teamBPreset], ...settled },
  };
  return {
    clusterList: { data: [clusterPreset], ...settled },
    namespaceLists,
    noList: { data: undefined, ...settled },
    requestedNamespaces: new Set<string>(),
  };
});

vi.mock('hooks/api/cluster-instance-presets', () => ({
  useClusterInstancePresets: () => queries.clusterList,
}));
vi.mock('hooks/api/instance-presets', () => ({
  useInstancePresets: (_clusterName: string, namespace: string) => {
    queries.requestedNamespaces.add(namespace);
    return queries.namespaceLists[namespace] ?? queries.noList;
  },
}));

const uiSchema: TopologyUISchemas = { replicaSet: { sections: {} } };
const FORM_DEFAULTS = { backup: { schedules: [] } };
const CLUSTER_PRESET_ID = getClusterPresetId('psmdb-default');
const TEAM_A_PRESET_ID = getNamespacePresetId('team-a', 'team-default');

const setup = () => {
  const view = renderHook(() => {
    const methods = useForm<DbWizardType>({
      defaultValues: {
        provider: 'psmdb',
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
      'psmdb',
      selectedNamespace || '',
      presetId || ''
    );
    useDatabaseFormSync({
      mode: FormMode.New,
      uiSchema,
      defaultValues: FORM_DEFAULTS,
      defaultTopology: 'replicaSet',
      selectedTopology: 'replicaSet',
      defaultNamespace: 'team-a',
      preset: selection,
      reset,
      getValues,
      setValue,
    });
    return { methods, selection };
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

  return { ...view, read, pickPreset, switchNamespace };
};

describe('presets across namespaces in the create wizard', () => {
  beforeEach(() => {
    queries.requestedNamespaces.clear();
  });

  it('lists the presets of the namespace the wizard defaults to', () => {
    const { result } = setup();

    expect(queries.requestedNamespaces).toContain('team-a');
    expect(result.current.selection.presets.map(({ id }) => id)).toEqual([
      CLUSTER_PRESET_ID,
      TEAM_A_PRESET_ID,
    ]);
  });

  it('drops a namespace preset on a namespace switch and keeps the new namespace', async () => {
    const { result, read, pickPreset, switchNamespace } = setup();
    pickPreset(TEAM_A_PRESET_ID);
    await waitFor(() => expect(read('backup.schedules')).toHaveLength(1));

    switchNamespace('team-b');

    await waitFor(() => expect(read('presetId')).toBe(''));
    // Back to the defaults, not to team-b's preset of the same name.
    expect(read('backup.schedules')).toEqual([]);
    expect(read('k8sNamespace')).toBe('team-b');
    expect(result.current.selection.droppedPreset).toEqual({
      name: 'team-default',
      namespace: 'team-b',
      presetNamespace: 'team-a',
    });
  });

  it('keeps a cluster preset through a namespace switch', async () => {
    const { result, read, pickPreset, switchNamespace } = setup();
    pickPreset(CLUSTER_PRESET_ID);
    await waitFor(() =>
      expect(result.current.selection.selectedPreset?.name).toBe(
        'psmdb-default'
      )
    );

    switchNamespace('team-b');

    await waitFor(() =>
      expect(queries.requestedNamespaces).toContain('team-b')
    );
    expect(read('presetId')).toBe(CLUSTER_PRESET_ID);
    expect(result.current.selection.droppedPreset).toBeNull();
  });
});
