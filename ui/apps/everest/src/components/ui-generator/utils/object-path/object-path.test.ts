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
  deepMerge,
  deleteByPath,
  flattenObject,
  formatDisplayValue,
  getByPath,
  isPlainObject,
  resolvePath,
  setByPath,
} from './object-path';

describe('object-path utils', () => {
  describe('resolvePath', () => {
    it('resolves a non-empty string path', () => {
      expect(resolvePath('spec.components.proxy.replicas')).toBe(
        'spec.components.proxy.replicas'
      );
    });

    it('returns undefined for an empty string', () => {
      expect(resolvePath('')).toBeUndefined();
    });

    it('returns undefined for undefined input', () => {
      expect(resolvePath(undefined)).toBeUndefined();
    });

    it('resolves first valid entry from a path array', () => {
      expect(resolvePath(['', 'spec.components.psmdb.replicas'])).toBe(
        'spec.components.psmdb.replicas'
      );
    });

    it('returns undefined for an empty array', () => {
      expect(resolvePath([])).toBeUndefined();
    });

    it('returns undefined when all array entries are empty', () => {
      expect(resolvePath(['', ''])).toBeUndefined();
    });
  });

  describe('getByPath', () => {
    it('returns value at a nested path', () => {
      const data = { spec: { replicas: 3 } };
      expect(getByPath(data, 'spec.replicas')).toBe(3);
    });

    it('returns undefined for a non-existent path', () => {
      expect(getByPath({ a: 1 }, 'b.c')).toBeUndefined();
    });

    it('returns undefined for an empty string path', () => {
      expect(getByPath({ a: 1 }, '')).toBeUndefined();
    });

    it('returns undefined when a non-object is in the path chain', () => {
      expect(getByPath({ a: 'string' }, 'a.b')).toBeUndefined();
    });

    it('returns a top-level key', () => {
      expect(getByPath({ x: 42 }, 'x')).toBe(42);
    });
  });

  describe('setByPath', () => {
    it('sets a deeply nested value, creating intermediary objects', () => {
      const data: Record<string, unknown> = {};
      setByPath(data, 'spec.components.proxy.replicas', 3);
      expect(getByPath(data, 'spec.components.proxy.replicas')).toBe(3);
    });

    it('overwrites an existing value', () => {
      const data: Record<string, unknown> = { a: { b: 1 } };
      setByPath(data, 'a.b', 2);
      expect(getByPath(data, 'a.b')).toBe(2);
    });

    it('is a no-op for empty path', () => {
      const data: Record<string, unknown> = { a: 1 };
      setByPath(data, '', 99);
      expect(data).toEqual({ a: 1 });
    });

    it('replaces a non-object intermediate with an object', () => {
      const data: Record<string, unknown> = { a: 'string' };
      setByPath(data, 'a.b', 5);
      expect(getByPath(data, 'a.b')).toBe(5);
    });
  });

  describe('deleteByPath', () => {
    it('deletes a deeply nested key', () => {
      const data: Record<string, unknown> = {};
      setByPath(data, 'spec.components.proxy.replicas', 3);
      deleteByPath(data, 'spec.components.proxy.replicas');
      expect(getByPath(data, 'spec.components.proxy.replicas')).toBeUndefined();
    });

    it('is a no-op when the path does not exist', () => {
      const data = { a: 1 };
      deleteByPath(data, 'x.y.z');
      expect(data).toEqual({ a: 1 });
    });

    it('is a no-op for an empty path', () => {
      const data = { a: 1 };
      deleteByPath(data, '');
      expect(data).toEqual({ a: 1 });
    });
  });

  describe('deepClone', () => {
    it('produces an independent copy', () => {
      const source = { spec: { topology: { type: 'ha' } } };
      const cloned = deepClone(source);
      (cloned.spec.topology as { type: string }).type = 'standalone';

      expect(source.spec.topology.type).toBe('ha');
      expect(cloned.spec.topology.type).toBe('standalone');
    });
  });

  describe('flattenObject', () => {
    it('flattens nested objects into dotted entries', () => {
      expect(
        flattenObject({
          spec: { components: { psmdb: { replicas: 3 } } },
        })
      ).toEqual([{ key: 'spec.components.psmdb.replicas', value: 3 }]);
    });

    it('returns empty array for non-object input', () => {
      expect(flattenObject(null)).toEqual([]);
      expect(flattenObject(42)).toEqual([]);
      expect(flattenObject('str')).toEqual([]);
    });

    it('returns empty array for an empty object', () => {
      expect(flattenObject({})).toEqual([]);
    });

    it('handles multiple sibling keys', () => {
      const result = flattenObject({ a: 1, b: 2 });
      expect(result).toEqual([
        { key: 'a', value: 1 },
        { key: 'b', value: 2 },
      ]);
    });

    it('uses prefix when provided', () => {
      expect(flattenObject({ x: 10 }, 'root')).toEqual([
        { key: 'root.x', value: 10 },
      ]);
    });
  });

  describe('formatDisplayValue', () => {
    it('formats undefined as em-dash', () => {
      expect(formatDisplayValue(undefined)).toBe('\u2014');
    });

    it('formats null as em-dash', () => {
      expect(formatDisplayValue(null)).toBe('\u2014');
    });

    it('formats booleans', () => {
      expect(formatDisplayValue(false)).toBe('No');
      expect(formatDisplayValue(true)).toBe('Yes');
    });

    it('formats objects as JSON', () => {
      expect(formatDisplayValue({ a: 1 })).toBe('{"a":1}');
    });

    it('formats numbers as strings', () => {
      expect(formatDisplayValue(42)).toBe('42');
    });

    it('formats strings as-is', () => {
      expect(formatDisplayValue('hello')).toBe('hello');
    });
  });

  describe('isPlainObject', () => {
    it('returns true for plain objects', () => {
      expect(isPlainObject({})).toBe(true);
      expect(isPlainObject({ a: 1 })).toBe(true);
    });

    it('returns false for arrays, null, and primitives', () => {
      expect(isPlainObject([])).toBe(false);
      expect(isPlainObject(null)).toBe(false);
      expect(isPlainObject('str')).toBe(false);
      expect(isPlainObject(123)).toBe(false);
      expect(isPlainObject(undefined)).toBe(false);
    });
  });

  describe('deepMerge', () => {
    it('overwrites leaf values with updates', () => {
      const base = { a: 1, b: 2 };
      const updates = { b: 99 };
      expect(deepMerge(base, updates)).toEqual({ a: 1, b: 99 });
    });

    it('adds new keys from updates', () => {
      const base = { a: 1 };
      const updates = { b: 2 };
      expect(deepMerge(base, updates)).toEqual({ a: 1, b: 2 });
    });

    it('deep-merges nested objects', () => {
      const base = {
        spec: {
          components: { engine: { replicas: 3 }, proxy: { replicas: 2 } },
          topology: { type: 'replicaSet' },
        },
      };
      const updates = {
        spec: {
          components: { engine: { replicas: 5 } },
        },
      };
      expect(deepMerge(base, updates)).toEqual({
        spec: {
          components: { engine: { replicas: 5 }, proxy: { replicas: 2 } },
          topology: { type: 'replicaSet' },
        },
      });
    });

    it('replaces non-object base value with object from updates', () => {
      const base = { a: 'string' };
      const updates = { a: { nested: true } };
      expect(deepMerge(base, updates)).toEqual({ a: { nested: true } });
    });

    it('replaces object base value with non-object from updates', () => {
      const base = { a: { nested: true } };
      const updates = { a: 42 };
      expect(deepMerge(base, updates)).toEqual({ a: 42 });
    });

    it('does not mutate the base object', () => {
      const base = { a: { b: 1 } };
      const updates = { a: { b: 2 } };
      deepMerge(base, updates);
      expect(base.a.b).toBe(1);
    });
  });
});
