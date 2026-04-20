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
import type {
  FormMode,
  Section,
} from 'components/ui-generator/ui-generator.types';
import { buildShapeFromComponents } from './build-shape-from-components';
import { convertToNestedSchema } from './convert-to-nested-schema';
import { applyCelValidation } from './apply-cel-validation';

export type BuildSectionSchemaOptions = {
  formMode?: FormMode;
  originalData?: Record<string, unknown>;
};

/*
 Builds a Zod schema scoped to a single section's fields with `.passthrough()`
so that values from other sections (present in form state for CEL) are not stripped.

CEL expressions are collected from ALL sections so cross-field validation still works.
*/
export const buildSectionZodSchema = (
  sectionKey: string,
  allSections: Record<string, Section>,
  options?: BuildSectionSchemaOptions
): { schema: z.ZodTypeAny; celDependencyGroups: string[][] } => {
  const targetSection = allSections[sectionKey];
  if (!targetSection?.components) {
    return {
      schema: z.object({}).passthrough(),
      celDependencyGroups: [],
    };
  }

  // Build Zod shape only for the target section
  const { schemaShape } = buildShapeFromComponents(
    targetSection.components,
    sectionKey,
    options?.formMode
  );

  // Collect CEL from ALL sections (cross-field validation)
  const allCelExpValidations: ReturnType<
    typeof buildShapeFromComponents
  >['celExpValidations'] = [];
  const allCelDependencyGroups: string[][] = [];

  Object.entries(allSections).forEach(([secKey, section]) => {
    if (section?.components) {
      const result = buildShapeFromComponents(
        section.components,
        secKey,
        options?.formMode
      );
      allCelExpValidations.push(...result.celExpValidations);
      allCelDependencyGroups.push(...result.celDependencyGroups);
    }
  });

  // Convert flat section schema to nested structure
  const nestedFields = convertToNestedSchema(schemaShape);
  let zodSchema: z.ZodTypeAny = z.object(nestedFields).passthrough();

  // Apply CEL validation from all sections
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
