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

import { FieldType } from '../../ui-generator.types';
import type { FieldCodecResolver } from '../field-codecs/field-codecs.types';
import { quantitiesEqual, readQuantity, writeQuantity } from './quantity';

// The form keeps a quantity as a number in the field's unit; the API takes a
// Kubernetes quantity string. A schema default is that number already.
export const getQuantityFieldCodec: FieldCodecResolver = (component) => {
  if (component.uiType !== FieldType.Quantity) return undefined;
  const { unit } = component.fieldParams;

  return {
    toForm: (apiValue) => readQuantity(apiValue, unit),
    toApi: (formValue, stored) => {
      if (formValue === undefined || formValue === null || formValue === '') {
        return formValue;
      }
      // A number that still stands for the stored quantity is written as
      // stored: "10240Mi" stays "10240Mi" instead of becoming "10Gi".
      if (stored !== undefined && quantitiesEqual(formValue, stored, unit)) {
        return stored;
      }
      const amount =
        typeof formValue === 'number' ? formValue : Number(formValue);
      return Number.isFinite(amount) ? writeQuantity(amount, unit) : formValue;
    },
  };
};
