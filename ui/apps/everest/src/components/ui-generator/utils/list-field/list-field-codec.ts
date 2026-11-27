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

import { isListComponent } from '../../ui-generator.types';
import type {
  FieldCodec,
  FieldCodecResolver,
} from '../field-codecs/field-codecs.types';
import { isStringList, listToRows, toList } from './list-field';

// The form keeps a list as rows ({ value }); the API takes the list.
const LIST_FIELD_CODEC: FieldCodec = {
  toForm: listToRows,
  toApi: toList,
  // A default is written in the API shape. One that isn't a list of strings
  // is ignored rather than written back as text.
  defaultToForm: (defaultValue) =>
    isStringList(defaultValue) ? listToRows(defaultValue) : undefined,
  // A rule reads what the field writes, so it can be written against the
  // API: the list, and nothing at all while it has no entries.
  toCel: toList,
  ownsWholeValue: true,
};

export const getListFieldCodec: FieldCodecResolver = (component) =>
  isListComponent(component) ? LIST_FIELD_CODEC : undefined;
