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

import { isMapComponent } from '../../ui-generator.types';
import type {
  FieldCodec,
  FieldCodecResolver,
} from '../field-codecs/field-codecs.types';
import { isPlainObject } from '../object-path/object-path';
import { isStringMap, objectToRows, toMap } from './map-field';

// `original` may be the instance as saved, and that can hold a literal `{}`.
// A rule finds it unset, like rows with no entry.
const toMapWithEntries = (formValue: unknown): unknown => {
  const map = toMap(formValue);
  return isPlainObject(map) && Object.keys(map).length === 0 ? undefined : map;
};

// The form keeps a map as rows ({ key, value }); the API takes the map.
const MAP_FIELD_CODEC: FieldCodec = {
  toForm: objectToRows,
  toApi: toMap,
  // A default is written in the API shape. One that isn't a map of strings is
  // ignored rather than written back as text.
  defaultToForm: (defaultValue) =>
    isStringMap(defaultValue) ? objectToRows(defaultValue) : undefined,
  // A rule reads what the field writes, so it can be written against the
  // API: the map, and nothing at all while it has no entries.
  toCel: toMapWithEntries,
  ownsWholeValue: true,
};

export const getMapFieldCodec: FieldCodecResolver = (component) =>
  isMapComponent(component) ? MAP_FIELD_CODEC : undefined;
