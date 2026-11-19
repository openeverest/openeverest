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
import { GroupDirection } from 'components/ui-generator/ui-generator.types';
import { StackWrapperProps } from './stack-wrapper.types';

// Lays out a group's direct children. Row keeps equal-width flex children
// (the former `groupType: line` behavior); column is the default stack.
const StackWrapper = ({
  children,
  direction = GroupDirection.Column,
  stackRef,
}: StackWrapperProps) => {
  if (direction === GroupDirection.Row) {
    return (
      <Stack
        ref={stackRef}
        direction="row"
        spacing={2}
        data-group-direction={GroupDirection.Row}
        sx={{
          alignItems: 'flex-start',
          width: '100%',
          flexWrap: 'nowrap',
          '> *': { flex: '1 1 0', minWidth: 0 },
        }}
      >
        {children}
      </Stack>
    );
  }

  return (
    <Stack
      ref={stackRef}
      spacing={2}
      data-group-direction={GroupDirection.Column}
    >
      {children}
    </Stack>
  );
};

export default StackWrapper;
