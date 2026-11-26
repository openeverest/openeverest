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

import type {
  Component,
  ComponentGroup,
  Section,
} from '../../ui-generator.types';
import { getMapFieldCodec } from '../map-field';
import {
  deepClone,
  deleteByPath,
  getByPath,
  isPlainObject,
  setByPath,
} from '../object-path';
import { getComponentTargetPaths } from '../preprocess/normalized-component';
import { getQuantityFieldCodec } from '../quantity';
import { walkLeafComponents } from '../schema-walker';
import type {
  CelValueReader,
  FieldCodec,
  FieldCodecResolver,
} from './field-codecs.types';

// The one place that knows which fields convert. A new field type whose form
// value differs from its API value adds its resolver here, and no stage
// changes.
const RESOLVERS: FieldCodecResolver[] = [
  getMapFieldCodec,
  getQuantityFieldCodec,
];

export const getFieldCodec = (component: Component): FieldCodec | undefined => {
  for (const resolve of RESOLVERS) {
    const codec = resolve(component);
    if (codec) return codec;
  }
  return undefined;
};

// Mutates `values`: every field with a codec is written the way the API takes
// it, at each path it writes. `saved` holds the saved values at the same
// paths, when the form edits something saved.
export const writeFieldsInApiShape = (
  values: Record<string, unknown>,
  components: Record<string, Component | ComponentGroup>,
  saved?: Record<string, unknown>
): void =>
  walkLeafComponents(components, ({ component }) => {
    const codec = getFieldCodec(component);
    if (!codec) return;
    getComponentTargetPaths(component).forEach((path) => {
      const apiValue = codec.toApi(
        getByPath(values, path),
        saved ? getByPath(saved, path) : undefined
      );
      if (apiValue === undefined) {
        deleteByPath(values, path);
      } else {
        setByPath(values, path, apiValue);
      }
    });
  });

// The paths of the fields that own their value as a whole.
export const getWholeValuePaths = (
  components: Record<string, Component | ComponentGroup>
): string[] => {
  const paths: string[] = [];
  walkLeafComponents(components, ({ component }) => {
    if (getFieldCodec(component)?.ownsWholeValue) {
      paths.push(...getComponentTargetPaths(component));
    }
  });
  return paths;
};

// One reader per path of a field that a CEL rule reads otherwise than the
// form keeps it.
export const getCelValueReaders = (
  sections: Record<string, Section>
): CelValueReader[] => {
  const readers: CelValueReader[] = [];
  Object.values(sections).forEach((section) => {
    if (!section?.components) return;
    walkLeafComponents(section.components, ({ component }) => {
      const { toCel } = getFieldCodec(component) ?? {};
      if (!toCel) return;
      getComponentTargetPaths(component).forEach((path) =>
        readers.push({ path, read: toCel })
      );
    });
  });
  return readers;
};

const holdsPath = (values: Record<string, unknown>, path: string): boolean => {
  const parentPath = path.split('.').slice(0, -1).join('.');
  return parentPath === '' || isPlainObject(getByPath(values, parentPath));
};

// The values a CEL rule evaluates: a copy of the form's values, or of the
// saved ones, with each reader's path read through it.
export const readValuesForCel = (
  values: Record<string, unknown>,
  readers: CelValueReader[]
): Record<string, unknown> => {
  if (readers.length === 0) return values;
  const result = deepClone(values);
  readers.forEach(({ path, read }) => {
    // Reading adds no object around a value. A rule that asks whether one is
    // there, like `has(spec.proxy)`, gets the same answer whether or not the
    // schema has such a field inside it.
    if (holdsPath(result, path)) {
      setByPath(result, path, read(getByPath(result, path)));
    }
  });
  return result;
};

// A rule names what it reads by API path, e.g. `spec.labels.team`, while the
// form has one field for the whole value at `spec.labels`. The rule has to
// run again when that field changes, so the dependency becomes the field.
export const watchFieldsReadByCel = (
  celDependencyGroups: string[][],
  readers: CelValueReader[]
): string[][] => {
  if (readers.length === 0) return celDependencyGroups;
  const toField = (dependency: string): string =>
    readers.find(({ path }) => dependency.startsWith(`${path}.`))?.path ??
    dependency;
  return celDependencyGroups.map((group) => {
    const fields = group.map(toField);
    return fields.some((field, index) => field !== group[index])
      ? Array.from(new Set(fields))
      : group;
  });
};
