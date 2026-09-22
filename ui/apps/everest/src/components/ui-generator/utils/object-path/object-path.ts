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
 *
 * Keys that themselves contain dots (e.g. Kubernetes qualified names like
 * "nvidia.com/gpu") must use bracket-quoted segments:
 * "resources.limits['nvidia.com/gpu']".
 */

const SIMPLE_SEGMENT = /^[A-Za-z0-9_]+$/;

/**
 * Splits a path into segments. Supports dot separators and bracket-quoted
 * segments (`['a.b']` / `["a.b"]` / `[0]`). Backslash escapes the quote and
 * backslash itself inside brackets.
 */
export const parsePath = (path: string): string[] => {
  if (typeof path !== 'string' || path.length === 0) {
    return [];
  }

  const parts: string[] = [];
  let current = '';
  let i = 0;

  const pushCurrent = (): void => {
    if (current.length > 0) {
      parts.push(current);
      current = '';
    }
  };

  while (i < path.length) {
    const char = path[i];

    if (char === '.') {
      pushCurrent();
      i += 1;
      continue;
    }

    if (char === '[') {
      pushCurrent();
      i += 1;
      // Skip whitespace inside brackets
      while (i < path.length && path[i] === ' ') i += 1;
      if (i >= path.length) break;

      const quote = path[i];
      if (quote === "'" || quote === '"') {
        i += 1;
        let segment = '';
        let closed = false;
        while (i < path.length) {
          const c = path[i];
          if (c === '\\' && i + 1 < path.length) {
            const next = path[i + 1];
            if (next === quote || next === '\\') {
              segment += next;
              i += 2;
              continue;
            }
          }
          if (c === quote) {
            closed = true;
            i += 1;
            break;
          }
          segment += c;
          i += 1;
        }
        if (closed) {
          parts.push(segment);
          // Skip to closing bracket
          while (i < path.length && path[i] === ' ') i += 1;
          if (path[i] === ']') i += 1;
          // Skip an optional dot after the bracket
          if (path[i] === '.') i += 1;
          continue;
        }
        // Unclosed quote: treat the rest literally
        current = `[${quote}${segment}`;
        continue;
      }

      // Unquoted bracket content (e.g. [0]): read until ']'
      let segment = '';
      while (i < path.length && path[i] !== ']') {
        segment += path[i];
        i += 1;
      }
      if (path[i] === ']') i += 1;
      const trimmed = segment.trim();
      if (trimmed.length > 0) {
        parts.push(trimmed);
      }
      if (path[i] === '.') i += 1;
      continue;
    }

    current += char;
    i += 1;
  }

  pushCurrent();
  return parts;
};

/**
 * Joins segments into a canonical path. Segments with dots, slashes,
 * spaces, or other special chars become bracket-quoted (`['a.b']`).
 */
export const joinPath = (parts: string[]): string => {
  let out = '';
  for (const part of parts) {
    if (part.length === 0) continue;
    if (SIMPLE_SEGMENT.test(part)) {
      out += out.length > 0 ? `.${part}` : part;
    } else {
      const escaped = part.replace(/\\/g, '\\\\').replace(/'/g, "\\'");
      // Bracket segments never take a leading dot:
      // "resources.limits['nvidia.com/gpu']"
      out += `['${escaped}']`;
    }
  }
  return out;
};

/**
 * Returns canonical string prefixes for a path, including the full path.
 */
export const getPathPrefixes = (path: string): string[] => {
  const parts = parsePath(path);
  const prefixes: string[] = [];
  for (let i = 1; i <= parts.length; i++) {
    prefixes.push(joinPath(parts.slice(0, i)));
  }
  return prefixes;
};

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

  return parsePath(path).reduce<unknown>((current, key) => {
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

  const parts = parsePath(path);
  if (parts.length === 0) return;
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

  const parts = parsePath(path);
  if (parts.length === 0) return;
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

  const prefixParts = prefix ? parsePath(prefix) : [];
  for (const [k, v] of Object.entries(obj)) {
    const fullKey = joinPath([...prefixParts, k]);
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
