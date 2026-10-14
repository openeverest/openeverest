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
import { formatQuantity, readQuantity, writeQuantity } from './quantity';

describe('readQuantity', () => {
  it.each([
    ['500m', '', 0.5],
    [2, '', 2],
    ['300m', 'm', 300],
    ['10Gi', 'Gi', 10],
    ['512Mi', 'Gi', 0.5],
    // Kubernetes stores 0.6Gi as milli-bytes (#2423).
    ['644245094400m', 'Gi', 0.6],
  ] as const)('reads %s in "%s" as %s', (stored, unit, expected) => {
    expect(readQuantity(stored, unit)).toBe(expected);
  });

  it('keeps a value that is not a quantity', () => {
    expect(readQuantity('16kg', 'Gi')).toBe('16kg');
  });
});

describe('writeQuantity', () => {
  it.each([
    [0.5, '', '0.5'],
    [10, 'Gi', '10Gi'],
    [0.1 + 0.2, '', '0.3'],
  ] as const)('writes %s in "%s" as %s', (value, unit, expected) => {
    expect(writeQuantity(value, unit)).toBe(expected);
  });

  it('keeps a value that is not a number', () => {
    expect(writeQuantity('500m', '')).toBe('500m');
  });
});

describe('formatQuantity', () => {
  it('shows the unit label, else the unit, else the bare amount', () => {
    expect(formatQuantity(0.5, { unitLabel: 'cores' })).toBe('0.5 cores');
    expect(formatQuantity(10, { unit: 'Gi' })).toBe('10 Gi');
    expect(formatQuantity(2, {})).toBe('2');
  });
});
