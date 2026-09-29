import { GroupWrapperProps } from 'components/ui-generator/ui-generator.types';

export type ToggleableWrapperProps = Pick<
  GroupWrapperProps,
  'children' | 'label' | 'description' | 'toggleable'
>;
