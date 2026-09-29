import { Stack } from '@mui/material';
import {
  ComponentGroup,
  GroupType,
} from 'components/ui-generator/ui-generator.types';
import React from 'react';
import { componentGroupMap } from '../constants';
import { getToggleableMeta } from '../utils/toggleable/toggleable';

export type UIGroupProps = {
  children: React.ReactNode;
  groupType?: GroupType;
  item?: ComponentGroup;
};

const UIGroup = ({ groupType, children, item }: UIGroupProps) => {
  const Component = groupType ? componentGroupMap[groupType] : undefined;

  return Component ? (
    <Component
      label={item?.label}
      description={item?.description}
      toggleable={item && getToggleableMeta(item)}
    >
      {children}
    </Component>
  ) : (
    <Stack spacing={2}>{children}</Stack>
  );
};

export default UIGroup;
