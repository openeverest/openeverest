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

import { Resources } from 'shared-types/dbCluster.types';

type kubernetesUnit =
  | 'k'
  | 'M'
  | 'G'
  | 'T'
  | 'P'
  | 'E'
  | 'Ki'
  | 'Mi'
  | 'Gi'
  | 'Ti'
  | 'Pi'
  | 'Ei';

const memoryMultipliers: Record<kubernetesUnit, number> = {
  k: 10 ** 3,
  M: 10 ** 6,
  G: 10 ** 9,
  T: 10 ** 12,
  P: 10 ** 15,
  E: 10 ** 18,
  Ki: 1024,
  Mi: 1024 ** 2,
  Gi: 1024 ** 3,
  Ti: 1024 ** 4,
  Pi: 1024 ** 5,
  Ei: 1024 ** 6,
};

export const cpuParser = (input: string) => {
  const milliMatch = input.match(/^([0-9]+)m$/);

  if (milliMatch) {
    // @ts-ignore
    return milliMatch[1] / 1000;
  }

  return parseFloat(input);
};

export const memoryParser = (
  input: string,
  targetUnit?: kubernetesUnit
): {
  value: number;
  originalUnit: kubernetesUnit | '';
} => {
  const unitMatch = input.match(/^([0-9.]+)([A-Za-z]{1,2})$/);

  if (unitMatch) {
    const value = parseFloat(unitMatch[1]);
    // @ts-ignore
    const sourceUnit: kubernetesUnit = unitMatch[2];
    const valueBytes = value * memoryMultipliers[sourceUnit];

    return {
      value: targetUnit
        ? valueBytes / memoryMultipliers[targetUnit]
        : parseFloat(input),
      originalUnit: sourceUnit,
    };
  }

  return {
    value: parseFloat(input),
    originalUnit: '',
  };
};

export const getResourcesDetailedString = (value: number, unit: string) => {
  return `${value} ${unit}`;
};

export const getTotalResourcesDetailedString = (
  value: number,
  numberOfNodes: number,
  unit: string,
  shardNr?: number,
  sharding?: boolean
) => {
  if (numberOfNodes === 1 && !sharding) {
    return `${value.toFixed(2)} ${unit}`;
  }

  const totalResources =
    sharding && shardNr
      ? value * numberOfNodes * shardNr
      : value * numberOfNodes;

  const formattedTotalResources =
    totalResources === Math.trunc(totalResources)
      ? totalResources.toString()
      : parseFloat(totalResources.toFixed(2));

  return `${formattedTotalResources} ${unit}`;
};

export const extractResourceCpuValue = (
  resources?: Resources
): number | string => resources?.limits?.cpu ?? resources?.cpu ?? 0;

export const extractResourceMemoryValue = (
  resources?: Resources
): number | string => resources?.limits?.memory ?? resources?.memory ?? 0;

// Legacy clusters (before the requests/limits split) stored a single flat
// cpu/memory value with no explicit requests.
// When a legacy cluster is edited with requests kept synced we intentionally
// write only limits (to avoid an unnecessary restart), leaving the flat
// cpu/memory fields as residual "0" values. In that hybrid state the effective
// request equals the limit (Kubernetes defaults absent requests to the limit),
// so we must fall back to the limit before the residual flat value.
export const extractResourceCpuRequestValue = (
  resources?: Resources
): number | string | undefined =>
  resources?.requests?.cpu ?? resources?.limits?.cpu ?? resources?.cpu;

export const extractResourceMemoryRequestValue = (
  resources?: Resources
): number | string | undefined =>
  resources?.requests?.memory ?? resources?.limits?.memory ?? resources?.memory;

// Tolerance for comparing parsed resource values. Memory parsing round-trips
// through byte multipliers (G -> bytes -> G), so two logically-equal values can
// differ in the last floating-point bit; a strict === would report a false
// mismatch. CPU/memory granularity is far coarser than this epsilon.
const RESOURCE_COMPARISON_EPSILON = 1e-9;

export const areResourceRequestsSynced = (resources?: Resources): boolean => {
  const cpuLimit = cpuParser(extractResourceCpuValue(resources).toString());
  const memoryLimit = memoryParser(
    extractResourceMemoryValue(resources).toString(),
    'G'
  ).value;

  const rawCpuRequest = extractResourceCpuRequestValue(resources);
  const rawMemoryRequest = extractResourceMemoryRequestValue(resources);

  const cpuSynced =
    rawCpuRequest === undefined ||
    Math.abs(cpuParser(rawCpuRequest.toString()) - cpuLimit) <
      RESOURCE_COMPARISON_EPSILON;
  const memorySynced =
    rawMemoryRequest === undefined ||
    Math.abs(
      memoryParser(rawMemoryRequest.toString(), 'G').value - memoryLimit
    ) < RESOURCE_COMPARISON_EPSILON;

  return cpuSynced && memorySynced;
};
