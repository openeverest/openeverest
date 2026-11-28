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

import { describe, expect, it } from 'vitest';
import { FieldType, type Component } from '../../ui-generator.types';
import { generateFieldId } from './generate-field-id';

const idOnlyField = (id: string): Component =>
  ({
    uiType: FieldType.Text,
    id,
    fieldParams: { label: id },
  }) as Component;

describe('generateFieldId', () => {
  it('encodes unsafe characters in each generated name segment', () => {
    expect(
      generateFieldId(idOnlyField('volume[0]'), ['basic', 'volume[0]'])
    ).toBe('g-basic.volume%5B0%5D');
  });

  it('encodes quotes, backslashes, and other RHF-significant characters', () => {
    expect(
      generateFieldId(idOnlyField(`a'"\\b!~`), ['basic', `a'"\\b!~`])
    ).toBe('g-basic.a%27%22%5Cb%21%7E');
  });

  it('distinguishes literal dots from hierarchy separators', () => {
    const literalDot = generateFieldId(idOnlyField('literal.dot'), [
      'basic',
      'literal.dot',
    ]);
    const hierarchy = generateFieldId(idOnlyField('dot'), [
      'basic',
      'literal',
      'dot',
    ]);

    expect(literalDot).toBe('g-basic.literal%2Edot');
    expect(hierarchy).toBe('g-basic.literal.dot');
    expect(literalDot).not.toBe(hierarchy);
  });

  it('does not collide with literal percent-encoded-looking names', () => {
    const dotted = generateFieldId(idOnlyField('gpu.limit'), [
      'basic',
      'gpu.limit',
    ]);
    const percentSequence = generateFieldId(idOnlyField('gpu%2Elimit'), [
      'basic',
      'gpu%2Elimit',
    ]);

    expect(dotted).toBe('g-basic.gpu%2Elimit');
    expect(percentSequence).toBe('g-basic.gpu%252Elimit');
    expect(dotted).not.toBe(percentSequence);
  });

  it('encodes numeric and reserved object-key segments', () => {
    expect(generateFieldId(idOnlyField('0'), ['basic', '0'])).toBe(
      'g-basic.%30'
    );
    expect(
      generateFieldId(idOnlyField('__proto__'), ['basic', '__proto__'])
    ).toBe('g-basic.%5F_proto__');
    expect(
      generateFieldId(idOnlyField('constructor'), ['basic', 'constructor'])
    ).toBe('g-basic.%63onstructor');
  });

  it('keeps explicit API paths on the existing form-id conversion', () => {
    const apiBackedField = {
      ...idOnlyField('gpu'),
      path: "spec.resources.limits['nvidia.com/gpu']",
    } as Component;

    expect(generateFieldId(apiBackedField, ['basic', 'gpu'])).toBe(
      'spec.resources.limits.nvidia%2Ecom%2Fgpu'
    );
  });
});
