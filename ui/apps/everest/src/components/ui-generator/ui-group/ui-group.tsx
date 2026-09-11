// Copyright (C) 2026 The OpenEverest Contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import { Stack } from '@mui/material';
import {
  ComponentGroup,
  GroupType,
} from 'components/ui-generator/ui-generator.types';
import React from 'react';
import { componentGroupMap } from '../constants';

export type UIGroupProps = {
  children: React.ReactNode;
  fieldName?: string;
  groupType?: GroupType;
  groupParams?: ComponentGroup['groupParams'];
  item?: ComponentGroup;
};

const UIGroup = ({
  groupType,
  children,
  fieldName,
  groupParams,
  item,
}: UIGroupProps) => {
  const Component = groupType ? componentGroupMap[groupType] : undefined;

  return (
    <>
      {Component ? (
        React.createElement(Component, {
          children,
          ...groupParams,
          fieldName,
          label: item?.label,
          description: item?.description,
          disabled: item?._disabled,
        })
      ) : (
        <Stack spacing={2}>{children}</Stack>
      )}
    </>
  );
};

export default UIGroup;
