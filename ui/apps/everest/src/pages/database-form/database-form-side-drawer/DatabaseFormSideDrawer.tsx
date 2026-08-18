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

import { useMemo } from 'react';
import { Divider, Drawer, Toolbar } from '@mui/material';
import { useActiveBreakpoint } from 'hooks/utils/useActiveBreakpoint';
import { DatabasePreview } from '../database-preview/database-preview';
import { DatabaseFormSideDrawerProps } from './DatabaseFormSideDrawer.types';

const DatabaseFormSideDrawer = ({
  activeStepId,
  handleSectionEdit,
  disabled,
  stepsWithErrors,
}: DatabaseFormSideDrawerProps) => {
  const { isDesktop } = useActiveBreakpoint();

  const PreviewContent = useMemo(
    () => (
      <DatabasePreview
        disabled={disabled}
        activeStepId={activeStepId}
        onSectionEdit={handleSectionEdit}
        stepsWithErrors={stepsWithErrors}
        sx={[
          {
            mt: 2,
          },
          !isDesktop && {
            padding: 0,
          },
        ]}
      />
    ),
    [disabled, activeStepId, handleSectionEdit, stepsWithErrors, isDesktop]
  );

  if (isDesktop) {
    return (
      <Drawer
        variant="permanent"
        anchor="right"
        sx={(theme) => ({
          // MuiDrawer-paper will take 25% of the whole screen because it has a "fixed" positioning
          // Hence, we must use vw here to have the same calculation
          // We subtract the padding
          width: `calc(25vw - ${theme.spacing(4)})`,
          flexShrink: 0,
          ml: 3,
          [`& .MuiDrawer-paper`]: {
            width: '25%',
            boxSizing: 'border-box',
          },
        })}
      >
        <Toolbar />
        {PreviewContent}
      </Drawer>
    );
  }

  return (
    <>
      <Divider
        orientation="horizontal"
        flexItem
        sx={(theme) => ({
          // This is a little tweak
          // We make the divider longer, adding the main padding value
          // Then, to make it begin before the main padding, we add a negative margin
          // This way, the divider will cross the whole section
          width: `calc(100% + ${theme.spacing(4 * 2)})`,
          ml: -4,
          mt: 6,
        })}
      />
      {PreviewContent}
    </>
  );
};

export default DatabaseFormSideDrawer;
