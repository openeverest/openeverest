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

import type { Component } from '../../ui-generator.types';

// How a field whose value in the form differs from the one in the API crosses
// between the two, e.g. a map kept as rows or a quantity kept as a number.
// Most fields hold the same value on both sides and have no codec.
export interface FieldCodec {
  // A saved value, as the form keeps it.
  toForm: (apiValue: unknown) => unknown;
  // The form's value, as the API takes it; `undefined` writes nothing.
  // `stored` is the saved value at the same path, when there is one.
  toApi: (formValue: unknown, stored: unknown) => unknown;
  // A schema `defaultValue`, as the form keeps it. Left out by a field whose
  // default is written the way the form keeps it already.
  defaultToForm?: (defaultValue: unknown) => unknown;
  // The value is the field's as a whole: an edit replaces the saved one
  // instead of merging into it, and nothing inside is dropped as empty.
  ownsWholeValue?: boolean;
}

export type FieldCodecResolver = (
  component: Component
) => FieldCodec | undefined;
