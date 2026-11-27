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
import type { FieldCodecResolver } from '../field-codecs/field-codecs.types';
import { createRowsFieldCodec } from '../rows-field/rows-field-codec';
import { toList } from './list-field';
import { LIST_FIELD_KIND } from './list-field-kind';

// The form keeps a list as rows ({ value }); the API takes the list. No entry
// leaves no list, so a rule reads what is written.
const LIST_FIELD_CODEC = createRowsFieldCodec(LIST_FIELD_KIND, {
  toApi: toList,
  toCel: toList,
});

export const getListFieldCodec: FieldCodecResolver = (component) =>
  isListComponent(component) ? LIST_FIELD_CODEC : undefined;
