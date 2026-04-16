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

import { TopologyUISchemas } from '../../ui-generator.types';
import {
  isPlainObject,
  deepClone,
  getByPath,
  setByPath,
  deleteByPath,
} from '../object-path';
import { walkTopologyComponents } from '../schema-walker';
import { getComponentTargetPaths } from '../preprocess/normalized-component';
import {
  extractBadgeMappings,
  applyBadgesToFormData,
} from '../badge-to-api/badge-to-api';

export type PostprocessInput = Record<string, unknown>;

export type MultiPathMapping = {
  sourceFieldId: string;
  targetPaths: string[];
  removeSourceField?: boolean;
};

export type PostprocessOptions = {
  schema?: TopologyUISchemas;
  selectedTopology?: string;
  multiPathMappings?: MultiPathMapping[];
};

// Empty value contract (applies to all field types by default):
// - Removed: undefined, null, ''
// - Preserved: false, 0, [], non-empty objects
export const isEmptyFieldValue = (value: unknown): boolean =>
  value === undefined || value === null || value === '';

const normalizeRuntimePathArray = (paths: unknown): string[] => {
  if (!Array.isArray(paths)) {
    return [];
  }

  return Array.from(
    new Set(paths.filter((p): p is string => typeof p === 'string' && !!p))
  );
};

export const extractMultiPathMappings = (
  schema: TopologyUISchemas,
  selectedTopology: string
): MultiPathMapping[] => {
  const mappings: MultiPathMapping[] = [];

  walkTopologyComponents(schema, selectedTopology, ({ component }) => {
    const normalizedPaths = getComponentTargetPaths(component);
    if (normalizedPaths.length <= 1) return;

    const sourceFieldId = normalizedPaths[0];
    const targetPaths = normalizedPaths.filter(
      (path) => path !== sourceFieldId
    );

    if (targetPaths.length === 0) return;

    mappings.push({
      sourceFieldId,
      targetPaths,
      removeSourceField: false,
    });
  });

  return mappings;
};

export const applyMultiPathMappings = (
  formValues: PostprocessInput,
  mappings: MultiPathMapping[]
): PostprocessInput => {
  if (mappings.length === 0) {
    return formValues;
  }

  const result = deepClone(formValues);

  mappings.forEach(
    ({ sourceFieldId, targetPaths, removeSourceField = true }) => {
      if (typeof sourceFieldId !== 'string' || !sourceFieldId) {
        return;
      }

      const normalizedTargetPaths = normalizeRuntimePathArray(targetPaths);
      if (normalizedTargetPaths.length === 0) {
        if (removeSourceField) {
          deleteByPath(result, sourceFieldId);
        }
        return;
      }

      const sourceValue = getByPath(result, sourceFieldId);

      if (sourceValue === undefined) {
        return;
      }

      normalizedTargetPaths.forEach((targetPath) => {
        setByPath(result, targetPath, sourceValue);
      });

      if (removeSourceField) {
        deleteByPath(result, sourceFieldId);
      }
    }
  );

  return result;
};

export const removeEmptyFieldValues = (
  input: PostprocessInput
): PostprocessInput => {
  const result = deepClone(input);

  const prune = (obj: Record<string, unknown>) => {
    Object.keys(obj).forEach((key) => {
      const value = obj[key];

      if (isEmptyFieldValue(value)) {
        delete obj[key];
        return;
      }

      if (!isPlainObject(value)) {
        return;
      }

      prune(value);

      if (Object.keys(value).length === 0) {
        delete obj[key];
      }
    });
  };

  prune(result);
  return result;
};

// TODO this can be refactored with adding separate number input into ui-lib
export const coerceSchemaNumbers = (
  input: PostprocessInput,
  schema: TopologyUISchemas,
  selectedTopology: string
): PostprocessInput => {
  const result = deepClone(input);
  const numberPaths: string[] = [];

  walkTopologyComponents(schema, selectedTopology, ({ component }) => {
    if (component.uiType === 'number') {
      const paths = getComponentTargetPaths(component);
      numberPaths.push(...paths);
    }
  });

  numberPaths.forEach((path) => {
    const value = getByPath(result, path);
    if (typeof value === 'string' && value !== '') {
      const num = Number(value);
      if (!Number.isNaN(num)) {
        setByPath(result, path, num);
      }
    }
  });

  return result;
};

export const postprocessSchemaData = (
  formValues: PostprocessInput,
  options?: PostprocessOptions
): PostprocessInput => {
  const extractedMappings =
    options?.schema && options.selectedTopology
      ? extractMultiPathMappings(options.schema, options.selectedTopology)
      : [];

  const allMappings = [
    ...extractedMappings,
    ...(options?.multiPathMappings ?? []),
  ];

  const mapped = applyMultiPathMappings(formValues, allMappings);

  const coerced =
    options?.schema && options.selectedTopology
      ? coerceSchemaNumbers(mapped, options.schema, options.selectedTopology)
      : mapped;

  const badgeMappings =
    options?.schema && options.selectedTopology
      ? extractBadgeMappings(options.schema, options.selectedTopology)
      : [];
  const withBadges = applyBadgesToFormData(coerced, badgeMappings);

  return removeEmptyFieldValues(withBadges);
};
