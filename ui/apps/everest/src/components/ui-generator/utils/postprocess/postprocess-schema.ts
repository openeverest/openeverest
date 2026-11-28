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
  formatPath,
  isPlainObject,
  deepClone,
  getByPath,
  parsePath,
  setByPath,
  deleteByPath,
  isEmptyFieldValue,
  toFormId,
} from '../object-path';
import {
  collectAllSchemaPaths,
  walkLeafComponents,
  walkTopologyComponents,
} from '../schema-walker';
import {
  dropOtherTopologyFormValues,
  dropOtherTopologyValues,
  isPathRetainedByTopology,
} from '../topology-scope';
import {
  getComponentSourcePath,
  getComponentTargetPaths,
} from '../preprocess/normalized-component';
import { getWidgetTargetPaths } from '../widget-targets';
import {
  extractBadgeMappings,
  applyBadgesToFormData,
} from '../badge-to-api/badge-to-api';
import {
  extractQuantityMappings,
  applyQuantitiesToFormData,
} from '../quantity';
import {
  TOGGLEABLE_SWITCHES_KEY,
  getInactiveToggleablePaths,
} from '../toggleable/toggleable';

export type PostprocessInput = Record<string, unknown>;

export type MultiPathMapping = {
  sourceFieldId: string;
  targetPaths: string[];
  removeSourceField?: boolean;
};

export type PostprocessOptions = {
  schema?: TopologyUISchemas;
  selectedTopology?: string;
  // API paths whose values are registered in RHF. Used when a caller has
  // section metadata but cannot provide a selected topology.
  registeredApiPaths?: string[];
  multiPathMappings?: MultiPathMapping[];
  // Instance / restore source at the same paths as form data. Used so an
  // untouched quantity field is written back as stored (e.g. "10240Mi").
  originalData?: Record<string, unknown>;
};

const normalizeRuntimePathArray = (paths: unknown): string[] => {
  if (!Array.isArray(paths)) {
    return [];
  }

  return Array.from(
    new Set(paths.filter((p): p is string => typeof p === 'string' && !!p))
  );
};

type RegisteredPathNode = {
  isField: boolean;
  children: Map<
    string,
    {
      apiSegment: string;
      apiPath: string;
      node: RegisteredPathNode;
    }
  >;
};

const createRegisteredPathNode = (): RegisteredPathNode => ({
  isField: false,
  children: new Map(),
});

const collectRegisteredApiPaths = (
  schema: TopologyUISchemas,
  selectedTopology: string
): string[] => {
  const paths = new Set<string>();
  const sections = schema[selectedTopology]?.sections;

  if (!sections) return [];

  Object.values(sections).forEach((section) => {
    if (!section?.components) return;

    walkLeafComponents(section.components, ({ component }) => {
      const sourcePath = getComponentSourcePath(component);
      if (sourcePath) paths.add(sourcePath);

      // extractInstanceValues stores widget targets at toFormId(path), so
      // these paths, unlike other widget output destinations, are RHF keys.
      getWidgetTargetPaths(component).forEach((path) => paths.add(path));
    });
  });

  return Array.from(paths);
};

const collectRetainedRegisteredApiPaths = (
  schema: TopologyUISchemas,
  selectedTopology: string
): string[] => {
  const selectedPaths = Array.from(
    collectAllSchemaPaths(schema[selectedTopology]?.sections ?? {})
  ).map(toFormId);
  const selectedRegisteredPaths = collectRegisteredApiPaths(
    schema,
    selectedTopology
  );
  const retainedPaths = Object.keys(schema)
    .filter((topology) => topology !== selectedTopology)
    .flatMap((topology) => collectRegisteredApiPaths(schema, topology))
    .filter((path) => isPathRetainedByTopology(toFormId(path), selectedPaths));

  return Array.from(new Set([...selectedRegisteredPaths, ...retainedPaths]));
};

