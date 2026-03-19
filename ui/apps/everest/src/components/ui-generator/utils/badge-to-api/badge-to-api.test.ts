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
import { extractBadgeMappings } from './badge-to-api';
import { FieldType, TopologyUISchemas } from '../../ui-generator.types';

describe('extractBadgeMappings', () => {
  it('expands multipath fields into independent badge mappings', () => {
    const schema: TopologyUISchemas = {
      single: {
        sections: {
          base: {
            components: {
              version: {
                uiType: FieldType.Text,
                path: ['spec.engine.version', 'spec.proxy.version'],
                fieldParams: {
                  label: 'Version',
                  badge: 'GB',
                  badgeToApi: true,
                },
              },
            },
          },
        },
      },
    };

    const mappings = extractBadgeMappings(schema, 'single');

    expect(mappings).toEqual([
      { path: 'spec.engine.version', badge: 'GB' },
      { path: 'spec.proxy.version', badge: 'GB' },
    ]);
  });
});
