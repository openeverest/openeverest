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

import {
  ComponentGroup,
  GroupType,
} from 'components/ui-generator/ui-generator.types';
import React from 'react';
import { componentGroupMap } from '../constants';
import StackWrapper from '../ui-group-wrappers/stack-wrapper';
import { readGroupDirection } from '../utils/group-params/read-group-direction';
import { getToggleableMeta } from '../utils/toggleable/toggleable';

export type UIGroupProps = {
  children: React.ReactNode;
  groupType?: GroupType;
  item?: ComponentGroup;
};

const UIGroup = ({ groupType, children, item }: UIGroupProps) => {
  const Component = groupType ? componentGroupMap[groupType] : undefined;
  const direction = readGroupDirection(item?.groupParams);

  return Component ? (
    <Component
      label={item?.label}
      description={item?.description}
      groupParams={item?.groupParams}
      toggleable={item && getToggleableMeta(item)}
    >
      {children}
    </Component>
  ) : (
    <StackWrapper direction={direction}>{children}</StackWrapper>
  );
};

export default UIGroup;
