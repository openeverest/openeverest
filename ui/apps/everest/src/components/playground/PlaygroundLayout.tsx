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

import { type ReactNode } from 'react';
import { Box, Divider, useTheme } from '@mui/material';

export interface PlaygroundLayoutProps {
  /** Left pane content (editor) */
  editor: ReactNode;
  /** Right pane content (preview) */
  preview: ReactNode;
}

/**
 * A simple split-pane layout for the playground PoC.
 *
 * Uses CSS flexbox rather than a third-party split-pane library
 * to avoid unnecessary dependencies. Both panes are equal width
 * with a visible divider.
 */
export const PlaygroundLayout = ({
  editor,
  preview,
}: PlaygroundLayoutProps) => {
  const theme = useTheme();

  return (
    <Box
      data-testid="playground-layout"
      sx={{
        display: 'flex',
        height: '100%',
        minHeight: 0,
        border: `1px solid ${theme.palette.divider}`,
        borderRadius: 1,
        overflow: 'hidden',
      }}
    >
      <Box
        sx={{
          flex: 1,
          display: 'flex',
          flexDirection: 'column',
          minWidth: 0,
          overflow: 'hidden',
        }}
        aria-label="Editor pane"
      >
        {editor}
      </Box>
      <Divider orientation="vertical" flexItem />
      <Box
        sx={{
          flex: 1,
          display: 'flex',
          flexDirection: 'column',
          minWidth: 0,
          overflow: 'hidden',
        }}
        aria-label="Preview pane"
      >
        {preview}
      </Box>
    </Box>
  );
};
