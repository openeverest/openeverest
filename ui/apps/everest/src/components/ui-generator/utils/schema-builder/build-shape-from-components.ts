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
  Component,
  ComponentGroup,
  FormMode,
  isWidgetComponent,
} from 'components/ui-generator/ui-generator.types';
import { ZOD_SCHEMA_MAP } from 'components/ui-generator/constants';
import { generateFieldId } from '../component-renderer/generate-field-id';
import { getComponentSourcePath } from '../preprocess/normalized-component';
import { applyValidationFromSchema } from './apply-from-schema';
import { resolveValidationForMode } from '../validation/resolve-validation-for-mode';
import {
  TOGGLEABLE_SWITCHES_KEY,
  getToggleableMeta,
  toggleableSwitchesSchema,
} from '../toggleable/toggleable';
import type {
  CelExpValidation,
  ComponentSchemaResult,
  ToggleableFieldRule,
} from './schema-builder.types';

export const buildShapeFromComponents = (
  components: { [key: string]: Component | ComponentGroup },
  basePath: string = '',
  formMode?: FormMode,
  activeSwitch?: string
): ComponentSchemaResult => {
  const schemaShape: Record<string, z.ZodTypeAny> = {};
  const celExpValidations: CelExpValidation[] = [];
  const celDependencyGroups: string[][] = [];
  const toggleableFieldRules: ToggleableFieldRule[] = [];

  Object.entries(components).forEach(([key, item]) => {
    const generatedName = basePath ? `${basePath}.${key}` : key;
    const fieldId = generateFieldId(item, generatedName);

    // Handle groups recursively
    if (item.uiType === 'group' && 'components' in item) {
      const toggleable = getToggleableMeta(item);
      if (toggleable) {
        schemaShape[TOGGLEABLE_SWITCHES_KEY] = toggleableSwitchesSchema;
      }
      const groupResult = buildShapeFromComponents(
        (item as ComponentGroup).components,
        generatedName,
        formMode,
        toggleable?.switchName ?? activeSwitch
      );

      // Merge nested schemas into parent level (flat structure)
      Object.assign(schemaShape, groupResult.schemaShape);
      celExpValidations.push(...groupResult.celExpValidations);
      celDependencyGroups.push(...groupResult.celDependencyGroups);
      toggleableFieldRules.push(...groupResult.toggleableFieldRules);
      return;
    }

    // Get base Zod schema for this UI type
    const component = item as Component;

    // Disabled fields bypass all validation — they can't be changed by the user
    if (component.fieldParams?.disabled) {
      schemaShape[fieldId] = z.any().optional();
      return;
    }

    // Widget components own their value shape and validation. A pure marker
    // widget (no bound path) owns no form value, so it must not contribute a
    // schema entry — its synthetic id would otherwise nest into a required
    // parent object and wrongly invalidate the form.
    if (isWidgetComponent(component)) {
      if (getComponentSourcePath(component)) {
        schemaShape[fieldId] = z.any().optional();
      }
      return;
    }

    const baseSchema = ZOD_SCHEMA_MAP[component.uiType] ?? z.any();

    // Resolve mode-aware validation to flat validation for current mode
    const resolvedValidation = resolveValidationForMode(
      component.validation,
      formMode
    );

    let fieldSchema: z.ZodTypeAny;

    // Apply validation rules if present
    if (resolvedValidation) {
      // Create a component with resolved (flat) validation for the downstream pipeline
      const resolvedComponent = {
        ...component,
        validation: resolvedValidation,
      } as unknown as Component;
      const { fieldSchema: validatedSchema, celData } =
        applyValidationFromSchema(resolvedComponent, baseSchema, fieldId);
      fieldSchema = validatedSchema;

      // Collect CEL validation data
      if (celData.celExpValidation) {
        celExpValidations.push({
          ...celData.celExpValidation,
          activeWhen: activeSwitch,
        });
      }
      if (celData.celDependencyGroup) {
        celDependencyGroups.push(celData.celDependencyGroup);
      }
    } else {
      // No validation means not required, same as a validated non-required field.
      fieldSchema = baseSchema.optional();
    }

    if (activeSwitch) {
      toggleableFieldRules.push({
        switchName: activeSwitch,
        fieldId,
        schema: fieldSchema,
      });
      // Still transforms valid values, but never fails on its own.
      schemaShape[fieldId] = fieldSchema.or(z.any());
      return;
    }

    schemaShape[fieldId] = fieldSchema;
  });

  return {
    schemaShape,
    celExpValidations,
    celDependencyGroups,
    toggleableFieldRules,
  };
};
