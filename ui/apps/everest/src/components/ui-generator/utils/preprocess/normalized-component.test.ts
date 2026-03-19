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
import { FieldType } from '../../ui-generator.types';
import {
  getComponentSourcePath,
  getComponentTargetPaths,
  withNormalizedPathMeta,
} from './normalized-component';

describe('normalized-component', () => {
  it('uses first path as source and keeps all targets for multipath fields', () => {
    const component = withNormalizedPathMeta({
      uiType: FieldType.Text,
      path: ['spec.engine.version', 'spec.proxy.version'],
      fieldParams: { label: 'Version' },
    });

    expect(getComponentSourcePath(component)).toBe('spec.engine.version');
    expect(getComponentTargetPaths(component)).toEqual([
      'spec.engine.version',
      'spec.proxy.version',
    ]);
  });

  it('supports single path fields', () => {
    const component = withNormalizedPathMeta({
      uiType: FieldType.Number,
      path: 'spec.replicas',
      fieldParams: { label: 'Replicas' },
    });

    expect(getComponentSourcePath(component)).toBe('spec.replicas');
    expect(getComponentTargetPaths(component)).toEqual(['spec.replicas']);
  });

  it('returns no source path for id-only fields', () => {
    const component = withNormalizedPathMeta({
      uiType: FieldType.Hidden,
      id: 'ui.hidden-field',
      fieldParams: {},
    });

    expect(getComponentSourcePath(component)).toBeUndefined();
    expect(getComponentTargetPaths(component)).toEqual([]);
  });
});
