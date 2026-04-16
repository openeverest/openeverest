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
import {
  extractBadgeMappings,
  applyBadgesToFormData,
  stripBadgeFromValue,
} from './badge-to-api';
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

  it('ignores components without badge or badgeToApi', () => {
    const schema: TopologyUISchemas = {
      ha: {
        sections: {
          resources: {
            components: {
              cpu: {
                uiType: FieldType.Number,
                path: 'spec.cpu',
                fieldParams: { label: 'CPU' },
              },
              memory: {
                uiType: FieldType.Number,
                path: 'spec.memory',
                fieldParams: { label: 'Memory', badge: 'Gi' },
              },
            },
          },
        },
      },
    };

    const mappings = extractBadgeMappings(schema, 'ha');
    expect(mappings).toEqual([]);
  });
});

describe('applyBadgesToFormData', () => {
  it('appends badge suffix to a nested field value', () => {
    const data = { spec: { memory: '4' } };
    const result = applyBadgesToFormData(data, [
      { path: 'spec.memory', badge: 'Gi' },
    ]);
    expect(result).toEqual({ spec: { memory: '4Gi' } });
  });

  it('applies badges to multiple fields', () => {
    const data = { spec: { memory: '4', disk: '25' } };
    const result = applyBadgesToFormData(data, [
      { path: 'spec.memory', badge: 'Gi' },
      { path: 'spec.disk', badge: 'Gi' },
    ]);
    expect(result).toEqual({ spec: { memory: '4Gi', disk: '25Gi' } });
  });

  it('skips undefined, null, and empty string values', () => {
    const data = { spec: { memory: undefined, disk: null, cpu: '' } };
    const result = applyBadgesToFormData(data, [
      { path: 'spec.memory', badge: 'Gi' },
      { path: 'spec.disk', badge: 'Gi' },
      { path: 'spec.cpu', badge: 'cores' },
    ]);
    expect(result).toEqual({
      spec: { memory: undefined, disk: null, cpu: '' },
    });
  });

  it('does not mutate the original data', () => {
    const data = { spec: { memory: '4' } };
    applyBadgesToFormData(data, [{ path: 'spec.memory', badge: 'Gi' }]);
    expect(data).toEqual({ spec: { memory: '4' } });
  });

  it('returns data unchanged when no mappings are provided', () => {
    const data = { spec: { memory: '4' } };
    const result = applyBadgesToFormData(data, []);
    expect(result).toEqual({ spec: { memory: '4' } });
  });
});

describe('stripBadgeFromValue', () => {
  it('removes a trailing badge from string values', () => {
    expect(stripBadgeFromValue('16Gi', 'Gi')).toBe('16');
  });

  it('trims optional whitespace before the badge', () => {
    expect(stripBadgeFromValue('16 Gi', 'Gi')).toBe('16');
  });

  it('returns the original value when badge does not match', () => {
    expect(stripBadgeFromValue('16Mi', 'Gi')).toBe('16Mi');
  });

  it('returns non-string values unchanged', () => {
    expect(stripBadgeFromValue(16, 'Gi')).toBe(16);
  });
});
