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

export const getSearchTokens = (query: string): string[] =>
  query.trim().toLowerCase().split(/\s+/).filter(Boolean);

const escapeRegExp = (value: string): string =>
  value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');

export const buildHighlightRegExp = (tokens: string[]): RegExp =>
  new RegExp(`(${tokens.map(escapeRegExp).join('|')})`, 'ig');

// Flattens an arbitrary object into a single searchable string of keys and
// primitive values — lets a card be searched by its whole source object, not
// just the fields it displays. Consumers pass the result as `item.searchText`.
export const flattenToSearchText = (value: unknown): string => {
  if (value === null || value === undefined) {
    return '';
  }
  if (
    typeof value === 'string' ||
    typeof value === 'number' ||
    typeof value === 'boolean'
  ) {
    return String(value);
  }
  if (Array.isArray(value)) {
    return value.map(flattenToSearchText).join(' ');
  }
  if (typeof value === 'object') {
    return Object.entries(value)
      .map(([key, nested]) => `${key} ${flattenToSearchText(nested)}`)
      .join(' ');
  }
  return '';
};

const getHaystack = (item: CardPickerItem): string =>
  (
    item.searchText ??
    `${item.title} ${item.subtitle ?? ''} ${item.detail ?? ''}`
  ).toLowerCase();

export const itemMatches = (
  item: CardPickerItem,
  tokens: string[]
): boolean => {
  const haystack = getHaystack(item);
  return tokens.every((token) => haystack.includes(token));
};

// Keeps the selected item visible even if it falls past the inline slice.
export const withSelectedPinned = (
  items: CardPickerItem[],
  count: number,
  selectedId: string
): CardPickerItem[] => {
  const shown = items.slice(0, count);
  if (shown.some((it) => it.id === selectedId)) {
    return shown;
  }
  const selected = items.find((it) => it.id === selectedId);
  if (!selected) {
    return shown;
  }
  return [...shown.slice(0, Math.max(0, count - 1)), selected];
};
