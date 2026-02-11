import { z } from 'zod';
import {
  Component,
  ComponentGroup,
  CelExpression,
} from 'components/ui-generator/ui-generator.types';
import { ZOD_SCHEMA_MAP } from 'components/ui-generator/constants';
import { generateFieldId } from '../component-renderer/generate-field-id';
import { applyValidationFromSchema } from './apply-from-schema';

export type ComponentSchemaResult = {
  schemaShape: Record<string, z.ZodTypeAny>;
  celExpValidations: { path: string[]; celExpressions: CelExpression[] }[];
  celDependencyGroups: string[][];
};

export const buildShapeFromComponents = (
  components: { [key: string]: Component | ComponentGroup },
  basePath: string = ''
): ComponentSchemaResult => {
  const schemaShape: Record<string, z.ZodTypeAny> = {};
  const celExpValidations: {
    path: string[];
    celExpressions: CelExpression[];
  }[] = [];
  const celDependencyGroups: string[][] = [];

  Object.entries(components).forEach(([key, item]) => {
    const generatedName = basePath ? `${basePath}.${key}` : key;
    const fieldId = generateFieldId(item, generatedName);

    // Handle groups recursively
    if (item.uiType === 'group' && 'components' in item) {
      const groupResult = buildShapeFromComponents(
        (item as ComponentGroup).components,
        generatedName
      );

      // Merge nested schemas into parent level (flat structure)
      Object.assign(schemaShape, groupResult.schemaShape);
      celExpValidations.push(...groupResult.celExpValidations);
      celDependencyGroups.push(...groupResult.celDependencyGroups);
      return;
    }

    // Get base Zod schema for this UI type
    const component = item as Component;
    const baseSchema = ZOD_SCHEMA_MAP[component.uiType] ?? z.any();

    let fieldSchema: z.ZodTypeAny;

    // Apply validation rules if present
    if ('validation' in component && component.validation) {
      const { fieldSchema: validatedSchema, celData } =
        applyValidationFromSchema(component, baseSchema, fieldId);
      fieldSchema = validatedSchema;

      // Collect CEL validation data
      if (celData.celExpValidation) {
        celExpValidations.push(celData.celExpValidation);
      }
      if (celData.celDependencyGroup) {
        celDependencyGroups.push(celData.celDependencyGroup);
      }
    } else {
      fieldSchema = baseSchema;
    }

    schemaShape[fieldId] = fieldSchema;
  });

  return {
    schemaShape,
    celExpValidations,
    celDependencyGroups,
  };
};
