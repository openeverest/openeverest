import {
  Component,
  ComponentGroup,
  Topology,
  TopologyUISchemas,
} from 'components/ui-generator/ui-generator.types';
import { generateFieldId } from 'components/ui-generator/utils/render-component';
import { UI_TYPE_DEFAULT_VALUE } from 'components/ui-generator/constants';

export const getDefaultValues = (
  schema: TopologyUISchemas,
  selectedTopology: string
): Record<string, unknown> => {
  const topology: Topology = schema[selectedTopology];
  if (!topology || !topology.sections) {
    return {};
  }

  const flatDefaults: Record<string, unknown> = {};

  Object.entries(topology.sections).forEach(([sectionKey, section]) => {
    if (section?.components) {
      const basePath = sectionKey;
      const sectionDefaults = buildDefaultsFromComponents(
        section.components,
        basePath
      );
      Object.assign(flatDefaults, sectionDefaults);
    }
  });

  // Convert flat object with dot-notation keys to nested object
  const nestedDefaults = convertToNestedObject(flatDefaults);

  return nestedDefaults;
};

/*
 * Recursively builds default values for form fields using unique field IDs.
 * Uses the 'path' field if available, otherwise uses the generated name from object path.
 */
const buildDefaultsFromComponents = (
  components: { [key: string]: Component | ComponentGroup },
  basePath: string = ''
): Record<string, unknown> => {
  const result: Record<string, unknown> = {};

  Object.entries(components).forEach(([key, item]) => {
    const generatedName = basePath ? `${basePath}.${key}` : key;
    const fieldId = generateFieldId(item, generatedName);

    if (item.uiType === 'group' && 'components' in item) {
      // recursively process nested components
      const nestedDefaults = buildDefaultsFromComponents(
        (item as ComponentGroup).components,
        generatedName
      );
      Object.assign(result, nestedDefaults);
    } else {
      const component = item as Component;
      if (
        'fieldParams' in component &&
        component.fieldParams?.defaultValue !== undefined
      ) {
        result[fieldId] = component.fieldParams.defaultValue;
      } else {
        result[fieldId] = UI_TYPE_DEFAULT_VALUE[component.uiType] ?? '';
      }
      // TODO default values from templates?
      // TODO default values form external source?
    }
  });

  return result;
};

/* 
Converts a flat object with dot-notation keys to a nested object
needs for correct reading of Zod and CELL validation errors
{ "spec.replica.nodes": 3 } => { spec: { replica: { nodes: 3 } } }
 */
const convertToNestedObject = (
  flatObj: Record<string, unknown>
): Record<string, unknown> => {
  const result: Record<string, any> = {};

  Object.entries(flatObj).forEach(([path, value]) => {
    const keys = path.split('.');
    let current = result;

    keys.forEach((key, index) => {
      if (index === keys.length - 1) {
        // las key - set the value
        current[key] = value;
      } else {
        // intermediate key - ensure object exists
        if (!current[key] || typeof current[key] !== 'object') {
          current[key] = {};
        }
        current = current[key];
      }
    });
  });

  return result;
};
