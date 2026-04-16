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

/**
 * Shared object-path utilities for navigating, mutating, and inspecting
 * nested objects by dot-separated paths (e.g. "spec.components.proxy.replicas").
 */

export const isPlainObject = (
  value: unknown
): value is Record<string, unknown> =>
  typeof value === 'object' && value !== null && !Array.isArray(value);

export const deepClone = <T>(value: T): T => {
  if (typeof structuredClone === 'function') {
    return structuredClone(value);
  }
  return JSON.parse(JSON.stringify(value)) as T;
};

export const resolvePath = (
  path: string | string[] | undefined
): string | undefined => {
  if (typeof path === 'string') {
    return path || undefined;
  }

  if (Array.isArray(path)) {
    return path.find((p): p is string => typeof p === 'string' && !!p);
  }

  return undefined;
};

export const getByPath = (
  obj: Record<string, unknown>,
  path: string
): unknown => {
  if (typeof path !== 'string' || path.length === 0) {
    return undefined;
  }

  return path.split('.').reduce<unknown>((current, key) => {
    if (!isPlainObject(current)) {
      return undefined;
    }

    return current[key];
  }, obj);
};

export const setByPath = (
  obj: Record<string, unknown>,
  path: string,
  value: unknown
): void => {
  if (typeof path !== 'string' || path.length === 0) {
    return;
  }

  const parts = path.split('.');
  let current: Record<string, unknown> = obj;

  for (let i = 0; i < parts.length - 1; i++) {
    const key = parts[i];
    if (!isPlainObject(current[key])) {
      current[key] = {};
    }
    current = current[key] as Record<string, unknown>;
  }

  current[parts[parts.length - 1]] = value;
};

export const deleteByPath = (
  obj: Record<string, unknown>,
  path: string
): void => {
  if (typeof path !== 'string' || path.length === 0) {
    return;
  }

  const parts = path.split('.');
  let current: Record<string, unknown> = obj;

  for (let i = 0; i < parts.length - 1; i++) {
    const key = parts[i];
    const next = current[key];
    if (!isPlainObject(next)) {
      return;
    }
    current = next;
  }

  delete current[parts[parts.length - 1]];
};

export type FlatEntry = { key: string; value: unknown };

export const flattenObject = (obj: unknown, prefix = ''): FlatEntry[] => {
  const result: FlatEntry[] = [];
  if (!isPlainObject(obj)) return result;

  for (const [k, v] of Object.entries(obj)) {
    const fullKey = prefix ? `${prefix}.${k}` : k;
    if (isPlainObject(v)) {
      result.push(...flattenObject(v, fullKey));
    } else {
      result.push({ key: fullKey, value: v });
    }
  }

  return result;
};

export const deepMerge = (
  base: Record<string, unknown>,
  updates: Record<string, unknown>
): Record<string, unknown> => {
  const result: Record<string, unknown> = { ...base };
  for (const key of Object.keys(updates)) {
    const baseVal = result[key];
    const updateVal = updates[key];
    if (isPlainObject(baseVal) && isPlainObject(updateVal)) {
      result[key] = deepMerge(baseVal, updateVal);
    } else {
      result[key] = updateVal;
    }
  }
  return result;
};

export const formatDisplayValue = (value: unknown): string => {
  if (value === undefined || value === null) return '\u2014';
  if (typeof value === 'boolean') return value ? 'Yes' : 'No';
  if (typeof value === 'object') return JSON.stringify(value);
  return String(value);
};
