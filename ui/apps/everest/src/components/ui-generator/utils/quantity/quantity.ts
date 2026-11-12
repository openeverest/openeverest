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

import {
  convertBytesToMemoryUnit,
  convertMemoryUnitToBytes,
  isKubernetesMemoryUnit,
  type kubernetesUnit,
} from 'utils/k8ResourceParser';

// Form numbers that round-trip through Kubernetes milli-normalisation can differ
// by a few ULPs; treat them as equal when they match to this absolute tolerance.
const QUANTITY_EQUALITY_EPSILON = 1e-9;

// Kubernetes quantity: optional suffix is always a fraction/multiple of the base
// unit (cores for CPU, bytes for memory) — the same multipliers apply to both.
const QUANTITY_STRING =
  /^(?<amount>[0-9.]+)(?<suffix>Ki|Mi|Gi|Ti|Pi|Ei|m|k|M|G|T|P|E)?$/;

const isEmptyQuantity = (value: unknown): boolean =>
  value === undefined || value === null || value === '';

const formatAmount = (amount: number): string => {
  if (Number.isInteger(amount)) {
    return String(amount);
  }
  // Avoid scientific notation for typical resource values; trim float noise.
  return String(parseFloat(amount.toPrecision(15)));
};

/** Parse a Kubernetes quantity string into its base unit (cores or bytes). */
const parseToBaseUnit = (raw: string): number | undefined => {
  const match = QUANTITY_STRING.exec(raw);
  if (!match?.groups) {
    return undefined;
  }

  const amount = parseFloat(match.groups.amount);
  if (!Number.isFinite(amount)) {
    return undefined;
  }

  const suffix = match.groups.suffix;
  if (!suffix) {
    // Bare number is already the base unit (Kubernetes canonical form).
    return amount;
  }

  if (!isKubernetesMemoryUnit(suffix)) {
    return undefined;
  }

  return convertMemoryUnitToBytes(amount, suffix);
};

/**
 * API → form: any Kubernetes quantity into a number in `unit`.
 * Without `unit`, the value stays in the base unit (cores / bytes).
 * Bare numbers are always the base unit — Kubernetes itself emits them when
 * it canonicalises (e.g. `"1000m"` → `"1"`).
 */
export const readQuantity = (
  raw: unknown,
  unit?: kubernetesUnit
): number | undefined => {
  if (isEmptyQuantity(raw)) {
    return undefined;
  }

  const targetUnit = unit && isKubernetesMemoryUnit(unit) ? unit : undefined;

  let base: number | undefined;

  if (typeof raw === 'number') {
    if (!Number.isFinite(raw)) {
      return undefined;
    }
    base = raw;
  } else if (typeof raw === 'string') {
    const trimmed = raw.trim();
    if (trimmed === '') {
      return undefined;
    }
    base = parseToBaseUnit(trimmed);
  } else {
    return undefined;
  }

  if (base === undefined) {
    return undefined;
  }

  return targetUnit ? convertBytesToMemoryUnit(base, targetUnit) : base;
};

/**
 * Form → API: always a quantity string. Fractional CPU is `"0.5"`, never a JSON float.
 */
export const writeQuantity = (
  amount: number,
  unit?: kubernetesUnit
): string => {
  const formatted = formatAmount(amount);
  return unit ? `${formatted}${unit}` : formatted;
};

/**
 * Display: value with its unit label (`0.5 cores`, `10 Gi`).
 * Numbers are treated as already in `unit`; strings are read first.
 */
export const formatQuantity = (
  rawOrNumber: unknown,
  unit?: kubernetesUnit,
  unitLabel?: string
): string | undefined => {
  const amount =
    typeof rawOrNumber === 'number'
      ? rawOrNumber
      : readQuantity(rawOrNumber, unit);

  if (amount === undefined || !Number.isFinite(amount)) {
    return undefined;
  }

  const label = unitLabel ?? unit;
  const formatted = formatAmount(amount);
  return label ? `${formatted} ${label}` : formatted;
};

/**
 * True when the form number represents the same quantity as the stored API value,
 * so an untouched field can be written back as stored (e.g. `"10240Mi"` stays).
 */
export const quantitiesEqual = (
  formAmount: unknown,
  storedRaw: unknown,
  unit?: kubernetesUnit
): boolean => {
  if (isEmptyQuantity(formAmount)) {
    return isEmptyQuantity(storedRaw);
  }

  const formNumber =
    typeof formAmount === 'number' ? formAmount : Number(formAmount);
  if (!Number.isFinite(formNumber)) {
    return false;
  }

  const storedNumber = readQuantity(storedRaw, unit);
  if (storedNumber === undefined) {
    return false;
  }

  return Math.abs(formNumber - storedNumber) < QUANTITY_EQUALITY_EPSILON;
};
