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

import { CardPickerItem } from './card-picker.types';
import {
  flattenToSearchText,
  getSearchTokens,
  itemMatches,
  withSelectedPinned,
} from './card-picker.utils';

describe('card-picker utils', () => {
  describe('getSearchTokens', () => {
    it('lowercases, trims and drops empty tokens', () => {
      expect(getSearchTokens('  Foo   Bar ')).toEqual(['foo', 'bar']);
      expect(getSearchTokens('')).toEqual([]);
      expect(getSearchTokens('   ')).toEqual([]);
    });
  });

  describe('flattenToSearchText', () => {
    it('flattens keys and primitive values from nested objects and arrays', () => {
      const text = flattenToSearchText({
        a: { b: 'X' },
        c: [1, true],
        d: null,
      });

      ['a', 'b', 'X', 'c', '1', 'true'].forEach((token) =>
        expect(text).toContain(token)
      );
    });
  });

  describe('itemMatches', () => {
    const item: CardPickerItem = {
      id: '1',
      title: 'Prod',
      subtitle: 'replicaSet',
      detail: '3 nodes',
      searchText: 'prod replicaset storageclass local-path',
    };

    it('matches against searchText (fields not shown on the card)', () => {
      expect(itemMatches(item, ['storageclass'])).toBe(true);
      expect(itemMatches(item, ['local-path'])).toBe(true);
    });

    it('requires every token (AND)', () => {
      expect(itemMatches(item, ['prod', 'storageclass'])).toBe(true);
      expect(itemMatches(item, ['prod', 'missing'])).toBe(false);
    });

    it('falls back to title/subtitle/detail when no searchText, case-insensitively', () => {
      const noSearch: CardPickerItem = {
        id: '2',
        title: 'Analytics',
        subtitle: 'Sharded',
        detail: '7 NODES',
      };

      expect(itemMatches(noSearch, ['sharded'])).toBe(true);
      expect(itemMatches(noSearch, ['nodes'])).toBe(true);
      expect(itemMatches(noSearch, ['storageclass'])).toBe(false);
    });
  });

  describe('withSelectedPinned', () => {
    const items: CardPickerItem[] = ['a', 'b', 'c', 'd', 'e'].map((id) => ({
      id,
      title: id,
    }));
    const ids = (result: CardPickerItem[]) => result.map((it) => it.id);

    it('returns the first N when the selection is already within them', () => {
      expect(ids(withSelectedPinned(items, 2, 'a'))).toEqual(['a', 'b']);
    });

    it('pins the selected item into the last slot when it falls past N', () => {
      expect(ids(withSelectedPinned(items, 2, 'd'))).toEqual(['a', 'd']);
    });

    it('returns the first N when there is no selection', () => {
      expect(ids(withSelectedPinned(items, 2, ''))).toEqual(['a', 'b']);
    });

    it('handles a zero budget by showing just the pinned selection', () => {
      expect(ids(withSelectedPinned(items, 0, 'c'))).toEqual(['c']);
    });
  });
});
