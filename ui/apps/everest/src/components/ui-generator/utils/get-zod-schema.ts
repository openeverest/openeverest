import { z } from 'zod';
import {
  Component,
  ComponentGroup,
  TopologyUISchemas,
  CelExpression,
  Topology,
} from 'components/ui-generator/ui-generator.types';
import { ZOD_SCHEMA_MAP, zodRuleMap } from 'components/ui-generator/constants';
import { generateFieldId } from 'components/ui-generator/utils/render-component';
import { extractCelFieldPaths, validateCelExpression } from './cel-validation';

export const buildZodSchema = (
  schema: TopologyUISchemas,
  selectedTopology: string
): { schema: z.ZodTypeAny; celDependencyGroups: string[][] } => {
  const celExpValidations: {
    path: string[];
    celExpressions: CelExpression[];
  }[] = [];
  const celDependencyGroups: string[][] = [];

  const buildShapeFromComponents = (
    components: { [key: string]: Component | ComponentGroup },
    basePath: string = ''
  ): Record<string, z.ZodTypeAny> => {
    const schemaShape: Record<string, z.ZodTypeAny> = {};

    Object.entries(components).forEach(([key, item]) => {
      const generatedName = basePath ? `${basePath}.${key}` : key;
      const fieldId = generateFieldId(item, generatedName);

      let fieldSchema: z.ZodTypeAny;

      // Handle groups recursively
      if (item.uiType === 'group' && 'components' in item) {
        const groupSchema = buildShapeFromComponents(
          (item as ComponentGroup).components,
          generatedName
        );
        // For groups, merge nested schemas into parent level (flat structure)
        Object.assign(schemaShape, groupSchema);
        return;
      }

      // Get base Zod schema for this UI type
      const component = item as Component;
      let baseSchema = ZOD_SCHEMA_MAP[component.uiType] ?? z.any();

      // Apply validation rules if present in schema
      if ('validation' in component && component.validation) {
        // For number types, we need to build the final schema with validation
        if (component.uiType === 'number') {
          let numberSchema = z.coerce.number({
            invalid_type_error: 'Please enter a valid number',
          });

          Object.entries(component.validation).forEach(([rule, ruleValue]) => {
            if (rule === 'celExpressions') return; // Handle CEL separately

            const zodMethod = zodRuleMap[rule];
            if (
              zodMethod &&
              typeof numberSchema[zodMethod as keyof typeof numberSchema] ===
                'function'
            ) {
              numberSchema = (numberSchema as any)[zodMethod](ruleValue);
            }
          });

          // Wrap with union and pipe for proper coercion
          fieldSchema = z
            .union([z.string().min(1), z.number()])
            .pipe(numberSchema);
        } else {
          // For non-number types, apply validation rules directly
          fieldSchema = baseSchema;
          Object.entries(component.validation).forEach(([rule, ruleValue]) => {
            if (rule === 'celExpressions') return; // Handle CEL separately

            const zodMethod = zodRuleMap[rule];
            if (
              zodMethod &&
              typeof fieldSchema[zodMethod as keyof typeof fieldSchema] ===
                'function'
            ) {
              fieldSchema = (fieldSchema as any)[zodMethod](ruleValue);
            }
          });
        }

        // Handle CEL expressions for cross-field validation
        if (
          'celExpressions' in component.validation &&
          component.validation.celExpressions
        ) {
          const celExpressions = component.validation.celExpressions;

          // Extract all field dependencies from all CEL expressions
          const allDeps = new Set<string>();
          celExpressions.forEach((celExpr) => {
            const deps = extractCelFieldPaths(celExpr.celExpr);
            deps.forEach((dep) => allDeps.add(dep.join('.')));
          });

          // Add to dependency groups for re-validation triggers
          if (allDeps.size > 0) {
            celDependencyGroups.push([fieldId, ...Array.from(allDeps)]);
          }

          // Store for superRefine validation
          celExpValidations.push({
            path: [fieldId],
            celExpressions,
          });
        }
      } else {
        // No validation rules - use base schema as-is
        fieldSchema = baseSchema;
      }

      schemaShape[fieldId] = fieldSchema;
    });

    return schemaShape;
  };

  const convertToNestedSchema = (
    flatSchema: Record<string, z.ZodTypeAny>
  ): Record<string, z.ZodTypeAny> => {
    const nested: Record<string, any> = {};

    Object.entries(flatSchema).forEach(([path, zodType]) => {
      const keys = path.split('.');
      let current = nested;

      keys.forEach((key, index) => {
        if (index === keys.length - 1) {
          // Last key - set the Zod type
          current[key] = zodType;
        } else {
          // Intermediate key - ensure object exists
          if (!current[key]) {
            current[key] = {};
          }
          current = current[key];
        }
      });
    });

    // Convert nested objects to z.object() schemas recursively
    const convertToZod = (obj: any): z.ZodTypeAny => {
      const shape: Record<string, z.ZodTypeAny> = {};

      Object.entries(obj).forEach(([key, value]) => {
        if (
          value &&
          typeof value === 'object' &&
          !(value instanceof z.ZodType)
        ) {
          // Nested object - convert recursively
          shape[key] = convertToZod(value);
        } else {
          // Zod type - use as-is
          shape[key] = value as z.ZodTypeAny;
        }
      });

      return z.object(shape);
    };

    return Object.fromEntries(
      Object.entries(nested).map(([key, value]) => [
        key,
        typeof value === 'object' && !(value instanceof z.ZodType)
          ? convertToZod(value)
          : value,
      ])
    );
  };

  // for selected topology only
  const buildCompleteSchema = (): z.ZodTypeAny => {
    const topology: Topology = schema[selectedTopology];

    if (!topology || !topology.sections) {
      console.warn(`No topology found for key: ${selectedTopology}`);
      return z.object({});
    }

    let flatFields: Record<string, z.ZodTypeAny> = {};

    // TODO typescript error
    Object.entries(topology.sections).forEach(([sectionKey, section]) => {
      if (section?.components) {
        const sectionSchema = buildShapeFromComponents(
          section.components,
          sectionKey
        );
        Object.assign(flatFields, sectionSchema);
      }
    });

    // Convert flat schema to nested structure
    const nestedFields = convertToNestedSchema(flatFields);
    return z.object(nestedFields).passthrough();
  };

  let zodSchema = buildCompleteSchema();

  if (celExpValidations.length > 0) {
    zodSchema = zodSchema.superRefine((data, ctx) => {
      celExpValidations.forEach(({ path, celExpressions }) => {
        const fieldPath = path.join('.');

        // Evaluate each CEL expression for this field
        celExpressions.forEach((celExpr) => {
          const validationResult = validateCelExpression(celExpr, data);

          if (!validationResult.isValid) {
            ctx.addIssue({
              code: z.ZodIssueCode.custom,
              message: validationResult.message || 'Validation failed',
              path: path,
            });
          }
        });
      });
    });
  }

  return {
    schema: zodSchema,
    celDependencyGroups,
  };
};
