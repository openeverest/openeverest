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
import { generateFieldId } from './generate-field-id';
import { FieldType } from '../../ui-generator.types';
import type { Component, ComponentGroup } from '../../ui-generator.types';

describe('generateFieldId', () => {
  it('returns the component path when path is a plain string', () => {
    const item: Component = {
      uiType: FieldType.Text,
      path: 'spec.proxy.replicas',
      fieldParams: { label: 'Replicas' },
    };
    expect(generateFieldId(item, 'myField')).toBe('spec.proxy.replicas');
  });

  it('returns g-{name} fallback for a group (no path)', () => {
    const item: ComponentGroup = {
      uiType: 'group',
      components: {},
    };
    expect(generateFieldId(item, 'resources')).toBe('g-resources');
  });

  it('returns g-{name} fallback for a hidden field (no path resolution)', () => {
    const item: Component = {
      uiType: 'hidden',
      path: 'spec.hidden',
      fieldParams: {},
    } as unknown as Component;
    expect(generateFieldId(item, 'hiddenField')).toBe('g-hiddenField');
  });

  it('returns g-{name} when component has no path', () => {
    const item = {
      uiType: FieldType.Text,
      fieldParams: { label: 'No path' },
    } as unknown as Component;
    expect(generateFieldId(item, 'noPath')).toBe('g-noPath');
  });
});
