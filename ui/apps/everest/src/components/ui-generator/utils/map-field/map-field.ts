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

// A map field keeps rows ({ key, value }) in the form and writes the map to the
// API. Every stage that crosses that boundary goes through here.

import {
  type Component,
  type ComponentGroup,
  type MapComponent,
  isMapComponent,
} from '../../ui-generator.types';
import { maskPasswordValue } from '../display-value';
import {
  deleteByPath,
  getByPath,
  isPlainObject,
  setByPath,
} from '../object-path';
import {
  getComponentSourcePath,
  getComponentTargetPaths,
} from '../preprocess/normalized-component';
import { walkLeafComponents } from '../schema-walker';
import { getMapRow } from './key-value-widget';
import type { MapRow } from './map-field.types';

export const isMapRow = (value: unknown): value is MapRow =>
  isPlainObject(value) &&
  typeof value.key === 'string' &&
  typeof value.value === 'string';

// The value a map field edits; anything else is left as it is.
export const isStringMap = (value: unknown): value is Record<string, string> =>
  isPlainObject(value) &&
  Object.values(value).every((entry) => typeof entry === 'string');

// A row added but never filled is dropped, not reported.
export const isEmptyRow = ({ key, value }: MapRow): boolean =>
  key === '' && value === '';

export const objectToRows = (value: unknown): MapRow[] =>
  isPlainObject(value)
    ? Object.entries(value).map(([key, entryValue]) => ({
        key,
        value: String(entryValue),
      }))
    : [];

export const rowsToObject = (
  rows: unknown
): Record<string, string> | undefined => {
  const entries = (Array.isArray(rows) ? rows : [])
    .filter(isMapRow)
    .filter(({ key }) => key !== '')
    .map(({ key, value }) => [key, value]);
  return entries.length > 0 ? Object.fromEntries(entries) : undefined;
};

// A saved map may hold what its field can't write, e.g. a number set with
// kubectl; it is still shown as stored.
const toDisplayText = (value: unknown): string =>
  typeof value === 'string' ? value : JSON.stringify(value);

// Read-only views (preview, overview) show a map one entry per line: a value
// may itself hold commas or `=`, like AWS tags (`Env=dev,Team=db`). A row
// field typed as a password is masked, as in its input. Reads both the form's
// rows and the saved map; empty when there is no entry.
export const formatMapFieldEntries = (
  value: unknown,
  component: MapComponent
): string[] => {
  const map = Array.isArray(value) ? rowsToObject(value) : value;
  if (!isPlainObject(map)) return [];
  const { key: keyField, value: valueField } = getMapRow(component).components;
  const show = (field: Component | ComponentGroup | undefined, text: string) =>
    field ? maskPasswordValue(field, text) : text;
  return Object.entries(map).map(
    ([key, entryValue]) =>
      `${show(keyField, key)} = ${show(valueField, toDisplayText(entryValue))}`
  );
};

export const getMapFieldPaths = (
  components: Record<string, Component | ComponentGroup>
): string[] => {
  const paths: string[] = [];
  walkLeafComponents(components, ({ component }) => {
    const path = isMapComponent(component)
      ? getComponentSourcePath(component)
      : undefined;
    if (path) paths.push(path);
  });
  return paths;
};

// Every API path a map field writes, the one the form reads included.
// Preprocess drops a map with several paths, so after it there is one each.
export const getMapFieldTargetPaths = (
  components: Record<string, Component | ComponentGroup>
): string[] => {
  const paths: string[] = [];
  walkLeafComponents(components, ({ component }) => {
    if (isMapComponent(component)) {
      paths.push(...getComponentTargetPaths(component));
    }
  });
  return paths;
};

// Mutates `values`: a map with no filled row leaves its path unset. A value
// that isn't rows, like the saved map of a field the form mode hides, is
// already in the API shape and is left as is.
export const writeMapFieldsInApiShape = (
  values: Record<string, unknown>,
  paths: string[]
): void =>
  paths.forEach((path) => {
    const rows = getByPath(values, path);
    if (rows !== undefined && !Array.isArray(rows)) return;
    const value = rowsToObject(rows);
    if (value) {
      setByPath(values, path, value);
    } else {
      deleteByPath(values, path);
    }
  });
