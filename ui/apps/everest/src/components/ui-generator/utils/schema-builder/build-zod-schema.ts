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

import { z } from 'zod';
import {
  FormMode,
  Topology,
  TopologyUISchemas,
} from 'components/ui-generator/ui-generator.types';
import { buildShapeFromComponents } from './build-shape-from-components';
import { convertToNestedSchema } from './convert-to-nested-schema';
import { applyCelValidation } from './apply-cel-validation';

export type BuildSchemaOptions = {
  formMode?: FormMode;
  originalData?: Record<string, unknown>;
};

export const buildZodSchema = (
  schema: TopologyUISchemas,
  selectedTopology: string,
  options?: BuildSchemaOptions
): { schema: z.ZodTypeAny; celDependencyGroups: string[][] } => {
  const topology: Topology = schema[selectedTopology];

  if (!topology || !topology.sections) {
    return {
      schema: z.object({}).passthrough(),
      celDependencyGroups: [],
    };
  }

  const flatFields: Record<string, z.ZodTypeAny> = {};
  const allCelExpValidations: ReturnType<
    typeof buildShapeFromComponents
  >['celExpValidations'] = [];
  const allCelDependencyGroups: string[][] = [];

  // Build schema from all sections
  Object.entries(topology.sections).forEach(([sectionKey, section]) => {
    if (section?.components) {
      const result = buildShapeFromComponents(
        section.components,
        sectionKey,
        options?.formMode
      );

      Object.assign(flatFields, result.schemaShape);
      allCelExpValidations.push(...result.celExpValidations);
      allCelDependencyGroups.push(...result.celDependencyGroups);
    }
  });

  // Convert flat schema to nested structure
  const nestedFields = convertToNestedSchema(flatFields);
  let zodSchema: z.ZodTypeAny = z.object(nestedFields).passthrough();

  // Apply CEL validation if needed
  zodSchema = applyCelValidation(
    zodSchema,
    allCelExpValidations,
    options?.originalData
  );

  return {
    schema: zodSchema,
    celDependencyGroups: allCelDependencyGroups,
  };
};
