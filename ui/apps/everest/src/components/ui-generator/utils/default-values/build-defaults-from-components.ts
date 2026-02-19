import {
  Component,
  ComponentGroup,
} from 'components/ui-generator/ui-generator.types';
import { generateFieldId } from '../component-renderer/generate-field-id';
import { UI_TYPE_DEFAULT_VALUE } from 'components/ui-generator/constants';

// Recursively builds default values for form fields using unique field IDs
export const buildDefaultsFromComponents = (
  components: { [key: string]: Component | ComponentGroup },
  basePath: string = ''
): Record<string, unknown> => {
  const result: Record<string, unknown> = {};

  Object.entries(components).forEach(([key, item]) => {
    const generatedName = basePath ? `${basePath}.${key}` : key;
    const fieldId = generateFieldId(item, generatedName);

    if (item.uiType === 'group' && 'components' in item) {
      // Recursively process nested components
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
    }
  });

  return result;
};
