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
  deepClone,
  deleteByPath,
  flattenObject,
  formatDisplayValue,
  getByPath,
  isPlainObject,
  resolvePath,
  setByPath,
} from './object-path';

describe('object-path utils', () => {
  it('resolves path from string and from path array', () => {
    expect(resolvePath('spec.components.proxy.replicas')).toBe(
      'spec.components.proxy.replicas'
    );
    expect(resolvePath(['', 'spec.components.psmdb.replicas'])).toBe(
      'spec.components.psmdb.replicas'
    );
    expect(resolvePath([])).toBeUndefined();
  });

  it('gets, sets and deletes nested values by path', () => {
    const data: Record<string, unknown> = {};

    setByPath(data, 'spec.components.proxy.replicas', 3);
    expect(getByPath(data, 'spec.components.proxy.replicas')).toBe(3);

    deleteByPath(data, 'spec.components.proxy.replicas');
    expect(getByPath(data, 'spec.components.proxy.replicas')).toBeUndefined();
  });

  it('deep clones objects', () => {
    const source = {
      spec: {
        topology: {
          type: 'ha',
        },
      },
    };

    const cloned = deepClone(source);
    (cloned.spec.topology as { type: string }).type = 'standalone';

    expect(source.spec.topology.type).toBe('ha');
    expect(cloned.spec.topology.type).toBe('standalone');
  });

  it('flattens nested objects into dotted entries', () => {
    expect(
      flattenObject({
        spec: {
          components: {
            psmdb: {
              replicas: 3,
            },
          },
        },
      })
    ).toEqual([
      {
        key: 'spec.components.psmdb.replicas',
        value: 3,
      },
    ]);
  });

  it('formats display values consistently', () => {
    expect(formatDisplayValue(undefined)).toBe('\u2014');
    expect(formatDisplayValue(false)).toBe('No');
    expect(formatDisplayValue(true)).toBe('Yes');
    expect(formatDisplayValue({ a: 1 })).toBe('{"a":1}');
    expect(formatDisplayValue(42)).toBe('42');
  });

  it('checks plain object shape', () => {
    expect(isPlainObject({})).toBe(true);
    expect(isPlainObject([])).toBe(false);
    expect(isPlainObject(null)).toBe(false);
    expect(isPlainObject('str')).toBe(false);
  });
});
