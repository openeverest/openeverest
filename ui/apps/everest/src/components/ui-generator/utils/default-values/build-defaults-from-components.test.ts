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
import { buildDefaultsFromComponents } from './build-defaults-from-components';
import {
  FieldType,
  type Component,
  type ComponentGroup,
} from '../../ui-generator.types';

const text = (path: string, defaultValue?: string): Component => ({
  uiType: FieldType.Text,
  path,
  fieldParams: { label: path, ...(defaultValue ? { defaultValue } : {}) },
});

describe('buildDefaultsFromComponents', () => {
  it('skips mode-hidden groups and does not invent a leaf default for the container', () => {
    const hidden: ComponentGroup = {
      uiType: 'hidden',
      components: {
        endpoint: text('spec.monitoring.endpoint', 'http://pmm'),
      },
    };

    const result = buildDefaultsFromComponents({
      monitoring: hidden,
      name: text('spec.name', 'db-1'),
    });

    expect(result).toEqual({ 'spec.name': 'db-1' });
    expect(result).not.toHaveProperty('spec.monitoring.endpoint');
    expect(result).not.toHaveProperty('g-monitoring');
  });
});
