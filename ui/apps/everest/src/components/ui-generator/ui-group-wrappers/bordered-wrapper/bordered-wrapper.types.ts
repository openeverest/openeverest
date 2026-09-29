import { ReactNode, Ref } from 'react';
import { GroupWrapperProps } from 'components/ui-generator/ui-generator.types';

export type BorderedWrapperProps = Pick<
  GroupWrapperProps,
  'children' | 'label' | 'description'
> & {
  // Control shown at the heading's end (e.g. a toggleable group's switch).
  action?: ReactNode;
  bodyRef?: Ref<HTMLDivElement>;
};
