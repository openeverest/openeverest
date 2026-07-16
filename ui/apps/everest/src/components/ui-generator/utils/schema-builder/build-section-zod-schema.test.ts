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

import { describe, it, expect } from 'vitest';
import { buildSectionZodSchema } from './build-section-zod-schema';
import type {
  Section,
  Component,
} from 'components/ui-generator/ui-generator.types';
import { FieldType } from 'components/ui-generator/ui-generator.types';

const makeTextComponent = (
  path: string,
  overrides: Partial<Component> = {}
): Component =>
  ({
    uiType: FieldType.Text,
    path,
    fieldParams: { label: path },
    ...overrides,
  }) as Component;

const makeNumberComponent = (
  path: string,
  overrides: Partial<Component> = {}
): Component =>
  ({
    uiType: FieldType.Number,
    path,
    fieldParams: { label: path },
    ...overrides,
  }) as Component;

describe('buildSectionZodSchema', () => {
  it('returns passthrough schema for missing section', () => {
    const { schema } = buildSectionZodSchema('missing', {});
    const result = schema.safeParse({ anything: 'goes' });
    expect(result.success).toBe(true);
  });

  it('validates only the target section fields', () => {
    const sections: Record<string, Section> = {
      basic: {
        components: {
          name: makeTextComponent('spec.name', {
            validation: { required: true },
          }),
        },
      },
      resources: {
        components: {
          cpu: makeNumberComponent('spec.cpu', {
            validation: { required: true, min: 1 },
          }),
        },
      },
    };

    const { schema } = buildSectionZodSchema('basic', sections);

    // Valid: basic section field present, resources field missing but passthrough allows it
    const valid = schema.safeParse({ spec: { name: 'test' } });
    expect(valid.success).toBe(true);

    // Invalid: basic section required field missing
    const invalid = schema.safeParse({ spec: { name: '' } });
    expect(invalid.success).toBe(false);
  });

  it('allows extra keys via passthrough', () => {
    const sections: Record<string, Section> = {
      basic: {
        components: {
          name: makeTextComponent('spec.name'),
        },
      },
    };

    const { schema } = buildSectionZodSchema('basic', sections);
    const result = schema.safeParse({
      spec: { name: 'test', extraField: 'value' },
      other: 123,
    });
    expect(result.success).toBe(true);
  });
});
