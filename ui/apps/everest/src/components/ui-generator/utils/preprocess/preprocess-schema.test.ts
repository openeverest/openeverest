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

import { afterEach, describe, expect, it, vi } from 'vitest';
import {
  Component,
  DataSource,
  FieldType,
  FormMode,
  SelectFieldParams,
  TopologyUISchemas,
} from '../../ui-generator.types';
import { Provider } from 'shared-types/api.types';
import { isSelectComponent } from '../../ui-component/utils/select-component-handler';
import { extractInstanceValues } from '../default-values/extract-instance-values';
import { preprocessSchema } from './preprocess-schema';

// Suppress console.warn from dev-time validation when using test providers
vi.mock('../../api-providers', async () => {
  const actual = await vi.importActual<typeof import('../../api-providers')>(
    '../../api-providers'
  );
  return {
    ...actual,
    providerRegistry: {
      ...actual.providerRegistry,
      has: () => true,
      getAvailableKeys: () => ['storageClasses'],
    },
  };
});

describe('preprocessSchema', () => {
  it('normalizes path metadata even without provider object', () => {
    const schema: TopologyUISchemas = {
      single: {
        sections: {
          base: {
            components: {
              version: {
                uiType: FieldType.Text,
                path: ['spec.engine.version', 'spec.proxy.version'],
                fieldParams: { label: 'Version' },
              },
            },
          },
        },
      },
    };

    const result = preprocessSchema(schema);
    const component = result.single.sections.base.components.version;

    if ('uiType' in component && !('components' in component)) {
      const leaf = component as Component;
      expect(leaf._normalized?.sourcePath).toBe('spec.engine.version');
      expect(leaf._normalized?.targetPaths).toEqual([
        'spec.engine.version',
        'spec.proxy.version',
      ]);
    }
  });

  it('preserves dataSource through preprocessing', () => {
    const dataSource: DataSource = {
      provider: 'storageClasses',
    };
    const schema: TopologyUISchemas = {
      single: {
        sections: {
          storage: {
            components: {
              storageClass: {
                uiType: FieldType.Select,
                dataSource,
                path: 'spec.components.engine.storage.storageClass',
                fieldParams: { label: 'Storage class' },
              },
            },
          },
        },
      },
    };

    const result = preprocessSchema(schema);

    expect(
      result.single.sections.storage.components.storageClass
    ).toMatchObject({ uiType: FieldType.Select, dataSource });
  });

  it('applies the form mode, so restore leaves out the source value of a field hidden in it', () => {
    const schema: TopologyUISchemas = {
      single: {
        sections: {
          base: {
            components: {
              users: {
                uiType: FieldType.Text,
                path: 'spec.userSecretRef.name',
                fieldParams: { label: 'Users' },
                modes: { [FormMode.Restore]: { uiType: 'hidden' } },
              },
              replicas: {
                uiType: FieldType.Number,
                path: 'spec.replicas',
                fieldParams: { label: 'Replicas' },
              },
            },
          },
        },
      },
    };
    const source = { spec: { userSecretRef: { name: 'users' }, replicas: 3 } };

    const { sections } = preprocessSchema(
      schema,
      undefined,
      FormMode.Restore
    ).single;
    const values = extractInstanceValues(sections, source, FormMode.Restore);

    expect(values).toHaveProperty('spec.replicas', 3);
    expect(values).not.toHaveProperty('spec.userSecretRef.name', 'users');
  });

  describe('optionsPath select pre-selection', () => {
    const versions = [{ name: '8.0' }, { name: '8.4' }];
    const provider: Provider = {
      spec: { versions, defaultVersion: '8.4' },
    };

    const versionPicker = (
      fieldParams: Partial<
        Extract<SelectFieldParams, { optionsPath: string }>
      > = {}
    ): TopologyUISchemas => ({
      single: {
        sections: {
          base: {
            components: {
              version: {
                uiType: FieldType.Select,
                path: 'spec.version',
                fieldParams: {
                  label: 'Version',
                  optionsPath: 'spec.versions',
                  optionsPathConfig: {
                    labelPath: 'name',
                    valuePath: 'name',
                    defaultValuePath: 'spec.defaultVersion',
                  },
                  ...fieldParams,
                },
              },
            },
          },
        },
      },
    });

    const defaultOf = (
      schema: TopologyUISchemas,
      withProvider: Provider = provider,
      mode: FormMode = FormMode.New
    ) => {
      const component = preprocessSchema(schema, withProvider, mode).single
        .sections.base.components.version as Component;
      if (!isSelectComponent(component)) {
        throw new Error('version is not a select component');
      }
      return component.fieldParams.defaultValue;
    };

    afterEach(() => {
      vi.restoreAllMocks();
    });

    it('pre-selects the option named by defaultValuePath on create', () => {
      expect(defaultOf(versionPicker())).toBe('8.4');
    });

    it('lets the schema defaultValue win over the provider default', () => {
      expect(defaultOf(versionPicker({ defaultValue: '8.0' }))).toBe('8.0');
    });

    it('falls back to the first option when defaultValuePath is absent', () => {
      const schema = versionPicker({
        optionsPathConfig: { labelPath: 'name', valuePath: 'name' },
      });
      expect(defaultOf(schema)).toBe('8.0');
    });

    it('falls back to the first option, with a warning, when the provider default is not an option', () => {
      const warn = vi.spyOn(console, 'warn').mockImplementation(() => {});
      const stale: Provider = {
        spec: { versions, defaultVersion: '9.9' },
      };

      expect(defaultOf(versionPicker(), stale)).toBe('8.0');
      expect(warn).toHaveBeenCalledWith(
        expect.stringContaining('"base.version" names default "9.9"')
      );
    });

    it('falls back to the first option without a warning when the provider declares no default', () => {
      const warn = vi.spyOn(console, 'warn').mockImplementation(() => {});
      const noDefault: Provider = { spec: { versions } };

      expect(defaultOf(versionPicker(), noDefault)).toBe('8.0');
      expect(warn).not.toHaveBeenCalled();
    });

    it('matches the provider default against option values, not the field it names', () => {
      vi.spyOn(console, 'warn').mockImplementation(() => {});
      const byEngine: Provider = {
        spec: {
          versions: [
            { name: '8.0', components: { engine: '8.0.1-1' } },
            { name: '8.4', components: { engine: '8.4.1-1' } },
          ],
          defaultVersion: '8.4',
        },
      };
      const schema = versionPicker({
        optionsPathConfig: {
          labelPath: 'name',
          valuePath: 'components.engine',
          defaultValuePath: 'spec.defaultVersion',
        },
      });

      expect(defaultOf(schema, byEngine)).toBe('8.0.1-1');
    });

    it('ignores the provider default outside create, so restore keeps the first-option fallback', () => {
      expect(defaultOf(versionPicker(), provider, FormMode.Restore)).toBe(
        '8.0'
      );
      expect(defaultOf(versionPicker(), provider, FormMode.Edit)).toBe('8.0');
      expect(defaultOf(versionPicker(), provider, FormMode.Import)).toBe('8.0');
    });
  });
});
