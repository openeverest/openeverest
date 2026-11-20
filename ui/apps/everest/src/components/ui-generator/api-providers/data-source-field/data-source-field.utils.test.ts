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
  getReconciledDataSourceValue,
  getStoredValueHint,
} from './data-source-field.utils';
import { Messages } from './data-source-field.messages';

describe('getReconciledDataSourceValue', () => {
  const options = [
    { label: 'first', value: 'first' },
    { label: 'second', value: 'second' },
  ];

  it('returns first option when value is empty and options are available', () => {
    expect(getReconciledDataSourceValue('', options)).toBe('first');
  });

  it('returns first option when value is undefined and options are available', () => {
    expect(getReconciledDataSourceValue(undefined, options)).toBe('first');
  });

  it('returns first option when value is null and options are available', () => {
    expect(getReconciledDataSourceValue(null, options)).toBe('first');
  });

  it('returns null when current value is valid', () => {
    expect(getReconciledDataSourceValue('second', options)).toBeNull();
  });

  it('returns first option when current value is invalid', () => {
    expect(getReconciledDataSourceValue('stale', options)).toBe('first');
  });

  it('returns empty string when value is invalid and options are empty', () => {
    expect(getReconciledDataSourceValue('stale', [])).toBe('');
  });

  it('returns null when value is empty and options are empty', () => {
    expect(getReconciledDataSourceValue('', [])).toBeNull();
  });

  it('returns first option when current value is non-string and options are available', () => {
    expect(getReconciledDataSourceValue(123, options)).toBe('first');
  });
});

describe('getStoredValueHint', () => {
  const options = [{ value: 'standard' }];
  const base = {
    storedValue: 'deleted-class',
    options,
    isLocked: false,
    unsetValueHint: 'Not set',
  };

  it.each([
    ['an empty value', { currentValue: '' }, 'Not set'],
    [
      'an empty value without an unset hint',
      { currentValue: '', unsetValueHint: undefined },
      undefined,
    ],
    ['a value among the options', { currentValue: 'standard' }, undefined],
    [
      'a newly picked value not among the options yet',
      { currentValue: 'just-created' },
      undefined,
    ],
    [
      'a stored value missing from the options',
      { currentValue: 'deleted-class' },
      Messages.unavailableValueHint,
    ],
    [
      'a stored value missing from the options in a locked field',
      { currentValue: 'deleted-class', isLocked: true },
      Messages.unavailableLockedValueHint,
    ],
  ])('for %s', (_case, input, expected) => {
    expect(getStoredValueHint({ ...base, ...input })).toBe(expected);
  });
});
