import { z } from 'zod';
import {
  Topology,
  TopologyUISchemas,
} from 'components/ui-generator/ui-generator.types';
import { buildShapeFromComponents } from './build-shape-from-components';
import { convertToNestedSchema } from './convert-to-nested-schema';
import { applyCelValidation } from './apply-cel-validation';

export const buildZodSchema = (
  schema: TopologyUISchemas,
  selectedTopology: string
): { schema: z.ZodTypeAny; celDependencyGroups: string[][] } => {
  const topology: Topology = schema[selectedTopology];

  if (!topology || !topology.sections) {
    console.warn(`No topology found for key: ${selectedTopology}`);
    return {
      schema: z.object({}).passthrough(),
      celDependencyGroups: [],
    };
  }

  let flatFields: Record<string, z.ZodTypeAny> = {};
  const allCelExpValidations: ReturnType<
    typeof buildShapeFromComponents
  >['celExpValidations'] = [];
  const allCelDependencyGroups: string[][] = [];

  // Build schema from all sections
  Object.entries(topology.sections).forEach(([sectionKey, section]) => {
    if (section?.components) {
      const result = buildShapeFromComponents(section.components, sectionKey);

      Object.assign(flatFields, result.schemaShape);
      allCelExpValidations.push(...result.celExpValidations);
      allCelDependencyGroups.push(...result.celDependencyGroups);
    }
  });

  // Convert flat schema to nested structure
  const nestedFields = convertToNestedSchema(flatFields);
  let zodSchema: z.ZodTypeAny = z.object(nestedFields).passthrough();

  // Apply CEL validation if needed
  zodSchema = applyCelValidation(zodSchema, allCelExpValidations);

  return {
    schema: zodSchema,
    celDependencyGroups: allCelDependencyGroups,
  };
};
