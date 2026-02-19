import type {
  Component,
  ComponentGroup,
} from 'components/ui-generator/ui-generator.types';

export const orderComponents = (
  components: { [key: string]: Component | ComponentGroup },
  componentsOrder?: string[]
): [string, Component | ComponentGroup][] => {
  const entries = Object.entries(components);

  if (!componentsOrder || componentsOrder.length === 0) {
    return entries;
  }

  const componentMap = new Map(entries);
  const orderedEntries: [string, Component | ComponentGroup][] = [];
  const unorderedKeys = new Set(entries.map(([key]) => key));

  // Add components in specified order
  componentsOrder.forEach((key) => {
    if (componentMap.has(key)) {
      orderedEntries.push([key, componentMap.get(key)!]);
      unorderedKeys.delete(key);
    }
  });

  // Add remaining components not in order array
  unorderedKeys.forEach((key) => {
    orderedEntries.push([key, componentMap.get(key)!]);
  });

  return orderedEntries;
};
