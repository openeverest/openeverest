import {
  Topology,
  TopologyUISchemas,
} from 'components/ui-generator/ui-generator.types';
import { buildDefaultsFromComponents } from './build-defaults-from-components';
import { convertToNestedObject } from './convert-to-nested-object';

// Processes all sections and converts flat defaults to nested object structure.
export const getDefaultValues = (
  schema: TopologyUISchemas,
  selectedTopology: string
): Record<string, unknown> => {
  const topology: Topology = schema[selectedTopology];

  if (!topology || !topology.sections) {
    return {};
  }

  const flatDefaults: Record<string, unknown> = {};

  // Build defaults from all sections
  Object.entries(topology.sections).forEach(([sectionKey, section]) => {
    if (section?.components) {
      const sectionDefaults = buildDefaultsFromComponents(
        section.components,
        sectionKey
      );
      Object.assign(flatDefaults, sectionDefaults);
    }
  });

  return convertToNestedObject(flatDefaults);
};