const createRegisteredPathTree = (apiPaths: string[]): RegisteredPathNode => {
  const root = createRegisteredPathNode();

  apiPaths.forEach((apiPath) => {
    if (!apiPath) return;

    const apiSegments = parsePath(apiPath);
    const formSegments = toFormId(apiPath).split('.');
    if (apiSegments.length !== formSegments.length) {
      throw new Error(`Cannot register malformed API path "${apiPath}".`);
    }

    let node = root;
    apiSegments.forEach((apiSegment, index) => {
      const formSegment = formSegments[index];
      let child = node.children.get(formSegment);
      if (!child) {
        child = {
          apiSegment,
          apiPath: formatPath(apiSegments.slice(0, index + 1)),
          node: createRegisteredPathNode(),
        };
        node.children.set(formSegment, child);
      } else if (child.apiSegment !== apiSegment) {
        throw new Error(
          `Registered API paths map the form segment "${formSegment}" to multiple API keys.`
        );
      }

      node = child.node;
    });

    node.isField = true;
  });

  return root;
};

const toApiShapedValues = (
  value: unknown,
  registeredPaths: RegisteredPathNode
): unknown => {
  if (registeredPaths.children.size === 0 || !isPlainObject(value)) {
    return value;
  }

  const result = { ...value };
  registeredPaths.children.forEach(
    ({ apiSegment, apiPath, node }, formSegment) => {
      if (!Object.prototype.hasOwnProperty.call(value, formSegment)) return;

      const childValue = value[formSegment];
      const apiValue = toApiShapedValues(childValue, node);
      if (formSegment === apiSegment) {
        if (apiValue !== childValue) result[formSegment] = apiValue;
        return;
      }

      if (Object.prototype.hasOwnProperty.call(value, apiSegment)) {
        throw new Error(
          `Cannot convert registered form path "${apiPath}": API key "${apiSegment}" already exists alongside "${formSegment}".`
        );
      }

      delete result[formSegment];
      result[apiSegment] = apiValue;
    }
  );
  return result;
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

  const formValuesForSelectedTopology =
    options?.schema && options.selectedTopology
      ? dropOtherTopologyFormValues(
          formValues,
          options.schema,
          options.selectedTopology
        )
      : formValues;

  const registeredApiPaths = [
    ...(options?.schema && options.selectedTopology
      ? collectRetainedRegisteredApiPaths(
          options.schema,
          options.selectedTopology
        )
      : []),
    ...(options?.registeredApiPaths ?? []),
  ];
  const apiValues = toApiShapedValues(
    formValuesForSelectedTopology,
    createRegisteredPathTree(registeredApiPaths)
  ) as PostprocessInput;

  const scoped =
    options?.schema && options.selectedTopology
      ? dropOtherTopologyValues(
          apiValues,
          options.schema,
          options.selectedTopology
        )
      : apiValues;

  const mapped = applyMultiPathMappings(scoped, allMappings);

  const coerced =
    options?.schema && options.selectedTopology
      ? coerceSchemaNumbers(mapped, options.schema, options.selectedTopology)
      : mapped;

  // Quantity paths must become strings before badge append; they are never
  // coerced to JSON numbers (that is what broke fractional CPU).
  const quantityMappings =
    options?.schema && options.selectedTopology
      ? extractQuantityMappings(options.schema, options.selectedTopology)
      : [];
  const withQuantities = applyQuantitiesToFormData(
    coerced,
    quantityMappings,
    options?.originalData
  );

  const badgeMappings =
    options?.schema && options.selectedTopology
      ? extractBadgeMappings(options.schema, options.selectedTopology)
      : [];
  const withBadges = applyBadgesToFormData(withQuantities, badgeMappings);

  const sections =
    options?.schema && options.selectedTopology
      ? options.schema[options.selectedTopology]?.sections
      : undefined;
  const withoutInactive = deepClone(withBadges);
  (sections ? getInactiveToggleablePaths(sections, formValues) : []).forEach(
    (path) => deleteByPath(withoutInactive, path)
  );
  deleteByPath(withoutInactive, TOGGLEABLE_SWITCHES_KEY);

  return removeEmptyFieldValues(withoutInactive);
};
