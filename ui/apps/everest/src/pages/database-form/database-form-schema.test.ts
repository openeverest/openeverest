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

import { z } from 'zod';
import {
  FieldType,
  FormMode,
  TopologyUISchemas,
} from 'components/ui-generator/ui-generator.types';
import { preprocessSchema } from 'components/ui-generator/utils/preprocess/preprocess-schema';
import {
  buildSectionZodSchema,
  buildZodSchema,
} from 'components/ui-generator/utils/schema-builder';
import { mergeSectionEdit } from 'components/ui-generator/utils/postprocess/merge-section-edit';
import { postprocessSchemaData } from 'components/ui-generator/utils/postprocess/postprocess-schema';
import { getDBWizardSchema } from './database-form-schema';
import { Messages } from './database-form.messages';

const BASE_VALUES = {
  provider: 'test-provider',
  dbName: 'test-instance',
  k8sNamespace: 'default',
  topology: { type: 'single' },
};

const UI_SCHEMA: TopologyUISchemas = preprocessSchema(
  {
    single: {
      sections: {
        instance: {
          components: {
            zone: {
              uiType: FieldType.Text,
              path: 'spec.zone',
              fieldParams: { label: 'Zone' },
              validation: { trim: true, toLowerCase: true },
            },
            replicas: {
              uiType: FieldType.Number,
              path: 'spec.replicas',
              fieldParams: { label: 'Replicas' },
              validation: { min: 1 },
            },
          },
        },
      },
    },
  },
  undefined,
  FormMode.New
);

const GENERATED_SCHEMA = buildZodSchema(UI_SCHEMA, 'single').schema;

describe('wizard parsed values', () => {
  it('keeps transformed nested values, coerced numbers, arrays, and unrelated step values without mutating the input', () => {
    const steps = z
      .object({
        backup: z
          .object({
            schedules: z.array(
              z
                .object({
                  name: z.string().trim().toUpperCase(),
                  retention: z.coerce.number(),
                })
                .passthrough()
            ),
          })
          .passthrough(),
      })
      .passthrough();
    const input = {
      ...BASE_VALUES,
      presetId: '',
      spec: {
        zone: ' EU-West ',
        replicas: '3',
        engine: { version: '8.0' },
        monitoring: [
          { name: 'metrics', parameters: { scrapeInterval: '30s' } },
        ],
      },
      backup: {
        classRefName: 's3',
        schedules: [
          { name: ' daily ', retention: '2', storageName: 'primary' },
        ],
      },
      pluginValues: { enabled: true },
    };
    const original = structuredClone(input);
    const parsed = getDBWizardSchema([], GENERATED_SCHEMA, steps).parse(input);
    expect(parsed).toEqual({
      ...BASE_VALUES,
      presetId: '',
      spec: {
        zone: 'eu-west',
        replicas: 3,
        engine: { version: '8.0' },
        monitoring: [
          { name: 'metrics', parameters: { scrapeInterval: '30s' } },
        ],
      },
      backup: {
        classRefName: 's3',
        schedules: [{ name: 'DAILY', retention: 2, storageName: 'primary' }],
      },
      pluginValues: { enabled: true },
    });
    expect(input).toEqual(original);
  });

  it.each([
    { label: 'missing', spec: {} },
    { label: 'undefined', spec: { zone: undefined } },
    { label: 'empty', spec: { zone: '' } },
    { label: 'whitespace', spec: { zone: '   ' } },
  ])('omits an optional $label value from submitted data', ({ spec }) => {
    const parsed: Record<string, unknown> = getDBWizardSchema(
      [],
      GENERATED_SCHEMA
    ).parse({ ...BASE_VALUES, spec });
    expect(
      postprocessSchemaData(parsed, {
        schema: UI_SCHEMA,
        selectedTopology: 'single',
      })
    ).not.toHaveProperty('spec.zone');
  });

  it('rejects null text at the original field path', () => {
    const result = getDBWizardSchema([], GENERATED_SCHEMA).safeParse({
      ...BASE_VALUES,
      spec: { zone: null },
    });
    expect(result.success).toBe(false);
    if (!result.success) {
      expect(result.error.issues).toEqual([
        expect.objectContaining({
          path: ['spec', 'zone'],
          code: 'invalid_type',
        }),
      ]);
    }
  });

  it('keeps name, generated-field, and static-step diagnostics together', () => {
    const generated = z
      .object({
        spec: z
          .object({ zone: z.string().trim().min(1, 'Zone is required') })
          .passthrough(),
      })
      .passthrough();
    const steps = z
      .object({
        backup: z
          .object({ classRefName: z.string().min(1, 'Choose a backup class') })
          .passthrough(),
      })
      .passthrough();
    const schema = getDBWizardSchema(
      [{ name: 'test-instance', namespace: 'default' }],
      generated,
      steps
    );
    const result = schema.safeParse({
      ...BASE_VALUES,
      spec: { zone: ' ' },
      backup: { classRefName: '' },
    });
    expect(result.success).toBe(false);
    if (!result.success) {
      expect(result.error.issues).toEqual([
        expect.objectContaining({
          path: ['dbName'],
          message: Messages.errors.dbName.duplicate,
        }),
        expect.objectContaining({
          path: ['spec', 'zone'],
          message: 'Zone is required',
        }),
        expect.objectContaining({
          path: ['backup', 'classRefName'],
          message: 'Choose a backup class',
        }),
      ]);
    }
  });

  it('applies each schema transform once in composition order', () => {
    const first = vi.fn((value: string) => value + '!');
    const second = vi.fn((value: string) => value + '?');
    const generated = z
      .object({
        spec: z.object({ zone: z.string().transform(first) }).passthrough(),
      })
      .passthrough();
    const steps = z
      .object({
        spec: z.object({ zone: z.string().transform(second) }).passthrough(),
      })
      .passthrough();
    const result = getDBWizardSchema([], generated, steps).parse({
      ...BASE_VALUES,
      spec: { zone: 'west' },
    });
    expect(result).toMatchObject({ spec: { zone: 'west!?' } });
    expect(first).toHaveBeenCalledOnce();
    expect(second).toHaveBeenCalledOnce();
  });

  it('keeps the section-edit transform and merge behavior', () => {
    const sections = UI_SCHEMA.single.sections;
    const { schema } = buildSectionZodSchema('instance', sections, {
      formMode: FormMode.Edit,
    });
    const input = { spec: { zone: ' EU-West ', replicas: '3' } };
    const parsed: Record<string, unknown> = schema.parse(input);
    const updated = mergeSectionEdit({
      spec: { zone: 'east', replicas: 1, engine: { version: '8.0' } },
      formData: parsed,
      sections,
      sectionKey: 'instance',
      topology: 'single',
    });
    expect(updated).toEqual({
      zone: 'eu-west',
      replicas: 3,
      engine: { version: '8.0' },
    });
    expect(input.spec.zone).toBe(' EU-West ');
  });
});
