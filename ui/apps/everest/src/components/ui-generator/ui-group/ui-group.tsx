import { Stack } from '@mui/material';
import {
  ComponentGroup,
  GroupType,
} from 'components/ui-generator/ui-generator.types';
import React from 'react';
import { componentGroupMap } from '../constants';

export type UIGroupProps = {
  children: React.ReactNode;
  groupType?: GroupType;
  groupParams?: Record<string, unknown>;
  item?: ComponentGroup;
};

const UIGroup = ({
  groupType,
  children,
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  groupParams: _groupParams,
  item,
}: UIGroupProps) => {
  const Component = groupType ? componentGroupMap[groupType] : undefined;

  return (
    <>
      {Component ? (
        React.createElement(Component, {
          children,
          label: item?.label,
        })
      ) : (
        <Stack spacing={2}>{children}</Stack>
      )}
    </>
  );
};

export default UIGroup;
