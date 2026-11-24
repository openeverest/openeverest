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
  formatQuantity,
  quantitiesEqual,
  readQuantity,
  writeQuantity,
} from './quantity';

describe('readQuantity', () => {
  describe('without unit (CPU / base)', () => {
    it.each([
      ['500m', 0.5],
      ['1.5', 1.5],
      ['0.5', 0.5],
      ['1', 1],
      ['1000m', 1],
      [1, 1],
      [0.5, 0.5],
    ] as const)('reads %j as %j cores', (raw, expected) => {
      expect(readQuantity(raw)).toBe(expected);
    });

    it('reads memory-suffixed values as bytes when no unit is set', () => {
      expect(readQuantity('10Gi')).toBe(10 * 1024 ** 3);
      expect(readQuantity('10240Mi')).toBe(10240 * 1024 ** 2);
    });

    it('treats bare m as millicores, not milli-bytes', () => {
      expect(readQuantity('500m')).toBe(0.5);
    });
  });

  describe('with unit Gi', () => {
    it.each([
      ['10Gi', 10],
      ['10240Mi', 10],
      ['1536Mi', 1.5],
      ['644245094400m', 0.6],
      ['1G', 10 ** 9 / 1024 ** 3],
      ['1234567', 1234567 / 1024 ** 3],
    ] as const)('reads %j as %j Gi', (raw, expected) => {
      expect(readQuantity(raw, 'Gi')).toBe(expected);
    });

    it('reads a JSON integer as bytes into the target unit', () => {
      expect(readQuantity(10 * 1024 ** 3, 'Gi')).toBe(10);
    });
  });

  describe('with unit m (bare numbers are base units)', () => {
    it.each([
      ['1', 1000],
      ['1.5', 1500],
      ['1000m', 1000],
      [1, 1000],
    ] as const)('reads %j as %j millicores', (raw, expected) => {
      expect(readQuantity(raw, 'm')).toBe(expected);
    });
  });

  it('returns undefined for empty values', () => {
    expect(readQuantity(undefined)).toBeUndefined();
    expect(readQuantity(null)).toBeUndefined();
    expect(readQuantity('')).toBeUndefined();
    expect(readQuantity('   ')).toBeUndefined();
  });
});

describe('writeQuantity', () => {
  it('writes fractional CPU as a string, not a JSON float', () => {
    const written = writeQuantity(0.5);
    expect(written).toBe('0.5');
    expect(typeof written).toBe('string');
  });

  it('writes whole numbers without a trailing decimal', () => {
    expect(writeQuantity(10)).toBe('10');
    expect(writeQuantity(1, 'Gi')).toBe('1Gi');
  });

  it('appends the unit on write', () => {
    expect(writeQuantity(10, 'Gi')).toBe('10Gi');
    expect(writeQuantity(0.6, 'Gi')).toBe('0.6Gi');
    expect(writeQuantity(512, 'Mi')).toBe('512Mi');
  });
});

describe('formatQuantity', () => {
  it('formats a form number with unitLabel', () => {
    expect(formatQuantity(0.5, undefined, 'cores')).toBe('0.5 cores');
    expect(formatQuantity(10, 'Gi')).toBe('10 Gi');
    expect(formatQuantity(10, 'Gi', 'GiB')).toBe('10 GiB');
  });

  it('formats a stored API value by reading into unit first', () => {
    expect(formatQuantity('500m', undefined, 'cores')).toBe('0.5 cores');
    expect(formatQuantity('10240Mi', 'Gi')).toBe('10 Gi');
    expect(formatQuantity('644245094400m', 'Gi')).toBe('0.6 Gi');
  });

  it('returns undefined for empty or unreadable values', () => {
    expect(formatQuantity(undefined, 'Gi')).toBeUndefined();
    expect(formatQuantity('', 'Gi')).toBeUndefined();
  });
});

describe('quantitiesEqual', () => {
  it('treats an equivalent stored quantity as equal to the form number', () => {
    expect(quantitiesEqual(10, '10Gi', 'Gi')).toBe(true);
    expect(quantitiesEqual(10, '10240Mi', 'Gi')).toBe(true);
    expect(quantitiesEqual(0.6, '644245094400m', 'Gi')).toBe(true);
    expect(quantitiesEqual(0.5, '500m')).toBe(true);
    expect(quantitiesEqual(0.5, 0.5)).toBe(true);
  });

  it('is false when the form number differs from the stored quantity', () => {
    expect(quantitiesEqual(11, '10Gi', 'Gi')).toBe(false);
    expect(quantitiesEqual(1, '500m')).toBe(false);
  });

  it('treats empty form and empty stored as equal', () => {
    expect(quantitiesEqual(undefined, undefined)).toBe(true);
    expect(quantitiesEqual('', null)).toBe(true);
    expect(quantitiesEqual(0, undefined)).toBe(false);
  });
});
