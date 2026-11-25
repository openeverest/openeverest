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

import { renderHook, waitFor } from '@testing-library/react';
import { UseFormGetValues, UseFormReset, useForm } from 'react-hook-form';
import {
  Component,
  FieldType,
  FormMode,
  TopologyUISchemas,
} from 'components/ui-generator/ui-generator.types';
import { getByPath } from 'components/ui-generator/utils/object-path/object-path';
import { DbWizardType } from '../../database-form-schema';
import {
  TEAM_A,
  clusterPreset as defaultClusterPreset,
  teamAPreset,
} from '../__mocks__/presets';
import { PresetStatus, SelectablePreset } from '../preset-selection.types';
import { usePresetFormSync } from './use-preset-form-sync';

const makeComponent = (path: string, defaultValue?: unknown): Component =>
  ({
    uiType: FieldType.Text,
    path,
    fieldParams: {
      label: path,
      ...(defaultValue !== undefined && { defaultValue }),
    },
    _normalized: { sourcePath: path, targetPaths: [path] },
  }) as Component;

// cpu: preset sets it. storageClass: schema has a UI default but the preset
// overrides it. extra: schema has a default the preset omits (scaffold gap).
const uiSchema = {
  replicaSet: {
    sections: {
      resources: {
        components: {
          cpu: makeComponent('spec.components.engine.resources.limits.cpu', 1),
          storageClass: makeComponent(
            'spec.components.engine.storage.storageClass',
            'ui-default-sc'
          ),
          extra: makeComponent(
            'spec.components.engine.extra',
            'scaffold-default'
          ),
        },
      },
    },
  },
} as unknown as TopologyUISchemas;

// Differs from the schema defaults above in cpu and storageClass, so a test
// can tell which of the two won.
const clusterPreset: SelectablePreset = {
  ...defaultClusterPreset,
  spec: {
    ...defaultClusterPreset.spec,
    components: {
      engine: {
        resources: { limits: { cpu: 2 } },
        storage: { size: '25Gi', storageClass: 'preset-sc' },
      },
    },
  },
};

const CPU_PATH = 'spec.components.engine.resources.limits.cpu';
const SC_PATH = 'spec.components.engine.storage.storageClass';
const EXTRA_PATH = 'spec.components.engine.extra';

interface HookArgs {
  mode?: FormMode;
  presetId?: string;
  preset?: SelectablePreset | null;
  status?: PresetStatus;
  namespace?: string;
}

const setup = (args: HookArgs = {}) => {
  const utils = renderHook(
    ({ mode, presetId, preset, status, namespace }: Required<HookArgs>) => {
      const methods = useForm({
        defaultValues: {
          provider: 'psmdb',
          dbName: 'my-db',
          k8sNamespace: 'ns-1',
          presetId: '',
          topology: { type: 'replicaSet' },
          backup: { enabled: false },
          spec: {},
        },
      });
      const notice = usePresetFormSync({
        mode,
        uiSchema,
        defaultValues: { backup: { enabled: false } },
        defaultTopology: 'replicaSet',
        selectedPreset: preset,
        presetId,
        status,
        namespace,
        reset: methods.reset as unknown as UseFormReset<DbWizardType>,
        getValues:
          methods.getValues as unknown as UseFormGetValues<DbWizardType>,
      });
      return { methods, notice };
    },
    {
      initialProps: {
        mode: args.mode ?? FormMode.New,
        presetId: args.presetId ?? clusterPreset.id,
        preset: args.preset === undefined ? clusterPreset : args.preset,
        status: args.status ?? 'ready',
        namespace: args.namespace ?? 'ns-1',
      },
    }
  );
  const read = (path: string): unknown =>
    getByPath(
      utils.result.current.methods.getValues() as Record<string, unknown>,
      path
    );
  return { ...utils, read };
};

