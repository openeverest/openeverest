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
  cpuParser,
  isKubernetesMemoryUnit,
  memoryParser,
} from 'utils/k8ResourceParser';

// Form numbers that round-trip through Kubernetes milli-normalisation can differ
// by a few ULPs; treat them as equal when they match to this absolute tolerance.
const QUANTITY_EQUALITY_EPSILON = 1e-9;

const MEMORY_SUFFIX_WITHOUT_MILLI =
  /^(?<amount>[0-9.]+)(?<suffix>Ki|Mi|Gi|Ti|Pi|Ei|k|M|G|T|P|E)$/;

const isEmptyQuantity = (value: unknown): boolean =>
  value === undefined || value === null || value === '';

const formatAmount = (amount: number): string => {
  if (Number.isInteger(amount)) {
    return String(amount);
  }
  // Avoid scientific notation for typical resource values; trim float noise.
  return String(parseFloat(amount.toPrecision(15)));
};

/**
 * API → form: any Kubernetes quantity into a number in `unit`.
 * Without `unit`, CPU-style values become cores; memory-suffixed values become bytes.
 * Bare `m` without `unit` is millicores (CPU), not milli-bytes — memory fields
 * always declare `unit` (e.g. Gi), where `m` is milli-bytes.
 */
export const readQuantity = (
  raw: unknown,
  unit?: string
): number | undefined => {
  if (isEmptyQuantity(raw)) {
    return undefined;
  }

  const targetUnit =
    unit && isKubernetesMemoryUnit(unit) ? unit : undefined;

  if (typeof raw === 'number') {
    if (!Number.isFinite(raw)) {
      return undefined;
    }
    // A JSON number is the base unit: cores (CPU) or bytes (memory).
    return targetUnit ? convertBytesToMemoryUnit(raw, targetUnit) : raw;
  }

  if (typeof raw !== 'string') {
    return undefined;
  }

  const trimmed = raw.trim();
  if (trimmed === '') {
    return undefined;
  }

  if (targetUnit) {
    const { value } = memoryParser(trimmed, targetUnit);
    return Number.isFinite(value) ? value : undefined;
  }

  // No unit: memory suffixes (except bare `m`) → bytes; otherwise → cores.
  const memoryMatch = MEMORY_SUFFIX_WITHOUT_MILLI.exec(trimmed);
  if (memoryMatch?.groups) {
    const amount = parseFloat(memoryMatch.groups.amount);
    const suffix = memoryMatch.groups.suffix;
    if (
      Number.isFinite(amount) &&
      suffix &&
      isKubernetesMemoryUnit(suffix)
    ) {
      return convertMemoryUnitToBytes(amount, suffix);
    }
  }

  const cores = cpuParser(trimmed);
  return Number.isFinite(cores) ? cores : undefined;
};

/**
 * Form → API: always a quantity string. Fractional CPU is `"0.5"`, never a JSON float.
 */
export const writeQuantity = (amount: number, unit?: string): string => {
  const formatted = formatAmount(amount);
  return unit ? `${formatted}${unit}` : formatted;
};

/**
 * Display: value with its unit label (`0.5 cores`, `10 Gi`).
 * Numbers are treated as already in `unit`; strings are read first.
 */
export const formatQuantity = (
  rawOrNumber: unknown,
  unit?: string,
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
  unit?: string
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
