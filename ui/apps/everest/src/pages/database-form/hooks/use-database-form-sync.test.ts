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

import { act, renderHook } from '@testing-library/react';
import { useForm, useWatch } from 'react-hook-form';
import { vi } from 'vitest';
import {
  Component,
  FieldType,
  FormMode,
  TopologyUISchemas,
} from 'components/ui-generator/ui-generator.types';
import { DbWizardFormFields } from 'consts';
import { DbWizardType } from '../database-form-schema';
import { useDatabaseFormSync } from './use-database-form-sync';

vi.mock('../preset-selection', () => ({ usePresetFormSync: vi.fn() }));

const numberField = (path: string, defaultValue: number): Component => ({
  uiType: FieldType.Number,
  path,
  fieldParams: { label: path, defaultValue },
});

const uiSchema: TopologyUISchemas = {
  standalone: {
    sections: {
      resources: {
        components: {
          replicas: numberField('spec.components.standalone.replicas', 1),
        },
      },
    },
  },
  cluster: {
    sections: {
      resources: {
        components: {
          coordinator: numberField('spec.components.mixCoord.replicas', 1),
        },
      },
    },
  },
};

const NO_PRESET = {
  selectedPreset: null,
  presetId: '',
  presetSelected: false,
  presetUnavailable: false,
};

const setup = () =>
  renderHook(
    ({ selectedTopology }: { selectedTopology: string }) => {
      const methods = useForm<DbWizardType>({
        defaultValues: {
          dbName: 'my-db',
          topology: { type: 'cluster' },
          spec: { components: { mixCoord: { replicas: 1 } } },
        },
      });
      useDatabaseFormSync({
        mode: FormMode.New,
        uiSchema,
        defaultValues: {},
        defaultTopology: 'cluster',
        selectedTopology,
        defaultNamespace: undefined,
        preset: NO_PRESET,
        reset: methods.reset,
        getValues: methods.getValues,
        setValue: methods.setValue,
      });
      return methods;
    },
    { initialProps: { selectedTopology: 'cluster' } }
  );

interface NamespaceProps {
  mode: FormMode;
  defaultNamespace: string | undefined;
}

const setupNamespace = (
  initialProps: NamespaceProps = {
    mode: FormMode.New,
    defaultNamespace: 'team-a',
  }
) =>
  renderHook(
    ({ mode, defaultNamespace }: NamespaceProps) => {
      const methods = useForm<DbWizardType>({
        defaultValues: {
          dbName: 'my-db',
          k8sNamespace: '',
          topology: { type: 'cluster' },
        },
      });
      // Subscribed before the sync runs, the way the wizard page reads it.
      const watchedNamespace = useWatch({
        control: methods.control,
        name: DbWizardFormFields.k8sNamespace,
      });
      useDatabaseFormSync({
        mode,
        uiSchema,
        defaultValues: {},
        defaultTopology: 'cluster',
        selectedTopology: 'cluster',
        defaultNamespace,
        preset: NO_PRESET,
        reset: methods.reset,
        getValues: methods.getValues,
        setValue: methods.setValue,
      });
      return { methods, watchedNamespace };
    },
    { initialProps }
  );

describe('useDatabaseFormSync topology switch', () => {
  it('drops the previous topology components and applies the new defaults', () => {
    const { result, rerender } = setup();

    rerender({ selectedTopology: 'standalone' });

    expect(
      result.current.getValues('spec.components.mixCoord.replicas')
    ).toBeUndefined();
    expect(result.current.getValues('spec.components')).toMatchObject({
      standalone: { replicas: 1 },
    });
    expect(result.current.getValues('dbName')).toBe('my-db');
  });

  it('restores defaults, not stale values, when switching back', () => {
    const { result, rerender } = setup();
    act(() => {
      result.current.setValue('spec.components.mixCoord.replicas', 5);
    });

    rerender({ selectedTopology: 'standalone' });
    rerender({ selectedTopology: 'cluster' });

    expect(result.current.getValues('spec.components.mixCoord.replicas')).toBe(
      1
    );
  });
});

describe('useDatabaseFormSync default namespace', () => {
  it('starts a new instance in the first namespace', () => {
    const { result } = setupNamespace();

    expect(
      result.current.methods.getValues(DbWizardFormFields.k8sNamespace)
    ).toBe('team-a');
  });

  // useWatch ignores a value written before its subscription is in place. The
  // page reads the namespace through useWatch, so a default it cannot see
  // leaves it working with an empty namespace.
  it('writes the default where the page subscription sees it', () => {
    const { result } = setupNamespace();

    expect(result.current.watchedNamespace).toBe('team-a');
  });

  it('waits for the namespaces to load', () => {
    const { result, rerender } = setupNamespace({
      mode: FormMode.New,
      defaultNamespace: undefined,
    });
    expect(
      result.current.methods.getValues(DbWizardFormFields.k8sNamespace)
    ).toBe('');

    rerender({ mode: FormMode.New, defaultNamespace: 'team-a' });

    expect(result.current.watchedNamespace).toBe('team-a');
  });

  it('keeps the chosen namespace when the list gets a new first entry', () => {
    const { result, rerender } = setupNamespace();
    act(() => {
      result.current.methods.setValue(
        DbWizardFormFields.k8sNamespace,
        'team-b'
      );
    });

    rerender({ mode: FormMode.New, defaultNamespace: 'another-team' });

    expect(
      result.current.methods.getValues(DbWizardFormFields.k8sNamespace)
    ).toBe('team-b');
  });

  it.each([FormMode.Restore, FormMode.Import])(
    'leaves the namespace to the source in %s mode',
    (mode) => {
      const { result } = setupNamespace({ mode, defaultNamespace: 'team-a' });

      expect(
        result.current.methods.getValues(DbWizardFormFields.k8sNamespace)
      ).toBe('');
    }
  );
});
