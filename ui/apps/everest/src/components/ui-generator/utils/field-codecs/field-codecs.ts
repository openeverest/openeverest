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

import type { Component, ComponentGroup } from '../../ui-generator.types';
import { getMapFieldCodec } from '../map-field';
import { deleteByPath, getByPath, setByPath } from '../object-path';
import { getComponentTargetPaths } from '../preprocess/normalized-component';
import { getQuantityFieldCodec } from '../quantity';
import { walkLeafComponents } from '../schema-walker';
import type { FieldCodec, FieldCodecResolver } from './field-codecs.types';

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
