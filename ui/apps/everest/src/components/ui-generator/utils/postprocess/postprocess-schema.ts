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
  Component,
  ComponentGroup,
  TopologyUISchemas,
} from '../../ui-generator.types';
import { getComponentTargetPaths } from '../preprocess/normalized-component';

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

const isPlainObject = (value: unknown): value is Record<string, unknown> =>
  typeof value === 'object' && value !== null && !Array.isArray(value);

const deepClone = <T>(value: T): T => {
  if (typeof structuredClone === 'function') {
    return structuredClone(value);
  }

  return JSON.parse(JSON.stringify(value)) as T;
};

const getByPath = (obj: Record<string, unknown>, path: string): unknown => {
  if (typeof path !== 'string' || path.length === 0) {
    if (import.meta.env.DEV) {
      // eslint-disable-next-line no-console
      console.error('[postprocessSchema][getByPath] Invalid path', {
        path,
        pathType: typeof path,
      });
    }
    return undefined;
  }

  return path.split('.').reduce<unknown>((current, key) => {
    if (!isPlainObject(current)) {
      return undefined;
    }
    return current[key];
  }, obj);
};

const setByPath = (
  obj: Record<string, unknown>,
  path: string,
  value: unknown
): void => {
  if (typeof path !== 'string' || path.length === 0) {
    if (import.meta.env.DEV) {
      // eslint-disable-next-line no-console
      console.error('[postprocessSchema][setByPath] Invalid path', {
        path,
        pathType: typeof path,
        value,
      });
    }
    return;
  }

  const parts = path.split('.');
  let current: Record<string, unknown> = obj;

  for (let i = 0; i < parts.length - 1; i++) {
    const key = parts[i];
    const next = current[key];

    if (!isPlainObject(next)) {
      current[key] = {};
    }

    current = current[key] as Record<string, unknown>;
  }

  current[parts[parts.length - 1]] = value;
};

const deleteByPath = (obj: Record<string, unknown>, path: string): void => {
  if (typeof path !== 'string' || path.length === 0) {
    if (import.meta.env.DEV) {
      // eslint-disable-next-line no-console
      console.error('[postprocessSchema][deleteByPath] Invalid path', {
        path,
        pathType: typeof path,
      });
    }
    return;
  }

  const parts = path.split('.');
  let current: Record<string, unknown> = obj;

  for (let i = 0; i < parts.length - 1; i++) {
    const key = parts[i];
    const next = current[key];

    if (!isPlainObject(next)) {
      return;
    }

    current = next;
  }

  delete current[parts[parts.length - 1]];
};

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
  const topology = schema[selectedTopology];
  if (!topology || !topology.sections) {
    return [];
  }

  const mappings: MultiPathMapping[] = [];

  const walk = (
    components: Record<string, Component | ComponentGroup>,
    basePath = ''
  ) => {
    Object.entries(components).forEach(([key, item]) => {
      const generatedName = basePath ? `${basePath}.${key}` : key;

      if (item.uiType === 'group' || item.uiType === 'hidden') {
        walk((item as ComponentGroup).components, generatedName);
        return;
      }

      const component = item as Component;
      const normalizedPaths = getComponentTargetPaths(component);
      if (normalizedPaths.length <= 1) {
        return;
      }

      const sourceFieldId = normalizedPaths[0];
      const targetPaths = normalizedPaths.filter(
        (path) => path !== sourceFieldId
      );

      if (targetPaths.length === 0) {
        return;
      }

      mappings.push({
        sourceFieldId,
        targetPaths,
        removeSourceField: false,
      });
    });
  };

  Object.values(topology.sections).forEach((section) => {
    if (section?.components) {
      walk(section.components);
    }
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
  return removeEmptyFieldValues(mapped);
};
