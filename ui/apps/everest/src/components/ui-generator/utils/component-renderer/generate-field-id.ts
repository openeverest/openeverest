import type {
  Component,
  ComponentGroup,
} from 'components/ui-generator/ui-generator.types';
import { getComponentSourcePath } from '../preprocess/normalized-component';

// Uses the 'path' property if available, otherwise generates an ID from the component name
export const generateFieldId = (
  item: Component | ComponentGroup,
  generatedName: string
): string => {
  const sourcePath = getComponentSourcePath(item);
  if (sourcePath) {
    return sourcePath;
  }

  return `g-${generatedName}`;
};
