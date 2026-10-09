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

import { memoryParser, parseQuantity } from '.';

describe('parseQuantity', () => {
  it.each([
    ['500m', 0.5],
    ['300m', 0.3],
    ['1.5', 1.5],
    ['2k', 2000],
    ['1Ki', 1024],
    ['1Gi', 1024 ** 3],
  ])('reads %s as %s', (input, expected) => {
    expect(parseQuantity(input)).toBe(expected);
  });

  it.each(['', 'abc', '16kg', '1.2.3'])('rejects %j', (input) => {
    expect(parseQuantity(input)).toBeUndefined();
  });
});

describe('memory parser', () => {
  it('correctly parses memory strings', () => {
    expect(memoryParser('1')).toEqual({ value: 1, originalUnit: '' });
    expect(memoryParser('1k', 'G')).toEqual({
      value: 1 * 10 ** -6,
      originalUnit: 'k',
    });
    expect(memoryParser('1G', 'Gi')).toEqual({
      value: 10 ** 9 / 1024 ** 3,
      originalUnit: 'G',
    });
    expect(memoryParser('1G', 'G')).toEqual({ value: 1, originalUnit: 'G' });
  });

  it('parses the milli-byte quantities Kubernetes emits for non-integer conversions', () => {
    // e.g. a 0.6Gi limit round-trips through Kubernetes as "644245094400m"
    // because 0.6Gi is not a whole number of bytes (see #2423).
    expect(memoryParser('644245094400m', 'Gi')).toEqual({
      value: 0.6,
      originalUnit: 'm',
    });
  });
});
