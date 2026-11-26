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
import { isStringMap, objectToRows, rowsToObject } from './map-field';

// A map with no filled row has no map to write. A value that isn't rows, like
// a saved map a form loaded unconverted, is in the API shape already.
const toMap = (formValue: unknown): unknown =>
  formValue !== undefined && !Array.isArray(formValue)
    ? formValue
    : rowsToObject(formValue);

// The form keeps a map as rows ({ key, value }); the API takes the map.
const MAP_FIELD_CODEC: FieldCodec = {
  toForm: objectToRows,
  toApi: toMap,
  // A default is written in the API shape. One that isn't a map of strings is
  // ignored rather than written back as text.
  defaultToForm: (defaultValue) =>
    isStringMap(defaultValue) ? objectToRows(defaultValue) : undefined,
  // A rule reads the map the field writes, so it can be written against the
  // API. With no entries that is an empty map, not a missing one: the API
  // reads the two alike, and `in` and `size()` then need no guard of the map.
  toCel: (formValue) => toMap(formValue) ?? {},
  ownsWholeValue: true,
};

export const getMapFieldCodec: FieldCodecResolver = (component) =>
  isMapComponent(component) ? MAP_FIELD_CODEC : undefined;
