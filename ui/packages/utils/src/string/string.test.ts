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
import kebabize from './string';

describe('kebabize', () => {
  it.each([
    ['dbName', 'db-name'],
    ['storageClass', 'storage-class'],
    ['spec.testText', 'spec.test-text'],
    ['Cancel', 'cancel'],
    ['myURLParser', 'my-url-parser'],
    ['URLParser', 'url-parser'],
    ['ABc', 'a-bc'],
    ['aAB', 'a-ab'],
  ])('hyphenates words and acronyms: %s → %s', (input, expected) => {
    expect(kebabize(input)).toBe(expected);
  });

  it.each([
    ['Create database', 'create-database'],
    ['Add new destination', 'add-new destination'],
    [' leading', '-leading'],
    ['a  -b', 'a-b'],
    ['a - b', 'a--b'],
    ['a  b  c', 'a-b  c'],
  ])('joins only the first whitespace run: %j → %j', (input, expected) => {
    expect(kebabize(input)).toBe(expected);
  });

  it('stays linear on long whitespace and capital runs', () => {
    const start = performance.now();
    kebabize(`${' '.repeat(100_000)}x`);
    kebabize(`${'A'.repeat(100_000)}a`);
    expect(performance.now() - start).toBeLessThan(1000);
  });
});
