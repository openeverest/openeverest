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
import { getReconciledDataSourceValue } from './data-source-field.utils';

describe('getReconciledDataSourceValue', () => {
  const options = [
    { label: 'first', value: 'first' },
    { label: 'second', value: 'second' },
  ];

  it('returns first option when value is empty and options are available', () => {
    expect(getReconciledDataSourceValue('', options, true)).toBe('first');
  });

  it('returns first option when value is undefined and options are available', () => {
    expect(getReconciledDataSourceValue(undefined, options, true)).toBe(
      'first'
    );
  });

  it('returns first option when value is null and options are available', () => {
    expect(getReconciledDataSourceValue(null, options, true)).toBe('first');
  });

  it('returns null when current value is valid', () => {
    expect(getReconciledDataSourceValue('second', options, true)).toBeNull();
  });

  it('returns first option when current value is invalid', () => {
    expect(getReconciledDataSourceValue('stale', options, true)).toBe('first');
  });

  it('returns empty string when value is invalid and options are empty', () => {
    expect(getReconciledDataSourceValue('stale', [], true)).toBe('');
  });

  it('returns null when value is empty and options are empty', () => {
    expect(getReconciledDataSourceValue('', [], true)).toBeNull();
  });

  it('returns first option when current value is non-string and options are available', () => {
    expect(getReconciledDataSourceValue(123, options, true)).toBe('first');
  });

  describe('with autoSelect disabled', () => {
    it('keeps an empty value empty', () => {
      expect(getReconciledDataSourceValue('', options, false)).toBeNull();
      expect(
        getReconciledDataSourceValue(undefined, options, false)
      ).toBeNull();
    });

    it('keeps a value that is missing from the options', () => {
      expect(getReconciledDataSourceValue('stale', options, false)).toBeNull();
      expect(getReconciledDataSourceValue('stale', [], false)).toBeNull();
    });
  });
});
