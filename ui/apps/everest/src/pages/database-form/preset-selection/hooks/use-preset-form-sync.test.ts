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
import { InstancePreset } from 'shared-types/api.types';
import { DbWizardType } from '../../database-form-schema';
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

const resolvedPreset: InstancePreset = {
  spec: {
    version: '8.0.12',
    topology: { type: 'replicaSet' },
    components: {
      engine: {
        resources: { limits: { cpu: 2 } },
        storage: { storageClass: 'preset-sc' },
      },
    },
  },
} as unknown as InstancePreset;

const CPU_PATH = 'spec.components.engine.resources.limits.cpu';
const SC_PATH = 'spec.components.engine.storage.storageClass';
const EXTRA_PATH = 'spec.components.engine.extra';

interface HookArgs {
  mode?: FormMode;
  presetName?: string;
  namespace?: string;
  preset?: InstancePreset | null;
}

const setup = (args: HookArgs = {}) => {
  const utils = renderHook(
    ({ mode, presetName, namespace, preset }: Required<HookArgs>) => {
      const methods = useForm({
        defaultValues: {
          provider: 'psmdb',
          dbName: 'my-db',
          k8sNamespace: 'ns-1',
          presetName: '',
          topology: { type: 'replicaSet' },
          backup: { enabled: false },
          spec: {},
        },
      });
      usePresetFormSync({
        mode,
        uiSchema,
        defaultValues: { backup: { enabled: false } },
        defaultTopology: 'replicaSet',
        resolvedPreset: preset,
        presetName,
        namespace,
        reset: methods.reset as unknown as UseFormReset<DbWizardType>,
        getValues:
          methods.getValues as unknown as UseFormGetValues<DbWizardType>,
      });
      return methods;
    },
    {
      initialProps: {
        mode: args.mode ?? FormMode.New,
        presetName: args.presetName ?? 'psmdb-replicaset',
        namespace: args.namespace ?? 'ns-1',
        preset: args.preset === undefined ? resolvedPreset : args.preset,
      },
    }
  );
  const read = (path: string): unknown =>
    getByPath(
      utils.result.current.getValues() as Record<string, unknown>,
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
    expect(read('presetName')).toBe('psmdb-replicaset');
  });

  it('reverts to defaults when the preset is cleared (None)', async () => {
    const { read, rerender } = setup();

    await waitFor(() => expect(read(CPU_PATH)).toBe(2));

    rerender({
      mode: FormMode.New,
      presetName: '',
      namespace: 'ns-1',
      preset: null,
    });

    await waitFor(() => expect(read('presetName')).toBe(''));
    expect(read(CPU_PATH)).toBeUndefined();
  });

  it('does nothing outside New mode', async () => {
    const { read } = setup({ mode: FormMode.Edit });

    // Give the effect a chance to run; it must not touch the form.
    await new Promise((r) => setTimeout(r, 0));
    expect(read(CPU_PATH)).toBeUndefined();
  });
});
