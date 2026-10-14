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

import { getQuantityUnitSize, parseQuantity } from 'utils/k8ResourceParser';
import type {
  QuantityFieldParams,
  QuantityUnit,
} from '../../ui-generator.types';

// 12 significant digits drop the float noise of unit conversion (0.3 / 0.001).
const roundAmount = (amount: number): number => Number(amount.toPrecision(12));

// A stored quantity as a number in `unit` ("10Gi" → 10 for Gi, "500m" → 0.5 for
// cores). A value that is not a quantity is returned as is.
export const readQuantity = (value: unknown, unit: QuantityUnit): unknown => {
  const base =
    typeof value === 'number'
      ? value
      : typeof value === 'string'
        ? parseQuantity(value)
        : undefined;
  return base === undefined
    ? value
    : roundAmount(base / getQuantityUnitSize(unit));
};

// A number in `unit` as the quantity string the API expects: a string is valid
// on every int-or-string path, where a float is rejected.
export const writeQuantity = (value: unknown, unit: QuantityUnit): unknown =>
  typeof value === 'number' ? `${roundAmount(value)}${unit}` : value;

export const formatQuantity = (
  amount: number,
  { unit = '', unitLabel }: QuantityFieldParams
): string => {
  const label = unitLabel ?? unit;
  return label ? `${amount} ${label}` : String(amount);
};
