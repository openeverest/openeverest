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
import { UseFormGetValues, UseFormReset, useForm } from 'react-hook-form';
import { vi } from 'vitest';
import {
  Component,
  FieldType,
  FormMode,
  TopologyUISchemas,
} from 'components/ui-generator/ui-generator.types';
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
  resolvedPreset: null,
  presetName: '',
  presetSelected: false,
  namespace: 'ns-1',
};

const setup = () =>
  renderHook(
    ({ selectedTopology }: { selectedTopology: string }) => {
      const methods = useForm({
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
        preset: NO_PRESET,
        reset: methods.reset as unknown as UseFormReset<DbWizardType>,
        getValues:
          methods.getValues as unknown as UseFormGetValues<DbWizardType>,
      });
      return methods;
    },
    { initialProps: { selectedTopology: 'cluster' } }
  );

describe('useDatabaseFormSync topology switch', () => {
  it('drops the previous topology components and applies the new defaults', () => {
    const { result, rerender } = setup();

    rerender({ selectedTopology: 'standalone' });

    expect(result.current.getValues('spec')).toEqual({
      components: { standalone: { replicas: 1 }, mixCoord: {} },
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