describe('usePresetFormSync', () => {
  it('populates the form with the preset value (C1)', async () => {
    const { read } = setup();

    await waitFor(() => expect(read(CPU_PATH)).toBe(2));
  });

  it('lets the preset value override the schema UI default (I2)', async () => {
    const { read } = setup();

    // storageClass has a UI default ("ui-default-sc") but the preset sets it —
    // the preset must win, never the schema default.
    await waitFor(() => expect(read(SC_PATH)).toBe('preset-sc'));
  });

  it('keeps a valid scaffold default for fields the preset omits', async () => {
    const { read } = setup();

    // The preset omits `extra`; it must stay at its empty scaffold default
    // (not become undefined, which would break form validation).
    await waitFor(() => expect(read(CPU_PATH)).toBe(2));
    expect(read(EXTRA_PATH)).toBe('scaffold-default');
  });

  it('keeps the user-chosen dbName and namespace (meta fields)', async () => {
    const { read } = setup();

    await waitFor(() => expect(read(CPU_PATH)).toBe(2));
    expect(read('dbName')).toBe('my-db');
    expect(read('k8sNamespace')).toBe('ns-1');
    expect(read('presetId')).toBe(clusterPreset.id);
  });

  // The preview reads the form, while the request sends the preset spec: both
  // must describe the same backups and monitoring.
  it('shows the preset backups and monitoring in the wizard form', async () => {
    const { read } = setup({ preset: teamAPreset, presetId: teamAPreset.id });

    await waitFor(() =>
      expect(read('backup.schedules')).toEqual([
        {
          name: 'daily',
          cron: '0 0 * * *',
          enabled: true,
          storageName: 'bs-team-a',
        },
      ])
    );
    expect(read('backup.classRef.name')).toBe('psmdb-backup-class');
    expect(read('backup.pitr')).toEqual({ 'bs-team-a': { enabled: true } });
    expect(read('monitoring.destinations')).toEqual(
      teamAPreset.spec.monitoring?.destinations
    );
  });

  it('shows no backups or monitoring for a preset that sets none', async () => {
    const { read } = setup();

    await waitFor(() => expect(read(CPU_PATH)).toBe(2));
    expect(read('backup.schedules')).toEqual([]);
    expect(read('monitoring.destinations')).toEqual([]);
  });

  it('reverts to defaults when the preset is cleared (None)', async () => {
    const { read, rerender } = setup();

    await waitFor(() => expect(read(CPU_PATH)).toBe(2));

    rerender({
      mode: FormMode.New,
      presetId: '',
      preset: null,
      status: 'none',
      namespace: 'ns-1',
    });

    await waitFor(() => expect(read('presetId')).toBe(''));
    expect(read(CPU_PATH)).toBeUndefined();
  });

  it('drops a preset that became unavailable, keeping the chosen namespace', async () => {
    const { read, rerender, result } = setup({
      preset: teamAPreset,
      presetId: teamAPreset.id,
    });

    await waitFor(() => expect(read('backup.schedules')).toHaveLength(1));
    expect(read(CPU_PATH)).toBe(1);

    // Namespace switched: the pick is still in the form but no longer resolves.
    rerender({
      mode: FormMode.New,
      presetId: teamAPreset.id,
      preset: null,
      status: 'unavailable',
      namespace: 'ns-1',
    });

    await waitFor(() => expect(read('presetId')).toBe(''));
    expect(read(CPU_PATH)).toBeUndefined();
    expect(read('backup.schedules')).toBeUndefined();
    expect(read('k8sNamespace')).toBe('ns-1');
    expect(read('dbName')).toBe('my-db');
    // The drop and the notice about it come from the same place.
    expect(result.current.notice.droppedPreset).toEqual({
      name: teamAPreset.name,
      wizardNamespace: 'ns-1',
      ownerNamespace: TEAM_A,
    });
  });

  it('does not report a preset the user cleared', async () => {
    const { read, rerender, result } = setup();
    await waitFor(() => expect(read(CPU_PATH)).toBe(2));

    rerender({
      mode: FormMode.New,
      presetId: '',
      preset: null,
      status: 'none',
      namespace: 'ns-1',
    });

    await waitFor(() => expect(read(CPU_PATH)).toBeUndefined());
    expect(result.current.notice.droppedPreset).toBeNull();
  });

  it('does nothing outside New mode', async () => {
    const { read } = setup({ mode: FormMode.Edit });

    // Give the effect a chance to run; it must not touch the form.
    await new Promise((r) => setTimeout(r, 0));
    expect(read(CPU_PATH)).toBeUndefined();
  });
});
