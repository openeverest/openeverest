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
  WidgetComponent,
} from 'components/ui-generator/ui-generator.types';
import { UI_TYPE_DEFAULT_VALUE } from 'components/ui-generator/constants';
import { getFieldCodec } from '../field-codecs';

// What the form holds for a field with nothing saved and no schema default:
// what its codec says, e.g. the one empty row of a map or a list, or else the
// empty value of its type. A widget manages its own.
export const getEmptyFormValue = (
  component: Exclude<Component, WidgetComponent>
): unknown =>
  getFieldCodec(component)?.emptyToForm?.() ??
  UI_TYPE_DEFAULT_VALUE[component.uiType];
