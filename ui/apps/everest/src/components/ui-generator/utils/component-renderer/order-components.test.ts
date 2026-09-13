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
import { orderComponents } from './order-components';
import { FieldType } from '../../ui-generator.types';
import type { Component } from '../../ui-generator.types';

const makeField = (path: string): Component => ({
  uiType: FieldType.Text,
  path,
  fieldParams: { label: path },
});

const makeComponents = () => ({
  a: makeField('spec.a'),
  b: makeField('spec.b'),
  c: makeField('spec.c'),
});

describe('orderComponents', () => {
  it('returns original order when no componentsOrder is given', () => {
    const result = orderComponents(makeComponents());
    expect(result.map(([k]) => k)).toEqual(['a', 'b', 'c']);
  });

  it('returns original order when componentsOrder is an empty array', () => {
    const result = orderComponents(makeComponents(), []);
    expect(result.map(([k]) => k)).toEqual(['a', 'b', 'c']);
  });

  it('reorders components according to componentsOrder', () => {
    const result = orderComponents(makeComponents(), ['c', 'a', 'b']);
    expect(result.map(([k]) => k)).toEqual(['c', 'a', 'b']);
  });

  it('appends unordered components after ordered ones', () => {
    const result = orderComponents(makeComponents(), ['c']);
    expect(result.map(([k]) => k)).toEqual(['c', 'a', 'b']);
  });

  it('silently skips keys in componentsOrder that do not exist', () => {
    const result = orderComponents(makeComponents(), ['z', 'b', 'a']);
    expect(result.map(([k]) => k)).toEqual(['b', 'a', 'c']);
  });

  it('preserves the component value at each key', () => {
    const components = makeComponents();
    const result = orderComponents(components, ['b']);
    const bEntry = result.find(([k]) => k === 'b');
    expect(bEntry?.[1]).toBe(components.b);
  });
});
