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

import { FieldType, type TopologyUISchemas } from '../../ui-generator.types';
import { walkTopologyComponents } from '../schema-walker';
import { getComponentTargetPaths } from '../preprocess/normalized-component';
import { deepClone, getByPath, setByPath } from '../object-path';
import type { kubernetesUnit } from 'utils/k8ResourceParser';
import { quantitiesEqual, writeQuantity } from './quantity';

export type QuantityMapping = {
  path: string;
  unit?: kubernetesUnit;
};

export const extractQuantityMappings = (
  schema: TopologyUISchemas,
  selectedTopology: string
): QuantityMapping[] => {
  const mappings: QuantityMapping[] = [];

  walkTopologyComponents(schema, selectedTopology, ({ component }) => {
    if (component.uiType !== FieldType.Quantity) {
      return;
    }
    getComponentTargetPaths(component).forEach((path) => {
      mappings.push({ path, unit: component.fieldParams.unit });
    });
  });

  return mappings;
};

/*
 Writes quantity fields as Kubernetes quantity strings. When originalData is
 provided and the form number still represents the stored quantity, the stored
 value is kept (e.g. "10240Mi" stays "10240Mi" instead of becoming "10Gi").
*/
export const applyQuantitiesToFormData = (
  formData: Record<string, unknown>,
  mappings: QuantityMapping[],
  originalData?: Record<string, unknown>
): Record<string, unknown> => {
  if (mappings.length === 0) {
    return formData;
  }

  const result = deepClone(formData);

  mappings.forEach(({ path, unit }) => {
    const value = getByPath(result, path);
    if (value === undefined || value === null || value === '') {
      return;
    }

    if (originalData) {
      const stored = getByPath(originalData, path);
      if (stored !== undefined && quantitiesEqual(value, stored, unit)) {
        setByPath(result, path, stored);
        return;
      }
    }

    const amount = typeof value === 'number' ? value : Number(value);
    if (!Number.isFinite(amount)) {
      return;
    }

    setByPath(result, path, writeQuantity(amount, unit));
  });

  return result;
};
