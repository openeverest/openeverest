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
import { WidgetType } from 'components/ui-generator/ui-generator.types';
import { buildAffinitySections } from './build-scheduling-sections';

describe('buildAffinitySections', () => {
  it('creates one affinity section per target, bound to its path', () => {
    const sections = buildAffinitySections([
      {
        key: 'engine',
        path: 'spec.components.engine.schedulingPolicy.affinity',
      },
      { key: 'proxy', path: 'spec.components.proxy.schedulingPolicy.affinity' },
    ]);

    expect(Object.keys(sections)).toEqual(['engine', 'proxy']);
    expect(sections.engine.components.affinity).toEqual({
      uiType: WidgetType.Affinity,
      path: 'spec.components.engine.schedulingPolicy.affinity',
      fieldParams: {},
    });
  });
});
