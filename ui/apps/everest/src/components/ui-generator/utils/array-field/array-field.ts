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

// An array field keeps rows in the form and writes its API shape (a map for
// outputType object). Every stage that crosses that boundary goes through here.

import type { KeyValuePair } from '@percona/ui-lib';
import {
  type Component,
  type ComponentGroup,
  isArrayComponent,
} from '../../ui-generator.types';
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

export const isKeyValuePair = (value: unknown): value is KeyValuePair =>
  isPlainObject(value) &&
  typeof value.key === 'string' &&
  typeof value.value === 'string';

// A row added but never filled is dropped, not reported.
export const isEmptyRow = ({ key, value }: KeyValuePair): boolean =>
  key === '' && value === '';

export const objectToRows = (value: unknown): KeyValuePair[] =>
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
    .filter(isKeyValuePair)
    .filter(({ key }) => key !== '')
    .map(({ key, value }) => [key, value]);
  return entries.length > 0 ? Object.fromEntries(entries) : undefined;
};

// Reads both the form's rows and the saved map; undefined when there is none.
export const formatArrayFieldValue = (value: unknown): string | undefined => {
  const map = Array.isArray(value) ? rowsToObject(value) : value;
  if (!isPlainObject(map) || Object.keys(map).length === 0) return undefined;
  return Object.entries(map)
    .map(([key, entryValue]) => `${key} = ${String(entryValue)}`)
    .join(', ');
};

export const getArrayFieldPaths = (
  components: Record<string, Component | ComponentGroup>
): string[] => {
  const paths: string[] = [];
  walkLeafComponents(components, ({ component }) => {
    const path = isArrayComponent(component)
      ? getComponentSourcePath(component)
      : undefined;
    if (path) paths.push(path);
  });
  return paths;
};

// Every API path an array field writes its map to: its own and, for a
// multi-path field, the paths the map is copied to.
export const getArrayFieldTargetPaths = (
  components: Record<string, Component | ComponentGroup>
): string[] => {
  const paths: string[] = [];
  walkLeafComponents(components, ({ component }) => {
    if (isArrayComponent(component)) {
      paths.push(...getComponentTargetPaths(component));
    }
  });
  return paths;
};

// Mutates `values`: an array with no filled row leaves its path unset. A value
// that isn't rows, like the saved map of a field the form mode hides, is
// already in the API shape and is left as is.
export const writeArrayFieldsInApiShape = (
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
