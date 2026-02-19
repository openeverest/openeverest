import type {
  Component,
  ComponentGroup,
} from 'components/ui-generator/ui-generator.types';

// Uses the 'path' property if available, otherwise generates an ID from the component name
export const generateFieldId = (
  item: Component | ComponentGroup,
  generatedName: string
): string => {
  if ('path' in item && item.path && typeof item.path === 'string') {
    return item.path;
  }

  return `g-${generatedName}`;
};
